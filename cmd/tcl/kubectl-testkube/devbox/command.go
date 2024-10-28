// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gookit/color"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/cmd/tcl/kubectl-testkube/devbox/devutils"
	"github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/savioxavier/termlink"
)

const (
	InterceptorMainPath = "cmd/tcl/devbox-mutating-webhook/main.go"
	AgentMainPath       = "cmd/api-server/main.go"
	ToolkitMainPath     = "cmd/testworkflow-toolkit/main.go"
	InitProcessMainPath = "cmd/testworkflow-init/main.go"
)

func NewDevBoxCommand() *cobra.Command {
	var (
		rawDevboxName    string
		autoAccept       bool
		baseAgentImage   string
		baseInitImage    string
		baseToolkitImage string
		syncResources    []string
	)

	cmd := &cobra.Command{
		Use:     "devbox",
		Hidden:  true,
		Aliases: []string{"dev"},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, ctxCancel := context.WithCancel(context.Background())
			stopSignal := make(chan os.Signal, 1)
			signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-stopSignal
				ctxCancel()
			}()

			startTs := time.Now()

			// Find repository root
			rootDir := devutils.FindDirContaining(InterceptorMainPath, AgentMainPath, ToolkitMainPath, InitProcessMainPath)
			if rootDir == "" {
				ui.Fail(errors.New("testkube repository not found"))
			}

			// Connect to cluster
			cluster, err := devutils.NewCluster()
			if err != nil {
				ui.Fail(err)
			}

			// Connect to Testkube
			cfg, err := config.Load()
			if err != nil {
				pterm.Error.Printfln("Failed to load config file: %s", err.Error())
				return
			}
			cloud, err := devutils.NewCloud(cfg.CloudContext, cmd)
			if err != nil {
				pterm.Error.Printfln("Failed to connect to Cloud: %s", err.Error())
				return
			}

			// Detect obsolete devbox environments
			if obsolete := cloud.ListObsolete(); len(obsolete) > 0 {
				count := 0
				for _, env := range obsolete {
					err := cloud.DeleteEnvironment(env.Id)
					if err != nil {
						fmt.Printf("Failed to delete obsolete devbox environment (%s): %s\n", env.Name, err.Error())
						continue
					}
					cluster.Namespace(env.Name).Destroy()
					count++
				}
				fmt.Printf("Deleted %d/%d obsolete devbox environments\n", count, len(obsolete))
			}

			// Initialize bare cluster resources
			namespace := cluster.Namespace(fmt.Sprintf("devbox-%s", rawDevboxName))
			objectStoragePod := namespace.Pod("devbox-storage")
			interceptorPod := namespace.Pod("devbox-interceptor")
			agentPod := namespace.Pod("devbox-agent")

			// Initialize binaries
			interceptorBin := devutils.NewBinary(InterceptorMainPath, cluster.OperatingSystem(), cluster.Architecture())
			agentBin := devutils.NewBinary(AgentMainPath, cluster.OperatingSystem(), cluster.Architecture())
			toolkitBin := devutils.NewBinary(ToolkitMainPath, cluster.OperatingSystem(), cluster.Architecture())
			initProcessBin := devutils.NewBinary(InitProcessMainPath, cluster.OperatingSystem(), cluster.Architecture())

			// Initialize wrappers over cluster resources
			interceptor := devutils.NewInterceptor(interceptorPod, baseInitImage, baseToolkitImage, interceptorBin)
			agent := devutils.NewAgent(agentPod, cloud, baseAgentImage, baseInitImage, baseToolkitImage)
			objectStorage := devutils.NewObjectStorage(objectStoragePod)
			var env *client.Environment

			// Cleanup
			cleanupCh := make(chan struct{})
			var cleanupMu sync.Mutex
			cleanup := func() {
				cleanupMu.Lock()

				fmt.Println("Deleting namespace...")
				if err := namespace.Destroy(); err != nil {
					fmt.Println("Failed to destroy namespace:", err.Error())
				}
				if env != nil && env.Id != "" {
					fmt.Println("Deleting environment...")
					if err = cloud.DeleteEnvironment(env.Id); err != nil {
						fmt.Println("Failed to delete environment:", err.Error())
					}
				}
			}
			go func() {
				<-ctx.Done()
				cleanup()
				close(cleanupCh)
			}()

			fail := func(err error) {
				fmt.Println("Error:", err.Error())
				cleanup()
				os.Exit(1)
			}

			// Create environment in the Cloud
			fmt.Println("Creating environment in Cloud...")
			env, err = cloud.CreateEnvironment(namespace.Name())
			if err != nil {
				fail(errors.Wrap(err, "failed to create Cloud environment"))
			}

			// Create namespace
			fmt.Println("Creating namespace...")
			if err = namespace.Create(); err != nil {
				fail(errors.Wrap(err, "failed to create namespace"))
			}

			g, _ := errgroup.WithContext(ctx)
			objectStorageReadiness := make(chan struct{})

			// Deploy object storage
			g.Go(func() error {
				fmt.Println("[Object Storage] Creating...")
				if err = objectStorage.Create(ctx); err != nil {
					fail(errors.Wrap(err, "failed to create object storage"))
				}
				fmt.Println("[Object Storage] Waiting for readiness...")
				if err = objectStorage.WaitForReady(ctx); err != nil {
					fail(errors.Wrap(err, "failed to wait for readiness"))
				}
				fmt.Println("[Object Storage] Ready")
				close(objectStorageReadiness)
				return nil
			})

			// Deploying interceptor
			g.Go(func() error {
				fmt.Println("[Interceptor] Building...")
				its := time.Now()
				_, err := interceptorBin.Build(ctx)
				if err != nil {
					fmt.Printf("[Interceptor] Build failed in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("[Interceptor] Built in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				fmt.Println("[Interceptor] Deploying...")
				if err = interceptor.Create(ctx); err != nil {
					fail(errors.Wrap(err, "failed to create interceptor"))
				}
				fmt.Println("[Interceptor] Waiting for readiness...")
				if err = interceptor.WaitForReady(ctx); err != nil {
					fail(errors.Wrap(err, "failed to create interceptor"))
				}
				fmt.Println("[Interceptor] Enabling...")
				if err = interceptor.Enable(ctx); err != nil {
					fail(errors.Wrap(err, "failed to enable interceptor"))
				}
				fmt.Println("[Interceptor] Ready")
				return nil
			})

			// Deploying the Agent
			g.Go(func() error {
				fmt.Println("[Agent] Building...")
				its := time.Now()
				_, err := agentBin.Build(ctx)
				if err != nil {
					fmt.Printf("[Agent] Build failed in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("[Agent] Built in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				<-objectStorageReadiness
				fmt.Println("[Agent] Uploading...")
				its = time.Now()
				err = objectStorage.Upload(ctx, "bin/testkube-api-server", agentBin.Path(), agentBin.Hash())
				if err != nil {
					fmt.Printf("[Agent] Upload failed in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("[Agent] Uploaded in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				fmt.Println("[Agent] Deploying...")
				if err = agent.Create(ctx, env); err != nil {
					fail(errors.Wrap(err, "failed to create agent"))
				}
				fmt.Println("[Agent] Waiting for readiness...")
				if err = agent.WaitForReady(ctx); err != nil {
					fail(errors.Wrap(err, "failed to create agent"))
				}
				fmt.Println("[Agent] Ready...")
				return nil
			})

			// Building Toolkit
			g.Go(func() error {
				fmt.Println("[Toolkit] Building...")
				its := time.Now()
				_, err := toolkitBin.Build(ctx)
				if err != nil {
					fmt.Printf("[Toolkit] Build failed in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("[Toolkit] Built in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				<-objectStorageReadiness
				fmt.Println("[Toolkit] Uploading...")
				its = time.Now()
				err = objectStorage.Upload(ctx, "bin/toolkit", toolkitBin.Path(), toolkitBin.Hash())
				if err != nil {
					fmt.Printf("[Toolkit] Upload failed in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("[Toolkit] Uploaded in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return nil
			})

			// Building Init Process
			g.Go(func() error {
				fmt.Println("[Init Process] Building...")
				its := time.Now()
				_, err := initProcessBin.Build(ctx)
				if err != nil {
					fmt.Printf("[Init Process] Build failed in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("[Init Process] Built in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				<-objectStorageReadiness
				fmt.Println("[Init Process] Uploading...")
				its = time.Now()
				err = objectStorage.Upload(ctx, "bin/init", initProcessBin.Path(), initProcessBin.Hash())
				if err != nil {
					fmt.Printf("[Init Process] Upload failed in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("[Init Process] Uploaded in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return nil
			})

			g.Wait()

			// Live synchronisation
			fmt.Println("Creating file system watcher...")
			goWatcher, err := devutils.NewFsWatcher(rootDir)
			if err != nil {
				fail(errors.Wrap(err, "failed to watch Testkube repository"))
			}

			if len(syncResources) > 0 {
				fmt.Println("Loading Test Workflows and Templates...")
				sync := devutils.NewCRDSync()

				// Initial run
				for _, path := range syncResources {
					_ = sync.Load(path)
				}
				fmt.Printf("Started synchronising %d Test Workflows and %d Templates...\n", sync.WorkflowsCount(), sync.TemplatesCount())

				// Propagate changes from FS to CRDSync
				yamlWatcher, err := devutils.NewFsWatcher(syncResources...)
				if err != nil {
					fail(errors.Wrap(err, "failed to watch for YAML changes"))
				}
				go func() {
					for {
						if ctx.Err() != nil {
							break
						}
						file, err := yamlWatcher.Next(ctx)
						if !strings.HasSuffix(file, ".yml") && !strings.HasSuffix(file, ".yaml") {
							continue
						}
						if err == nil {
							_ = sync.Load(file)
						}
					}
				}()

				// Propagate changes from CRDSync to Cloud
				go func() {
					parallel := make(chan struct{}, 10)
					process := func(update *devutils.CRDSyncUpdate) {
						parallel <- struct{}{}
						switch update.Op {
						case devutils.CRDSyncUpdateOpCreate:
							client, err := cloud.Client(env.Id)
							if err != nil {
								fail(errors.Wrap(err, "failed to create cloud client"))
							}
							if update.Template != nil {
								update.Template.Spec.Events = nil // ignore Cronjobs
								_, err := client.CreateTestWorkflowTemplate(*testworkflows.MapTemplateKubeToAPI(update.Template))
								if err != nil {
									fmt.Printf("CRD Sync: creating template: %s: error: %s\n", update.Template.Name, err.Error())
								} else {
									fmt.Println("CRD Sync: created template:", update.Template.Name)
								}
							} else {
								update.Workflow.Spec.Events = nil // ignore Cronjobs
								_, err := client.CreateTestWorkflow(*testworkflows.MapKubeToAPI(update.Workflow))
								if err != nil {
									fmt.Printf("CRD Sync: creating workflow: %s: error: %s\n", update.Workflow.Name, err.Error())
								} else {
									fmt.Println("CRD Sync: created workflow:", update.Workflow.Name)
								}
							}
						case devutils.CRDSyncUpdateOpUpdate:
							client, err := cloud.Client(env.Id)
							if err != nil {
								fail(errors.Wrap(err, "failed to create cloud client"))
							}
							if update.Template != nil {
								update.Template.Spec.Events = nil // ignore Cronjobs
								_, err := client.UpdateTestWorkflowTemplate(*testworkflows.MapTemplateKubeToAPI(update.Template))
								if err != nil {
									fmt.Printf("CRD Sync: updating template: %s: error: %s\n", update.Template.Name, err.Error())
								} else {
									fmt.Println("CRD Sync: updated template:", update.Template.Name)
								}
							} else {
								update.Workflow.Spec.Events = nil
								_, err := client.UpdateTestWorkflow(*testworkflows.MapKubeToAPI(update.Workflow))
								if err != nil {
									fmt.Printf("CRD Sync: updating workflow: %s: error: %s\n", update.Workflow.Name, err.Error())
								} else {
									fmt.Println("CRD Sync: updated workflow:", update.Workflow.Name)
								}
							}
						case devutils.CRDSyncUpdateOpDelete:
							client, err := cloud.Client(env.Id)
							if err != nil {
								fail(errors.Wrap(err, "failed to create cloud client"))
							}
							if update.Template != nil {
								err := client.DeleteTestWorkflowTemplate(update.Template.Name)
								if err != nil {
									fmt.Printf("CRD Sync: deleting template: %s: error: %s\n", update.Template.Name, err.Error())
								} else {
									fmt.Println("CRD Sync: deleted template:", update.Template.Name)
								}
							} else {
								err := client.DeleteTestWorkflow(update.Workflow.Name)
								if err != nil {
									fmt.Printf("CRD Sync: deleting workflow: %s: error: %s\n", update.Workflow.Name, err.Error())
								} else {
									fmt.Println("CRD Sync: deleted workflow:", update.Workflow.Name)
								}
							}
						}
						<-parallel
					}
					for {
						if ctx.Err() != nil {
							break
						}
						update, err := sync.Next(ctx)
						if err != nil {
							continue
						}
						go process(update)
					}
				}()
			}

			fmt.Println("Waiting for file changes...")

			rebuild := func(ctx context.Context) {
				g, _ := errgroup.WithContext(ctx)
				ts := time.Now()
				fmt.Println("Rebuilding applications...")
				g.Go(func() error {
					its := time.Now()
					_, err := agentBin.Build(ctx)
					if err != nil {
						fmt.Printf("  Agent: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("  Agent: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					its = time.Now()
					err = objectStorage.Upload(ctx, "bin/testkube-api-server", agentBin.Path(), agentBin.Hash())
					if err != nil {
						fmt.Printf("  Agent: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("  Agent: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					// Restart only if it has changes - TODO: do in a nicer way
					if time.Since(its).Truncate(time.Millisecond).String() != "0s" {
						err := agentPod.Restart(ctx)
						if err == nil {
							fmt.Printf("  Agent: restarted. Waiting for readiness...\n")
							_ = agentPod.RefreshData(ctx)
							err = agentPod.WaitForReady(ctx)
							if ctx.Err() != nil {
								return nil
							}
							if err == nil {
								fmt.Printf("  Agent: ready again\n")
							} else {
								fail(errors.Wrap(err, "failed to wait for agent pod readiness"))
							}
						} else {
							fmt.Printf("  Agent: restart failed: %s\n", err.Error())
						}
					}
					return nil
				})
				g.Go(func() error {
					its := time.Now()
					_, err := toolkitBin.Build(ctx)
					if err != nil {
						fmt.Printf("  Toolkit: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("  Toolkit: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					its = time.Now()
					err = objectStorage.Upload(ctx, "bin/toolkit", toolkitBin.Path(), toolkitBin.Hash())
					if err != nil {
						fmt.Printf("  Toolkit: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("  Toolkit: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
					return nil
				})
				g.Go(func() error {
					its := time.Now()
					_, err := initProcessBin.Build(ctx)
					if err != nil {
						fmt.Printf("  Init Process: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("  Init Process: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					its = time.Now()
					err = objectStorage.Upload(ctx, "bin/init", initProcessBin.Path(), initProcessBin.Hash())
					if err != nil {
						fmt.Printf("  Init Process: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("  Init Process: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
					return nil
				})
				err = g.Wait()
				if ctx.Err() != nil {
					fmt.Println("Applications synchronised in", time.Since(ts))
				}
			}

			fmt.Printf(color.Green.Render("Development box is ready. Took %s\n"), time.Since(startTs))
			if termlink.SupportsHyperlinks() {
				fmt.Println("Dashboard:", termlink.Link(cloud.DashboardUrl(env.Slug, "dashboard/test-workflows"), cloud.DashboardUrl(env.Slug, "dashboard/test-workflows")))
			} else {
				fmt.Println("Dashboard:", cloud.DashboardUrl(env.Slug, "dashboard/test-workflows"))
			}

			rebuildCtx, rebuildCtxCancel := context.WithCancel(ctx)
			for {
				if ctx.Err() != nil {
					break
				}
				file, err := goWatcher.Next(ctx)
				if err != nil {
					fmt.Println("file system watcher error:", err.Error())
				} else if strings.HasSuffix(file, ".go") {
					relPath, _ := filepath.Rel(rootDir, file)
					if relPath == "" {
						relPath = file
					}
					fmt.Printf("%s changed\n", relPath)
					rebuildCtxCancel()
					rebuildCtx, rebuildCtxCancel = context.WithCancel(ctx)
					go rebuild(rebuildCtx)
				}
			}

			<-cleanupCh
		},
	}

	cmd.Flags().StringVarP(&rawDevboxName, "name", "n", fmt.Sprintf("%d", time.Now().UnixNano()), "devbox name")
	cmd.Flags().StringSliceVarP(&syncResources, "sync", "s", nil, "synchronise resources at paths")
	cmd.Flags().StringVar(&baseInitImage, "init-image", "kubeshop/testkube-tw-init:latest", "base init image")
	cmd.Flags().StringVar(&baseToolkitImage, "toolkit-image", "kubeshop/testkube-tw-toolkit:latest", "base toolkit image")
	cmd.Flags().StringVar(&baseAgentImage, "agent-image", "kubeshop/testkube-api-server:latest", "base agent image")
	cmd.Flags().BoolVarP(&autoAccept, "yes", "y", false, "auto accept without asking for confirmation")

	return cmd
}
