package diagnostics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Offline collection uses the local kubectl and kubeconfig, so it works when
// the Nudgebee server itself is down. The output has the same JSON shape as
// the server's pods and logs sections.

// Kubectl runs kubectl against one namespace.
type Kubectl struct {
	Context   string
	Namespace string
	// Run is the command runner; tests replace it.
	Run func(ctx context.Context, args ...string) ([]byte, error)
}

// NewKubectl returns a runner that shells out to the kubectl on PATH.
func NewKubectl(kubeContext, namespace string) *Kubectl {
	return &Kubectl{Context: kubeContext, Namespace: namespace, Run: runKubectl}
}

func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("kubectl %s: %s", args[0], Redact(msg))
	}
	return out, nil
}

func (k *Kubectl) run(ctx context.Context, args ...string) ([]byte, error) {
	full := []string{}
	full = append(full, args...)
	full = append(full, "-n", k.Namespace)
	if k.Context != "" {
		full = append(full, "--context", k.Context)
	}
	return k.Run(ctx, full...)
}

type ContainerStatus struct {
	Name                string     `json:"name"`
	Image               string     `json:"image"`
	Ready               bool       `json:"ready"`
	RestartCount        int32      `json:"restart_count"`
	State               string     `json:"state"`
	StateReason         string     `json:"state_reason,omitempty"`
	LastTerminatedAt    *time.Time `json:"last_terminated_at,omitempty"`
	LastTerminateReason string     `json:"last_terminate_reason,omitempty"`
	LastExitCode        *int32     `json:"last_exit_code,omitempty"`
}

type PodStatus struct {
	Name       string            `json:"name"`
	Phase      string            `json:"phase"`
	Node       string            `json:"node,omitempty"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	Containers []ContainerStatus `json:"containers"`
}

type EventRecord struct {
	Type     string    `json:"type"`
	Reason   string    `json:"reason"`
	Object   string    `json:"object"`
	Message  string    `json:"message"`
	Count    int32     `json:"count"`
	LastSeen time.Time `json:"last_seen"`
}

type PodsSection struct {
	Namespace  string        `json:"namespace"`
	Pods       []PodStatus   `json:"pods"`
	Events     []EventRecord `json:"events"`
	EventsNote string        `json:"events_note"`
}

type ContainerLog struct {
	Pod       string       `json:"pod"`
	Container string       `json:"container"`
	Current   *FilteredLog `json:"current,omitempty"`
	Previous  *FilteredLog `json:"previous,omitempty"`
	Error     string       `json:"error,omitempty"`
}

type LogsSection struct {
	WindowHours int            `json:"window_hours"`
	Containers  []ContainerLog `json:"containers"`
}

// The subset of `kubectl get pods -o json` the bundle reads.
type kubeState struct {
	Running *struct{} `json:"running"`
	Waiting *struct {
		Reason string `json:"reason"`
	} `json:"waiting"`
	Terminated *struct {
		Reason     string    `json:"reason"`
		ExitCode   int32     `json:"exitCode"`
		FinishedAt time.Time `json:"finishedAt"`
	} `json:"terminated"`
}

type kubePod struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		NodeName   string `json:"nodeName"`
		Containers []struct {
			Name  string `json:"name"`
			Image string `json:"image"`
		} `json:"containers"`
	} `json:"spec"`
	Status struct {
		Phase             string     `json:"phase"`
		StartTime         *time.Time `json:"startTime"`
		ContainerStatuses []struct {
			Name                 string    `json:"name"`
			Ready                bool      `json:"ready"`
			RestartCount         int32     `json:"restartCount"`
			State                kubeState `json:"state"`
			LastTerminationState kubeState `json:"lastState"`
		} `json:"containerStatuses"`
	} `json:"status"`
}

type kubeEvent struct {
	Type           string    `json:"type"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Count          int32     `json:"count"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
	EventTime      time.Time `json:"eventTime"`
	InvolvedObject struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	} `json:"involvedObject"`
}

func stateOf(s kubeState) (string, string) {
	switch {
	case s.Running != nil:
		return "running", ""
	case s.Waiting != nil:
		return "waiting", s.Waiting.Reason
	case s.Terminated != nil:
		return "terminated", s.Terminated.Reason
	}
	return "unknown", ""
}

// CollectPods lists pods and events. The returned pods feed CollectLogs.
func (k *Kubectl) CollectPods(ctx context.Context, maxEvents int) (*PodsSection, []PodStatus, error) {
	raw, err := k.run(ctx, "get", "pods", "-o", "json")
	if err != nil {
		return nil, nil, err
	}
	var list struct {
		Items []kubePod `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, nil, fmt.Errorf("parse pods: %w", err)
	}
	out := &PodsSection{Namespace: k.Namespace, Pods: []PodStatus{}, Events: []EventRecord{},
		EventsNote: "Kubernetes keeps events for about one hour by default; older causes are not here."}
	for _, p := range list.Items {
		images := map[string]string{}
		for _, c := range p.Spec.Containers {
			images[c.Name] = c.Image
		}
		ps := PodStatus{Name: p.Metadata.Name, Phase: p.Status.Phase, Node: p.Spec.NodeName, StartedAt: p.Status.StartTime, Containers: []ContainerStatus{}}
		for _, cs := range p.Status.ContainerStatuses {
			state, reason := stateOf(cs.State)
			c := ContainerStatus{Name: cs.Name, Image: images[cs.Name], Ready: cs.Ready, RestartCount: cs.RestartCount, State: state, StateReason: reason}
			if t := cs.LastTerminationState.Terminated; t != nil {
				at, code := t.FinishedAt, t.ExitCode
				c.LastTerminatedAt, c.LastTerminateReason, c.LastExitCode = &at, t.Reason, &code
			}
			ps.Containers = append(ps.Containers, c)
		}
		// Containers without a status yet (pending pod) still get their logs tried.
		if len(ps.Containers) == 0 {
			for _, c := range p.Spec.Containers {
				ps.Containers = append(ps.Containers, ContainerStatus{Name: c.Name, Image: c.Image, State: "unknown"})
			}
		}
		out.Pods = append(out.Pods, ps)
	}
	sort.Slice(out.Pods, func(i, j int) bool { return out.Pods[i].Name < out.Pods[j].Name })

	rawEvents, err := k.run(ctx, "get", "events", "-o", "json")
	if err != nil {
		out.EventsNote = "events unavailable: " + err.Error()
		return out, out.Pods, nil
	}
	var events struct {
		Items []kubeEvent `json:"items"`
	}
	if err := json.Unmarshal(rawEvents, &events); err != nil {
		out.EventsNote = "events unavailable: " + err.Error()
		return out, out.Pods, nil
	}
	for _, e := range events.Items {
		last := e.LastTimestamp
		if last.IsZero() {
			last = e.EventTime
		}
		out.Events = append(out.Events, EventRecord{Type: e.Type, Reason: e.Reason, Object: e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name,
			Message: Redact(e.Message), Count: e.Count, LastSeen: last})
	}
	sort.Slice(out.Events, func(i, j int) bool { return out.Events[i].LastSeen.After(out.Events[j].LastSeen) })
	if len(out.Events) > maxEvents {
		out.Events = out.Events[:maxEvents]
	}
	return out, out.Pods, nil
}

// LogLimits mirror the server's defaults.
type LogLimits struct {
	Window      time.Duration
	MaxBytes    int64
	TailLines   int
	MaxProblems int
	Parallel    int
}

// maxLogLines bounds what kubectl returns per container.
const maxLogLines = 10000

func (k *Kubectl) readLog(ctx context.Context, pod, container string, previous bool, l LogLimits) (string, error) {
	// --limit-bytes would keep the oldest bytes of the window; the newest are
	// what matters, so read the last lines and trim from the front instead.
	args := []string{"logs", pod, "-c", container, "--tail=" + strconv.Itoa(maxLogLines)}
	if previous {
		args = append(args, "--previous")
	} else {
		args = append(args, "--since="+l.Window.String())
	}
	out, err := k.run(ctx, args...)
	return string(KeepNewest(out, l.MaxBytes)), err
}

// KeepNewest returns the last max bytes of b, starting at a line boundary.
func KeepNewest(b []byte, max int64) []byte {
	if int64(len(b)) <= max {
		return b
	}
	b = b[int64(len(b))-max:]
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		b = b[i+1:]
	}
	return b
}

// CollectLogs reads each container's recent log, and its previous instance's
// log when it restarted, keeping only problem lines and a short tail.
func (k *Kubectl) CollectLogs(ctx context.Context, pods []PodStatus, l LogLimits) *LogsSection {
	type job struct {
		pod, container string
		restarts       int32
	}
	var jobs []job
	for _, p := range pods {
		for _, c := range p.Containers {
			jobs = append(jobs, job{pod: p.Name, container: c.Name, restarts: c.RestartCount})
		}
	}
	results := make([]ContainerLog, len(jobs))
	sem := make(chan struct{}, l.Parallel)
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			cl := ContainerLog{Pod: j.pod, Container: j.container}
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				cl.Error = ctx.Err().Error()
				results[i] = cl
				return
			}
			raw, err := k.readLog(ctx, j.pod, j.container, false, l)
			if err != nil {
				cl.Error = err.Error()
			} else {
				f := FilterLog(raw, l.TailLines, l.MaxProblems)
				cl.Current = &f
			}
			if j.restarts > 0 {
				if prev, err := k.readLog(ctx, j.pod, j.container, true, l); err == nil {
					f := FilterLog(prev, l.TailLines, l.MaxProblems)
					cl.Previous = &f
				}
			}
			results[i] = cl
		}(i, j)
	}
	wg.Wait()
	return &LogsSection{WindowHours: int(l.Window.Hours()), Containers: results}
}
