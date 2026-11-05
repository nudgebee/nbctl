package diagnostics

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

const podsJSON = `{"items":[{
  "metadata":{"name":"services-server-1"},
  "spec":{"nodeName":"node-a","containers":[{"name":"api","image":"registry.example.com/api:1.2.3"}]},
  "status":{"phase":"Running","startTime":"2026-11-05T10:00:00Z","containerStatuses":[{
    "name":"api","ready":false,"restartCount":2,
    "state":{"waiting":{"reason":"CrashLoopBackOff"}},
    "lastState":{"terminated":{"reason":"OOMKilled","exitCode":137,"finishedAt":"2026-11-05T11:00:00Z"}}}]}}]}`

const eventsJSON = `{"items":[{"type":"Warning","reason":"BackOff","message":"restarting, token=abcdef123456","count":3,
  "lastTimestamp":"2026-11-05T11:01:00Z","involvedObject":{"kind":"Pod","name":"services-server-1"}}]}`

func fakeKubectl(t *testing.T, logs map[string]string) (*Kubectl, *[][]string) {
	var mu sync.Mutex
	var calls [][]string
	k := &Kubectl{Namespace: "nb", Context: "ctx-1", Run: func(_ context.Context, args ...string) ([]byte, error) {
		mu.Lock()
		calls = append(calls, args)
		mu.Unlock()
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "get pods"):
			return []byte(podsJSON), nil
		case strings.HasPrefix(joined, "get events"):
			return []byte(eventsJSON), nil
		case strings.HasPrefix(joined, "logs") && strings.Contains(joined, "--previous"):
			return []byte(logs["previous"]), nil
		case strings.HasPrefix(joined, "logs"):
			return []byte(logs["current"]), nil
		}
		t.Fatalf("unexpected kubectl call: %v", args)
		return nil, nil
	}}
	return k, &calls
}

func TestCollectPodsAndLogs(t *testing.T) {
	k, calls := fakeKubectl(t, map[string]string{
		"current":  "level=info msg=starting\n",
		"previous": "level=error msg=\"db down\" password=hunter2xyz\npanic: boom\n",
	})
	section, pods, err := k.CollectPods(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	c := section.Pods[0].Containers[0]
	if c.Image != "registry.example.com/api:1.2.3" || c.StateReason != "CrashLoopBackOff" || c.LastTerminateReason != "OOMKilled" || *c.LastExitCode != 137 {
		t.Errorf("container = %+v", c)
	}
	if section.Events[0].Message != "restarting, token=[REDACTED]" {
		t.Errorf("event message = %q", section.Events[0].Message)
	}

	logs := k.CollectLogs(context.Background(), pods, LogLimits{Window: 6 * time.Hour, MaxBytes: 1 << 20, TailLines: 10, MaxProblems: 10, Parallel: 2})
	cl := logs.Containers[0]
	if cl.Previous == nil || len(cl.Previous.Problems) != 2 {
		t.Fatalf("previous log = %+v", cl.Previous)
	}
	if strings.Contains(strings.Join(cl.Previous.Problems, "\n"), "hunter2xyz") {
		t.Errorf("secret leaked: %v", cl.Previous.Problems)
	}

	// Every call is scoped to the namespace and context.
	for _, call := range *calls {
		joined := strings.Join(call, " ")
		if !strings.Contains(joined, "-n nb") || !strings.Contains(joined, "--context ctx-1") {
			t.Errorf("call not scoped: %v", call)
		}
	}
}

func TestCollectPodsError(t *testing.T) {
	k := &Kubectl{Namespace: "nb", Run: func(context.Context, ...string) ([]byte, error) {
		return nil, errors.New("forbidden")
	}}
	if _, _, err := k.CollectPods(context.Background(), 10); err == nil {
		t.Fatal("expected error")
	}
}

func TestKeepNewest(t *testing.T) {
	b := []byte("old line 1\nold line 2\nnew line 3\n")
	if got := string(KeepNewest(b, 15)); got != "new line 3\n" {
		t.Errorf("got %q", got)
	}
	if got := string(KeepNewest(b, 1000)); got != string(b) {
		t.Errorf("got %q", got)
	}
}
