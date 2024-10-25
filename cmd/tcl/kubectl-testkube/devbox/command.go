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

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/cmd/tcl/kubectl-testkube/devbox/devutils"
	"github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
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

			// Build initial binaries
			g, _ := errgroup.WithContext(ctx)
			fmt.Println("Building initial binaries...")
			g.Go(func() error {
				its := time.Now()
				_, err := interceptorBin.Build(ctx)
				if err != nil {
					fmt.Printf("Interceptor: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("Interceptor: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return err
			})
			g.Go(func() error {
				its := time.Now()
				_, err := agentBin.Build(ctx)
				if err != nil {
					fmt.Printf("Agent: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("Agent: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return err
			})
			g.Go(func() error {
				its := time.Now()
				_, err := toolkitBin.Build(ctx)
				if err != nil {
					fmt.Printf("Toolkit: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("Toolkit: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return err
			})
			g.Go(func() error {
				its := time.Now()
				_, err := initProcessBin.Build(ctx)
				if err != nil {
					fmt.Printf("Init Process: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("Init Process: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return err
			})
			err = g.Wait()

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

			// Deploy object storage
			fmt.Println("Creating object storage...")
			if err = objectStorage.Create(ctx); err != nil {
				fail(errors.Wrap(err, "failed to create object storage"))
			}
			fmt.Println("Waiting for object storage readiness...")
			if err = objectStorage.WaitForReady(ctx); err != nil {
				fail(errors.Wrap(err, "failed to wait for readiness"))
			}

			// Deploying interceptor
			fmt.Println("Deploying interceptor...")
			if err = interceptor.Create(ctx); err != nil {
				fail(errors.Wrap(err, "failed to create interceptor"))
			}
			fmt.Println("Waiting for interceptor readiness...")
			if err = interceptor.WaitForReady(ctx); err != nil {
				fail(errors.Wrap(err, "failed to create interceptor"))
			}

			// Uploading binaries
			g, _ = errgroup.WithContext(ctx)
			fmt.Println("Uploading binaries...")
			g.Go(func() error {
				its := time.Now()
				file, err := os.Open(agentBin.Path())
				if err != nil {
					return err
				}
				defer file.Close()
				err = objectStorage.Upload(ctx, "bin/testkube-api-server", file, agentBin.Hash())
				if err != nil {
					fmt.Printf("Agent: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("Agent: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return err
			})
			g.Go(func() error {
				its := time.Now()
				file, err := os.Open(toolkitBin.Path())
				if err != nil {
					return err
				}
				defer file.Close()
				err = objectStorage.Upload(ctx, "bin/toolkit", file, toolkitBin.Hash())
				if err != nil {
					fmt.Printf("Toolkit: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("Toolkit: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return err
			})
			g.Go(func() error {
				its := time.Now()
				file, err := os.Open(initProcessBin.Path())
				if err != nil {
					return err
				}
				defer file.Close()
				err = objectStorage.Upload(ctx, "bin/init", file, initProcessBin.Hash())
				if err != nil {
					fmt.Printf("Init Process: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
				} else {
					fmt.Printf("Init Process: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
				}
				return err
			})
			err = g.Wait()

			// Enabling Pod interceptor
			fmt.Println("Enabling interceptor...")
			if err = interceptor.Enable(ctx); err != nil {
				fail(errors.Wrap(err, "failed to enable interceptor"))
			}

			// Deploying agent
			fmt.Println("Deploying agent...")
			if err = agent.Create(ctx, env); err != nil {
				fail(errors.Wrap(err, "failed to create agent"))
			}
			fmt.Println("Waiting for agent readiness...")
			if err = agent.WaitForReady(ctx); err != nil {
				fail(errors.Wrap(err, "failed to create agent"))
			}
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
						if err == nil {
							_ = sync.Load(file)
						}
					}
				}()

				// Propagate changes from CRDSync to Cloud
				go func() {
					parallel := make(chan struct{}, 30)
					for {
						if ctx.Err() != nil {
							break
						}
						update, err := sync.Next(ctx)
						if err != nil {
							continue
						}
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
									fmt.Printf("Failed to create Test Workflow Template: %s: %s\n", update.Template.Name, err.Error())
								}
							} else {
								update.Workflow.Spec.Events = nil // ignore Cronjobs
								_, err := client.CreateTestWorkflow(*testworkflows.MapKubeToAPI(update.Workflow))
								if err != nil {
									fmt.Printf("Failed to create Test Workflow: %s: %s\n", update.Workflow.Name, err.Error())
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
									fmt.Printf("Failed to update Test Workflow Template: %s: %s\n", update.Template.Name, err.Error())
								}
							} else {
								update.Workflow.Spec.Events = nil
								_, err := client.UpdateTestWorkflow(*testworkflows.MapKubeToAPI(update.Workflow))
								if err != nil {
									fmt.Printf("Failed to update Test Workflow: %s: %s\n", update.Workflow.Name, err.Error())
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
									fmt.Printf("Failed to delete Test Workflow Template: %s: %s\n", update.Template.Name, err.Error())
								}
							} else {
								err := client.DeleteTestWorkflow(update.Workflow.Name)
								if err != nil {
									fmt.Printf("Failed to delete Test Workflow: %s: %s\n", update.Workflow.Name, err.Error())
								}
							}
						}
						<-parallel
					}
				}()
			}

			fmt.Println("Waiting for file changes...")

			rebuild := func(ctx context.Context) {
				g, _ := errgroup.WithContext(ctx)
				fmt.Println("Rebuilding binaries...")
				g.Go(func() error {
					its := time.Now()
					_, err := agentBin.Build(ctx)
					if err != nil {
						fmt.Printf("Agent: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("Agent: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					its = time.Now()
					file, err := os.Open(agentBin.Path())
					if err != nil {
						return err
					}
					defer file.Close()
					err = objectStorage.Upload(ctx, "bin/testkube-api-server", file, agentBin.Hash())
					if err != nil {
						fmt.Printf("Agent: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("Agent: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					// TODO: Restart only if it has changes
					if time.Since(its).Truncate(time.Millisecond).String() != "0s" {
						err := agentPod.Restart(ctx)
						if err == nil {
							fmt.Printf("Agent: restarted. Waiting for readiness...\n")
							_ = agentPod.RefreshData(ctx)
							err = agentPod.WaitForReady(ctx)
							if ctx.Err() != nil {
								return nil
							}
							if err == nil {
								fmt.Printf("Agent: ready again\n")
							} else {
								fail(errors.Wrap(err, "failed to wait for agent pod readiness"))
							}
						} else {
							fmt.Printf("Agent: restart failed: %s\n", err.Error())
						}
					}
					return nil
				})
				g.Go(func() error {
					its := time.Now()
					_, err := toolkitBin.Build(ctx)
					if err != nil {
						fmt.Printf("Toolkit: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("Toolkit: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					its = time.Now()
					file, err := os.Open(toolkitBin.Path())
					if err != nil {
						return err
					}
					defer file.Close()
					err = objectStorage.Upload(ctx, "bin/toolkit", file, toolkitBin.Hash())
					if err != nil {
						fmt.Printf("Toolkit: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("Toolkit: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
					return nil
				})
				g.Go(func() error {
					its := time.Now()
					_, err := initProcessBin.Build(ctx)
					if err != nil {
						fmt.Printf("Init Process: build finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("Init Process: build finished in %s.\n", time.Since(its).Truncate(time.Millisecond))

					its = time.Now()
					file, err := os.Open(initProcessBin.Path())
					if err != nil {
						return err
					}
					defer file.Close()
					err = objectStorage.Upload(ctx, "bin/init", file, initProcessBin.Hash())
					if err != nil {
						fmt.Printf("Init Process: upload finished in %s. Error: %s\n", time.Since(its).Truncate(time.Millisecond), err)
						return err
					}
					fmt.Printf("Init Process: upload finished in %s.\n", time.Since(its).Truncate(time.Millisecond))
					return nil
				})
				err = g.Wait()
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
