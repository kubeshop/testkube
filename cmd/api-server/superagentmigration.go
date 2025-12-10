package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/backoff"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	"github.com/kubeshop/testkube/pkg/cloud"
)

type superAgentMigrationLogger interface {
	Infow(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
}

type superAgentMigrationConfig struct {
	agentId                                          string
	proContextControlPlaneHasSourceOfTruthCapability bool
	proContextAgentIsSuperAgent                      bool
	forceSuperAgentMode                              bool
	terminationLogPath                               string
	namespace                                        string
}

type superAgentMigrationGRPCClient interface {
	MigrateSuperAgent(ctx context.Context, in *cloud.MigrateSuperAgentRequest, opts ...grpc.CallOption) (*cloud.MigrateSuperAgentResponse, error)
}

type superAgentMigrationKubernetesResourceLister interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

type superAgentMigrationSyncStore interface {
	UpdateOrCreateTestTrigger(context.Context, testtriggersv1.TestTrigger) error
	UpdateOrCreateTestWorkflow(context.Context, testworkflowsv1.TestWorkflow) error
	UpdateOrCreateTestWorkflowTemplate(context.Context, testworkflowsv1.TestWorkflowTemplate) error
	UpdateOrCreateWebhook(context.Context, executorv1.Webhook) error
	UpdateOrCreateWebhookTemplate(context.Context, executorv1.WebhookTemplate) error
}

// migrateSuperAgent checks to see whether the current agent is a super agent and runs migration if so.
// "Super" Agents are deprecated, instead they are being migrated to a more generic Agent with Capabilities.
// The migration can only occur if:
// - The Control Plane supports being a Source of Truth.
// - The current Agent is still considered to be a Super Agent by the Control Plane.
// - The current Agent is not being held back as a Super Agent by an override.
// If the migration completes successfully then this function will cause the entire program to exit.
// This function will block until the migration is successfully completed.
// Additionally in a specific circumstance this function will attempt to rollback the agent to a super
// agent, this rollback can only occur if:
// - The current Agent is not considered to be a Super Agent by the Control Plane.
// - The current Agent is being held back as a Super Agent by an override.
// As with forward migration this function will block until the rollback is successful and then will
// exit the entire program.
func migrateSuperAgent(ctx context.Context, log superAgentMigrationLogger, cfg superAgentMigrationConfig, grpcClient superAgentMigrationGRPCClient, kubeClient superAgentMigrationKubernetesResourceLister, syncStore superAgentMigrationSyncStore) {
	isOrWasSuperAgent := strings.HasPrefix(cfg.agentId, "tkcroot_")
	if !isOrWasSuperAgent || !cfg.proContextControlPlaneHasSourceOfTruthCapability {
		return
	}

	if cfg.proContextAgentIsSuperAgent && !cfg.forceSuperAgentMode {
		// If the sync store is a NoOpStore then TLS is not enabled and migration cannot progress.
		if _, ok := syncStore.(syncagent.NoOpStore); ok {
			log.Errorw("Unable to perform Super Agent migration when TLS is not configured. Please configure TLS and restart the Agent to perform migration.")
			// Currently do not enforce TLS (do not exit the program if TLS is not configured but migration is desired).
			// Once we want to force TLS and SuperAgent migration then this return can be removed and the logging
			// and exiting behaviour can be used instead.
			return
			// Attempt to write to the termination log to make cluster operators' lives easier when working out why
			// the Agent is dying. Errors here are ignored as this is a nice to have and we're about to die so there
			// isn't any relevant error handling to perform here.
			_ = os.WriteFile(cfg.terminationLogPath, []byte("Insecure TLS settings configured"), os.ModePerm) //nolint:govet // This code is unreachable on purpose to enable simpler migration to enforced TLS and migrations in the future
			os.Exit(1)                                                                                        //nolint:govet // This code is unreachable on purpose to enable simpler migration to enforced TLS and migrations in the future
		}
		b := backoff.New(0, 0)
		// The eventual migration call itself requires its own backoff as the other backoff is
		// regularly reset to avoid overloading other systems during errors preparing for the
		// final migration call.
		migrationBackoff := backoff.New(0, 0)
		// Migration should be attempted forever because we need to migrate at some point!
		for {
			// Snapshot all syncable resources.
			var (
				testTriggerList          = testtriggersv1.TestTriggerList{}
				testWorkflowList         = testworkflowsv1.TestWorkflowList{}
				testWorkflowTemplateList = testworkflowsv1.TestWorkflowTemplateList{}
				webhookList              = executorv1.WebhookList{}
				webhookTemplateList      = executorv1.WebhookTemplateList{}
			)
			// Any error here will result in lists being repopulated to ensure that snapshots are as close to a single point in time
			// as possible.
			// Also any error here will stall the migration until the error is resolved. I'm not expecting these function calls to error
			// unless there is some issue with the Kubernetes API or connection to the Kubernetes API, which shouldn't really be happening,
			// so this is a bit of overkill error handling but we must ensure that all resources are synchronised before the migration can
			// be finalised.
			for {
				if err := kubeClient.List(ctx, &testTriggerList, client.InNamespace(cfg.namespace)); err != nil {
					retryAfter := b.Duration()
					log.Errorw("error listing TestTriggers in Namespace, unable to migrate SuperAgent, will retry after backoff.",
						"namespace", cfg.namespace,
						"backoff", retryAfter,
						"error", err.Error())
					time.Sleep(retryAfter)
					continue
				}
				if err := kubeClient.List(ctx, &testWorkflowList, client.InNamespace(cfg.namespace)); err != nil {
					retryAfter := b.Duration()
					log.Errorw("error listing TestWorkflows in Namespace, unable to migrate SuperAgent, will retry after backoff.",
						"namespace", cfg.namespace,
						"backoff", retryAfter,
						"error", err.Error())
					time.Sleep(retryAfter)
					continue
				}
				if err := kubeClient.List(ctx, &testWorkflowTemplateList, client.InNamespace(cfg.namespace)); err != nil {
					retryAfter := b.Duration()
					log.Errorw("error listing TestWorkflowTemplates in Namespace, unable to migrate SuperAgent, will retry after backoff.",
						"namespace", cfg.namespace,
						"backoff", retryAfter,
						"error", err.Error())
					time.Sleep(retryAfter)
					continue
				}
				if err := kubeClient.List(ctx, &webhookList, client.InNamespace(cfg.namespace)); err != nil {
					retryAfter := b.Duration()
					log.Errorw("error listing Webhooks in Namespace, unable to migrate SuperAgent, will retry after backoff.",
						"namespace", cfg.namespace,
						"backoff", retryAfter,
						"error", err.Error())
					time.Sleep(retryAfter)
					continue
				}
				if err := kubeClient.List(ctx, &webhookTemplateList, client.InNamespace(cfg.namespace)); err != nil {
					retryAfter := b.Duration()
					log.Errorw("error listing WebhookTemplates in Namespace, unable to migrate SuperAgent, will retry after backoff.",
						"namespace", cfg.namespace,
						"backoff", retryAfter,
						"error", err.Error())
					time.Sleep(retryAfter)
					continue
				}
				break
			}
			b.Reset()

			// Sync resources to the Control Plane.
			// Any error here will result in the client call being retried forever until it succeeds, once we reach this point we must
			// ensure that resources are fully synchronised to the Control Plane before the migration finalisation can take place, otherwise
			// the Control Plane cannot be correctly called the Source of Truth.
			for _, t := range testTriggerList.Items {
				for {
					if err := syncStore.UpdateOrCreateTestTrigger(ctx, t); err != nil {
						retryAfter := b.Duration()
						log.Errorw("error updating or creating TestTrigger, unable to migrate SuperAgent, will retry after backoff.",
							"TestTrigger", t.Name,
							"backoff", retryAfter,
							"error", err.Error())
						time.Sleep(retryAfter)
						continue
					}
					break
				}
			}
			b.Reset()
			for _, t := range testWorkflowList.Items {
				for {
					if err := syncStore.UpdateOrCreateTestWorkflow(ctx, t); err != nil {
						retryAfter := b.Duration()
						log.Errorw("error updating or creating TestWorkflow, unable to migrate SuperAgent, will retry after backoff.",
							"TestWorkflow", t.Name,
							"backoff", retryAfter,
							"error", err.Error())
						time.Sleep(retryAfter)
						continue
					}
					break
				}
			}
			b.Reset()
			for _, t := range testWorkflowTemplateList.Items {
				for {
					if err := syncStore.UpdateOrCreateTestWorkflowTemplate(ctx, t); err != nil {
						retryAfter := b.Duration()
						log.Errorw("error updating or creating TestWorkflowTemplate, unable to migrate SuperAgent, will retry after backoff.",
							"TestWorkflowTemplate", t.Name,
							"backoff", retryAfter,
							"error", err.Error())
						time.Sleep(retryAfter)
						continue
					}
					break
				}
			}
			b.Reset()
			for _, t := range webhookList.Items {
				for {
					if err := syncStore.UpdateOrCreateWebhook(ctx, t); err != nil {
						retryAfter := b.Duration()
						log.Errorw("error updating or creating Webhook, unable to migrate SuperAgent, will retry after backoff.",
							"Webhook", t.Name,
							"backoff", retryAfter,
							"error", err.Error())
						time.Sleep(retryAfter)
						continue
					}
					break
				}
			}
			b.Reset()
			for _, t := range webhookTemplateList.Items {
				for {
					if err := syncStore.UpdateOrCreateWebhookTemplate(ctx, t); err != nil {
						retryAfter := b.Duration()
						log.Errorw("error updating or creating WebhookTemplate, unable to migrate SuperAgent, will retry after backoff.",
							"WebhookTemplate", t.Name,
							"backoff", retryAfter,
							"error", err.Error())
						time.Sleep(retryAfter)
						continue
					}
					break
				}
			}
			b.Reset()

			// Inform the Control Plane that we have synchronised and can now safely migrate.
			if _, err := grpcClient.MigrateSuperAgent(ctx, &cloud.MigrateSuperAgentRequest{}); err != nil { //nolint:staticcheck // Marked as deprecated so nobody else is tempted to use it.
				// On a failure log and retry with a backoff just in case.
				retryAfter := migrationBackoff.Duration()
				log.Errorw("Failed to migrate SuperAgent, will retry after backoff.",
					"backoff", retryAfter,
					"error", err)
				time.Sleep(retryAfter)
				continue
			}

			// Once everything has successfully migrated, die. The expectation is that the agent will be restarted
			// causing it to requery the ProContext resulting in the IsSuperAgent field now being set to "false"
			// resulting in the agent no longer operating as a Super Agent and instead being successfully migrated
			// to a regular agent with capabilities.
			log.Infow("migrated super agent successfully, agent will now restart in normal agent mode.")
			os.Exit(0)
		}
	} else if !cfg.proContextAgentIsSuperAgent && cfg.forceSuperAgentMode {
		log.Infow("Rolling back agent to super agent. This may result in data added to the Control Plane no longer being accessible!")
		// If the agent is not currently considered to be a super agent, but the user has requested
		// that super agent mode is forced, then rollback the super agent migration.
		b := backoff.New(0, 0)
		// As with forward migration, we need this migration to go through before anything else can
		// start up, so keep retrying until the migration is accepted.
		for {
			if _, err := grpcClient.MigrateSuperAgent(ctx, &cloud.MigrateSuperAgentRequest{Rollback: true}); err != nil { //nolint:staticcheck // Marked as deprecated so nobody else is tempted to use it.
				// On a failure log and retry with a backoff just in case.
				retryAfter := b.Duration()
				log.Errorw("Failed to rollback Agent to SuperAgent, will retry after backoff.",
					"backoff", retryAfter,
					"error", err)
				time.Sleep(retryAfter)
				continue
			}

			// Once everything has successfully migrated, die. The expectation is that the agent will be restarted
			// causing it to requery the ProContext resulting in the IsSuperAgent field now being set to "true"
			// resulting in the agent operating as a Super Agent.
			log.Infow("Rolled back super agent successfully, agent will now restart in super agent mode.")
			os.Exit(0)
		}
	}
}
