package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

func isConnectedUsingDate(lastConnectedDateStr string) bool {
	if lastConnectedDateStr == "" {
		return false
	}
	lastConnectedDate, err := time.Parse(time.RFC3339Nano, lastConnectedDateStr)
	if err != nil {
		return false
	}
	return time.Since(lastConnectedDate).Hours() < 48
}

var accountsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a single account by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		req := graphql.NewRequest(`
			query($id: uuid!) {
				cloud_accounts_by_pk(id: $id) {
					id
					account_name
					account_type
					cloud_provider
					status
					account_number
					created_at
					updated_at
					synced_at
					agent_synced_at
					agents {
						last_connected_at
						status
						version
						connection_status
						k8s_provider
						k8s_version
					}
				}
			}
		`)

		req.Var("id", args[0])

		var respData struct {
			CloudAccount struct {
				ID            string `json:"id"`
				AccountName   string `json:"account_name"`
				AccountType   string `json:"account_type"`
				CloudProvider string `json:"cloud_provider"`
				Status        string `json:"status"`
				AccountNumber string `json:"account_number"`
				CreatedAt     string `json:"created_at"`
				UpdatedAt     string `json:"updated_at"`
				SyncedAt      string `json:"synced_at"`
				AgentSyncedAt string `json:"agent_synced_at"`
				Agents        []struct {
					LastConnectedAt  string          `json:"last_connected_at"`
					Status           string          `json:"status"`
					Version          string          `json:"version"`
					ConnectionStatus json.RawMessage `json:"connection_status"`
					K8sProvider      string          `json:"k8s_provider"`
					K8sVersion       string          `json:"k8s_version"`
				} `json:"agents"`
			} `json:"cloud_accounts_by_pk"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if respData.CloudAccount.ID == "" {
			fmt.Println("Account not found.")
			return nil
		}

		if respData.CloudAccount.AccountType == "kubernetes" {
			var agents []struct {
				Version         string
				Status          string
				LastConnectedAt string
				K8sProvider     string
				K8sVersion      string
			}

			var features struct {
				Relay          string
				Prometheus     string
				AlertManager   string
				Logs           string
				Traces         string
				OpenCost       string
				NodeAgent      string
				AgentNamespace string
			}

			var scheduledJobs []struct {
				ActionName        string
				JobStatus         string
				ExecutionCount    int
				LastExecutionTime string
				Cron              string
			}

			for _, agent := range respData.CloudAccount.Agents {
				agents = append(agents, struct {
					Version         string
					Status          string
					LastConnectedAt string
					K8sProvider     string
					K8sVersion      string
				}{
					Version:         agent.Version,
					Status:          agent.Status,
					LastConnectedAt: agent.LastConnectedAt,
					K8sProvider:     agent.K8sProvider,
					K8sVersion:      agent.K8sVersion,
				})

				var connectionStatus struct {
					RelayConnection        bool   `json:"relayConnection"`
					PrometheusConnection   bool   `json:"prometheusConnection"`
					AlertManagerConnection bool   `json:"alertManagerConnection"`
					LogsConnection         bool   `json:"logsConnection"`
					TracesEnabled          bool   `json:"tracesEnabled"`
					OpencostConnection     bool   `json:"opencostConnection"`
					NodeAgentConnection    bool   `json:"nodeAgentConnection"`
					InstallationNamespace  string `json:"installationNamespace"`
					ScheduleJobs           []struct {
						RunnableParams struct {
							ActionFuncName string `json:"action_func_name"`
						} `json:"runnable_params"`
						State struct {
							JobStatus       int   `json:"job_status"`
							ExecCount       int   `json:"exec_count"`
							LastExecTimeSec int64 `json:"last_exec_time_sec"`
						} `json:"state"`
						SchedulingParams struct {
							CronExpression string `json:"cron_expression"`
						} `json:"scheduling_params"`
					} `json:"schedule_jobs"`
				}

				if err := json.Unmarshal(agent.ConnectionStatus, &connectionStatus); err != nil {
					return err
				}

				features.Relay = "Disconnected"
				if connectionStatus.RelayConnection {
					features.Relay = "Connected"
				}
				features.Prometheus = "Disconnected"
				if connectionStatus.PrometheusConnection {
					features.Prometheus = "Connected"
				}
				features.AlertManager = "Disconnected"
				if connectionStatus.AlertManagerConnection {
					features.AlertManager = "Connected"
				}
				features.Logs = "Disconnected"
				if connectionStatus.LogsConnection {
					features.Logs = "Connected"
				}
				features.Traces = "Disconnected"
				if connectionStatus.TracesEnabled {
					features.Traces = "Connected"
				}
				features.OpenCost = "Disconnected"
				if connectionStatus.OpencostConnection {
					features.OpenCost = "Connected"
				}
				features.NodeAgent = "Disconnected"
				if connectionStatus.NodeAgentConnection {
					features.NodeAgent = "Connected"
				}
				features.AgentNamespace = connectionStatus.InstallationNamespace

				for _, job := range connectionStatus.ScheduleJobs {
					jobStatus := ""
					switch job.State.JobStatus {
					case 1:
						jobStatus = "New"
					case 2:
						jobStatus = "Running"
					default:
						jobStatus = "Done"
					}

					lastExecTime := "-"
					if job.State.LastExecTimeSec > 0 {
						lastExecTime = time.Unix(job.State.LastExecTimeSec, 0).String()
					}

					scheduledJobs = append(scheduledJobs, struct {
						ActionName        string
						JobStatus         string
						ExecutionCount    int
						LastExecutionTime string
						Cron              string
					}{
						ActionName:        job.RunnableParams.ActionFuncName,
						JobStatus:         jobStatus,
						ExecutionCount:    job.State.ExecCount,
						LastExecutionTime: lastExecTime,
						Cron:              job.SchedulingParams.CronExpression,
					})
				}
			}

			output := struct {
				ID            string
				AccountName   string
				AccountType   string
				CloudProvider string
				Status        string
				AccountNumber string
				CreatedAt     string
				UpdatedAt     string
				SyncedAt      string
				AgentSyncedAt string
				Agents        []struct {
					Version         string
					Status          string
					LastConnectedAt string
					K8sProvider     string
					K8sVersion      string
				}
				Features struct {
					Relay          string
					Prometheus     string
					AlertManager   string
					Logs           string
					Traces         string
					OpenCost       string
					NodeAgent      string
					AgentNamespace string
				}
				ScheduledJobs []struct {
					ActionName        string
					JobStatus         string
					ExecutionCount    int
					LastExecutionTime string
					Cron              string
				}
			}{
				ID:            respData.CloudAccount.ID,
				AccountName:   respData.CloudAccount.AccountName,
				AccountType:   respData.CloudAccount.AccountType,
				CloudProvider: respData.CloudAccount.CloudProvider,
				Status:        respData.CloudAccount.Status,
				AccountNumber: respData.CloudAccount.AccountNumber,
				CreatedAt:     respData.CloudAccount.CreatedAt,
				UpdatedAt:     respData.CloudAccount.UpdatedAt,
				SyncedAt:      respData.CloudAccount.SyncedAt,
				AgentSyncedAt: respData.CloudAccount.AgentSyncedAt,
				Agents:        agents,
				Features:      features,
				ScheduledJobs: scheduledJobs,
			}

			format.GetFormat().Print(output)
		} else {
			var cloudFeatures struct {
				Events          string
				Resources       string
				Recommendations string
				Spends          string
			}

			for _, agent := range respData.CloudAccount.Agents {
				var connectionStatus struct {
					Events struct {
						End string `json:"end"`
					} `json:"events"`
					Resources struct {
						UpdatedAt string `json:"updated_at"`
					} `json:"resources"`
					Recommendations struct {
						UpdatedAt string `json:"updated_at"`
					} `json:"recommendations"`
					Spends struct {
						UpdatedAt string `json:"updated_at"`
					} `json:"spends"`
				}

				if err := json.Unmarshal(agent.ConnectionStatus, &connectionStatus); err != nil {
					return err
				}

				cloudFeatures.Events = "Disconnected"
				if isConnectedUsingDate(connectionStatus.Events.End) {
					cloudFeatures.Events = "Connected"
				}
				cloudFeatures.Resources = "Disconnected"
				if isConnectedUsingDate(connectionStatus.Resources.UpdatedAt) {
					cloudFeatures.Resources = "Connected"
				}
				cloudFeatures.Recommendations = "Disconnected"
				if isConnectedUsingDate(connectionStatus.Recommendations.UpdatedAt) {
					cloudFeatures.Recommendations = "Connected"
				}
				cloudFeatures.Spends = "Disconnected"
				if isConnectedUsingDate(connectionStatus.Spends.UpdatedAt) {
					cloudFeatures.Spends = "Connected"
				}
			}

			output := struct {
				ID            string
				AccountName   string
				AccountType   string
				CloudProvider string
				Status        string
				AccountNumber string
				CreatedAt     string
				UpdatedAt     string
				SyncedAt      string
				AgentSyncedAt string
				CloudFeatures struct {
					Events          string
					Resources       string
					Recommendations string
					Spends          string
				}
			}{
				ID:            respData.CloudAccount.ID,
				AccountName:   respData.CloudAccount.AccountName,
				AccountType:   respData.CloudAccount.AccountType,
				CloudProvider: respData.CloudAccount.CloudProvider,
				Status:        respData.CloudAccount.Status,
				AccountNumber: respData.CloudAccount.AccountNumber,
				CreatedAt:     respData.CloudAccount.CreatedAt,
				UpdatedAt:     respData.CloudAccount.UpdatedAt,
				SyncedAt:      respData.CloudAccount.SyncedAt,
				AgentSyncedAt: respData.CloudAccount.AgentSyncedAt,
				CloudFeatures: cloudFeatures,
			}

			format.GetFormat().Print(output)
		}

		return nil
	},
}

func init() {
	accountsCmd.AddCommand(accountsGetCmd)
}
