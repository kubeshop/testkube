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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/minio/minio-go/v7"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	common2 "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	workflows []testworkflowsv1.TestWorkflow
	templates []testworkflowsv1.TestWorkflowTemplate
)

func load(filePaths []string) (workflows []testworkflowsv1.TestWorkflow, templates []testworkflowsv1.TestWorkflowTemplate) {
	found := map[string]struct{}{}
	for _, filePath := range filePaths {
		err := filepath.Walk(filePath, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			// Ignore already registered file path
			if _, ok := found[path]; ok {
				return nil
			}
			// Ignore non-YAML files
			if !strings.HasSuffix(path, ".yml") && !strings.HasSuffix(path, ".yaml") {
				return nil
			}

			// Read the files
			found[path] = struct{}{}

			// Parse the YAML file
			file, err := os.Open(path)
			if err != nil {
				fmt.Printf(ui.Red("%s: failed to read: %s\n"), path, err.Error())
				return nil
			}

			decoder := yaml.NewDecoder(file)
			for {
				var obj map[string]interface{}
				err := decoder.Decode(&obj)
				if errors.Is(err, io.EOF) {
					file.Close()
					break
				}
				if err != nil {
					fmt.Printf(ui.Red("%s: failed to parse yaml: %s\n"), path, err.Error())
					break
				}

				if obj["kind"] == nil || !(obj["kind"].(string) == "TestWorkflow" || obj["kind"].(string) == "TestWorkflowTemplate") {
					continue
				}

				if obj["kind"].(string) == "TestWorkflow" {
					bytes, _ := yaml.Marshal(obj)
					tw := testworkflowsv1.TestWorkflow{}
					err := common.DeserializeCRD(&tw, bytes)
					if tw.Name == "" {
						continue
					}
					if err != nil {
						fmt.Printf(ui.Red("%s: failed to deserialize TestWorkflow: %s\n"), path, err.Error())
						continue
					}
					workflows = append(workflows, tw)
				} else if obj["kind"].(string) == "TestWorkflowTemplate" {
					bytes, _ := yaml.Marshal(obj)
					tw := testworkflowsv1.TestWorkflowTemplate{}
					err := common.DeserializeCRD(&tw, bytes)
					if tw.Name == "" {
						continue
					}
					if err != nil {
						fmt.Printf(ui.Red("%s: failed to deserialize TestWorkflowTemplate: %s\n"), path, err.Error())
						continue
					}
					templates = append(templates, tw)
				}
			}
			file.Close()
			return nil
		})
		ui.ExitOnError(fmt.Sprintf("Reading '%s'", filePath), err)
	}
	return
}

func NewDevBoxCommand() *cobra.Command {
	var (
		rawDevboxName    string
		autoAccept       bool
		baseAgentImage   string
		baseInitImage    string
		baseToolkitImage string
		syncResources    []string
	)

	ask := func(label string) bool {
		if autoAccept {
			return true
		}
		accept, _ := pterm.DefaultInteractiveConfirm.WithDefaultValue(true).Show(label)
		return accept
	}

	cmd := &cobra.Command{
		Use:     "devbox",
		Hidden:  true,
		Aliases: []string{"dev"},
		Run: func(cmd *cobra.Command, args []string) {
			devboxName := fmt.Sprintf("devbox-%s", rawDevboxName)

			// Load Testkube configuration
			cfg, err := config.Load()
			if err != nil {
				pterm.Error.Printfln("Failed to load config file: %s", err.Error())
				return
			}
			cloud, err := NewCloud(cfg.CloudContext)
			if err != nil {
				pterm.Error.Printfln("Failed to connect to Control Plane: %s", err.Error())
				return
			}

			// Print debug data for the Control Plane
			cloud.Debug()

			// Detect obsolete devbox environments
			if obsolete := cloud.ListObsolete(); len(obsolete) > 0 {
				if ask(fmt.Sprintf("Should delete %d obsolete devbox environments?", len(obsolete))) {
					count := 0
					for _, env := range obsolete {
						// TODO: Delete namespaces too
						err := cloud.DeleteEnvironment(env.Id)
						if err != nil {
							pterm.Error.Printfln("Failed to delete obsolete devbox environment (%s): %s", env.Name, err.Error())
						} else {
							count++
						}
					}
					pterm.Success.Printfln("Deleted %d/%d obsolete devbox environments", count, len(obsolete))
				}
			}

			// Verify if the User accepts this Kubernetes cluster
			if !ask("Should continue with that organization?") {
				return
			}

			// Connect to Kubernetes cluster
			cluster, err := NewCluster()
			if err != nil {
				pterm.Error.Printfln("Failed to connect to Kubernetes cluster: %s", err.Error())
				return
			}

			// Print debug data for the cluster
			cluster.Debug()

			// Verify if the User accepts this Kubernetes cluster
			if !ask("Should continue with that cluster?") {
				return
			}

			// Print devbox information
			PrintHeader("Development box")
			PrintItem("Name", devboxName, "")

			interceptorBinarySource := findFile("cmd/tcl/devbox-mutating-webhook/main.go")
			agentBinarySource := findFile("cmd/api-server/main.go")
			toolkitBinarySource := findFile("cmd/testworkflow-toolkit/main.go")
			initBinarySource := findFile("cmd/testworkflow-init/main.go")

			agentImageSource := findFile("build/api-server/Dockerfile")
			toolkitImageSource := findFile("build/testworkflow-toolkit/Dockerfile")
			initImageSource := findFile("build/testworkflow-init/Dockerfile")

			if interceptorBinarySource == "" {
				pterm.Error.Printfln("Pod Interceptor: source not found in the current tree.")
				return
			} else {
				PrintItem("Pod Interceptor", "build from source", filepath.Dir(interceptorBinarySource))
			}

			if agentBinarySource == "" || agentImageSource == "" {
				pterm.Error.Printfln("Agent: source not found in the current tree.")
				return
			} else {
				PrintItem("Agent", "build from source", filepath.Dir(agentBinarySource))
			}

			if initBinarySource == "" || initImageSource == "" {
				pterm.Error.Printfln("Init Process: source not found in the current tree.")
				return
			} else {
				PrintItem("Init Process", "build from source", filepath.Dir(initBinarySource))
			}
			if toolkitBinarySource == "" || toolkitImageSource == "" {
				pterm.Error.Printfln("Toolkit: source not found in the current tree.")
				return
			} else {
				PrintItem("Toolkit", "build from source", filepath.Dir(toolkitBinarySource))
			}

			// Create devbox environment
			if !ask("Continue creating devbox environment?") {
				return
			}

			// Configure access objects
			ns := cluster.Namespace(devboxName)
			storage := cluster.ObjectStorage(devboxName)
			interceptor := cluster.PodInterceptor(devboxName)
			agent := NewAgent(cluster.ClientSet(), devboxName, AgentConfig{
				AgentImage:   baseAgentImage,
				ToolkitImage: baseToolkitImage,
				InitImage:    baseInitImage,
			})

			// Destroying
			ctx, ctxCancel := context.WithCancel(context.Background())
			stopSignal := make(chan os.Signal, 1)
			signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-stopSignal
				ctxCancel()
			}()

			// Prepare spinners
			PrintActionHeader("Setting up...")
			endSpinner := PrintSpinner(
				"environment", "Creating environment",
				"namespace", "Configuring cluster namespace",
				"storage", "Deploying object storage",
				"storageReady", "Waiting for object storage readiness",
				"storageForwarding", "Forwarding object storage ports",
				"interceptor", "Building Pod interceptor",
				"interceptorDeploy", "Deploying Pod interceptor",
				"interceptorReady", "Waiting for Pod interceptor readiness",
				"interceptorEnable", "Enabling Pod interceptor",
				"agent", "Deploying Agent",
				"agentReady", "Waiting for Agent readiness",
			)

			// Create the environment in the organization
			env, err := cloud.CreateEnvironment(devboxName)
			if err != nil {
				endSpinner("environment", err)
				return
			}
			endSpinner("environment")

			// Create the namespace
			if err = ns.Create(); err != nil {
				endSpinner("namespace", err)
				return
			}
			endSpinner("namespace")

			// Deploy object storage
			if err = storage.Deploy(); err != nil {
				endSpinner("storage", err)
				return
			}
			endSpinner("storage")

			// Wait for object storage readiness
			if err = storage.WaitForReady(); err != nil {
				endSpinner("storageReady", err)
				return
			}
			endSpinner("storageReady")

			// Wait for object storage port forwarding
			if err = storage.Forward(); err != nil {
				endSpinner("storageForwarding", err)
				return
			}
			endSpinner("storageForwarding")

			// Building the Pod interceptor
			interceptorBinaryFilePath := "/tmp/devbox-pod-interceptor"
			interceptorBinary := NewBinary(
				interceptorBinarySource,
				interceptorBinaryFilePath,
				cluster.OperatingSystem(),
				cluster.Architecture(),
			)
			if _, err = interceptorBinary.Build(ctx); err != nil {
				endSpinner("interceptor", err)
				return
			}
			endSpinner("interceptor")

			// Deploying the Pod interceptor
			if err = interceptor.Deploy(interceptorBinaryFilePath, baseInitImage, baseToolkitImage); err != nil {
				endSpinner("interceptorDeploy", err)
				return
			}
			endSpinner("interceptorDeploy")

			// Wait for Pod interceptor readiness
			if err = interceptor.WaitForReady(); err != nil {
				endSpinner("interceptorReady", err)
				return
			}
			endSpinner("interceptorReady")

			// Enable Pod interceptor
			if err = interceptor.Enable(); err != nil {
				endSpinner("interceptorEnable", err)
				return
			}
			endSpinner("interceptorEnable")

			// Deploy Agent
			if err = agent.Deploy(*env, cloud); err != nil {
				endSpinner("agent", err)
				return
			}
			endSpinner("agent")

			// Wait for Agent readiness
			if err = agent.WaitForReady(); err != nil {
				endSpinner("agentReady", err)
				return
			}
			endSpinner("agentReady")

			PrintHeader("Environment")
			PrintItem("Environment ID", env.Id, "")
			PrintItem("Agent Token", env.AgentToken, "")
			PrintItem("Dashboard", cloud.DashboardUrl(env.Id, ""), "")

			//imageRegistry.Debug()
			storage.Debug()
			agent.Debug()

			// CONNECTING TO THE CLOUD
			agentBinaryFilePath := filepath.Join(filepath.Dir(agentImageSource), "testkube-api-server")
			agentBinary := NewBinary(
				agentBinarySource,
				agentBinaryFilePath,
				cluster.OperatingSystem(),
				cluster.Architecture(),
			)
			initBinaryFilePath := filepath.Join(filepath.Dir(initImageSource), "testworkflow-init")
			initBinary := NewBinary(
				initBinarySource,
				initBinaryFilePath,
				cluster.OperatingSystem(),
				cluster.Architecture(),
			)
			toolkitBinaryFilePath := filepath.Join(filepath.Dir(toolkitImageSource), "testworkflow-toolkit")
			toolkitBinary := NewBinary(
				toolkitBinarySource,
				toolkitBinaryFilePath,
				cluster.OperatingSystem(),
				cluster.Architecture(),
			)

			storageClient, err := storage.Connect()
			if err != nil {
				ui.Fail(fmt.Errorf("failed to connect to the Object Storage: %s", err))
			}
			storageClient.CreateBucket(ctx, "devbox")

			buildImages := func(ctx context.Context) (bool, error) {
				fmt.Println("Building...")
				var errsMu sync.Mutex
				errs := make([]error, 0)
				agentChanged := false
				initChanged := false
				toolkitChanged := false
				ts := time.Now()
				var wg sync.WaitGroup
				wg.Add(3)
				go func() {
					prevHash := agentBinary.Hash()
					hash, err := agentBinary.Build(ctx)
					if err != nil {
						errsMu.Lock()
						errs = append(errs, err)
						errsMu.Unlock()
					} else {
						if prevHash != hash {
							agentChanged = true
						}
					}
					wg.Done()
				}()
				go func() {
					prevHash := initBinary.Hash()
					hash, err := initBinary.Build(ctx)
					if err != nil {
						errsMu.Lock()
						errs = append(errs, err)
						errsMu.Unlock()
					} else {
						if prevHash != hash {
							initChanged = true
						}
					}
					wg.Done()
				}()
				go func() {
					prevHash := toolkitBinary.Hash()
					hash, err := toolkitBinary.Build(ctx)
					if err != nil {
						errsMu.Lock()
						errs = append(errs, err)
						errsMu.Unlock()
					} else {
						if prevHash != hash {
							toolkitChanged = true
						}
					}
					wg.Done()
				}()
				wg.Wait()

				if errors.Is(ctx.Err(), context.Canceled) {
					return false, context.Canceled
				}

				fmt.Println("Built binaries in", time.Since(ts))

				if len(errs) == 0 && ctx.Err() == nil && (initChanged || toolkitChanged || agentChanged) {
					fmt.Println("Packing...")
					ts = time.Now()
					count := 0
					if initChanged {
						count++
					}
					if toolkitChanged {
						count++
					}
					if agentChanged {
						count++
					}

					tarFile, err := os.Create("/tmp/devbox-binaries.tar.gz")
					if err != nil {
						return false, err
					}
					tarStream := artifacts.NewTarStream()
					var mu sync.Mutex
					go func() {
						mu.Lock()
						io.Copy(tarFile, tarStream)
						mu.Unlock()
					}()

					if initChanged {
						file, err := os.Open(initBinaryFilePath)
						if err != nil {
							return false, err
						}
						fileStat, err := file.Stat()
						if err != nil {
							file.Close()
							return false, err
						}
						tarStream.Add("testworkflow-init", file, fileStat)
						file.Close()
					}
					if toolkitChanged {
						file, err := os.Open(toolkitBinaryFilePath)
						if err != nil {
							return false, err
						}
						fileStat, err := file.Stat()
						if err != nil {
							file.Close()
							return false, err
						}
						tarStream.Add("testworkflow-toolkit", file, fileStat)
						file.Close()
					}
					if agentChanged {
						file, err := os.Open(agentBinaryFilePath)
						if err != nil {
							return false, err
						}
						fileStat, err := file.Stat()
						if err != nil {
							file.Close()
							return false, err
						}
						tarStream.Add("testkube-api-server", file, fileStat)
						file.Close()
					}

					tarStream.Close()
					mu.Lock()
					mu.Unlock()

					fmt.Printf("Packed %d binaries in %s\n", count, time.Since(ts))
					ts = time.Now()

					if ctx.Err() != nil {
						return false, nil
					}

					fmt.Println("Uploading...")
					tarFile, err = os.Open("/tmp/devbox-binaries.tar.gz")
					if err != nil {
						return false, err
					}
					defer tarFile.Close()
					tarFileStat, err := tarFile.Stat()
					if err != nil {
						return false, err
					}
					err = storageClient.SaveFileDirect(ctx, "binaries", "binaries.tar.gz", tarFile, tarFileStat.Size(), minio.PutObjectOptions{
						DisableMultipart: true,
						ContentEncoding:  "gzip",
						ContentType:      "application/gzip",
						UserMetadata: map[string]string{
							"X-Amz-Meta-Snowball-Auto-Extract": "true",
							"X-Amz-Meta-Minio-Snowball-Prefix": "binaries",
						},
					})
					os.Remove("/tmp/devbox-binaries.tar.gz")

					if count > 0 && ctx.Err() == nil {
						fmt.Printf("Uploaded %d binaries in %s\n", count, time.Since(ts))
					}
				}

				return initChanged || agentChanged || toolkitChanged, errors.Join(errs...)
			}

			buildImages(ctx)

			// Load Test Workflows from file system
			if len(syncResources) > 0 {
				workflows, templates = load(syncResources)
				fmt.Printf("found %d Test Workflows in file system (and %d templates)\n", len(workflows), len(templates))
			}

			// Inject Test Workflows from file system
			common2.GetClient(cmd) // refresh token
			cloudClient, err := client.GetClient(client.ClientCloud, client.Options{
				Insecure:           cloud.AgentInsecure(),
				ApiUri:             cloud.ApiURI(),
				CloudApiKey:        cloud.ApiKey(),
				CloudOrganization:  env.OrganizationId,
				CloudEnvironment:   env.Id,
				CloudApiPathPrefix: fmt.Sprintf("/organizations/%s/environments/%s/agent", env.OrganizationId, env.Id),
			})
			if err != nil {
				ui.Warn(fmt.Sprintf("failed to connect to cloud: %s", err.Error()))
			} else {
				var errs atomic.Int32
				queue := make(chan struct{}, 30)
				wg := sync.WaitGroup{}
				wg.Add(len(templates))
				for _, w := range templates {
					go func(w testworkflowsv1.TestWorkflowTemplate) {
						queue <- struct{}{}
						_, err = cloudClient.CreateTestWorkflowTemplate(testworkflows.MapTestWorkflowTemplateKubeToAPI(w))
						if err != nil {
							errs.Add(1)
							fmt.Printf("failed to create test workflow template: %s: %s\n", w.Name, err.Error())
						}
						<-queue
						wg.Done()
					}(w)
				}
				wg.Wait()
				fmt.Printf("Uploaded %d/%d templates.\n", len(templates)-int(errs.Load()), len(templates))
				errs.Swap(0)
				wg = sync.WaitGroup{}
				wg.Add(len(workflows))
				for _, w := range workflows {
					go func(w testworkflowsv1.TestWorkflow) {
						queue <- struct{}{}
						_, err = cloudClient.CreateTestWorkflow(testworkflows.MapTestWorkflowKubeToAPI(w))
						if err != nil {
							errs.Add(1)
							fmt.Printf("failed to create test workflow: %s: %s\n", w.Name, err.Error())
						}
						<-queue
						wg.Done()
					}(w)
				}
				wg.Wait()
				fmt.Printf("Uploaded %d/%d workflows.\n", len(workflows)-int(errs.Load()), len(workflows))
			}

			fsWatcher, err := fsnotify.NewWatcher()
			if err != nil {
				ui.Fail(err)
			}

			var watchFsRecursive func(dirPath string) error
			watchFsRecursive = func(dirPath string) error {
				if err := fsWatcher.Add(dirPath); err != nil {
					return err
				}
				return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
					if err != nil || !d.IsDir() {
						return nil
					}
					if filepath.Base(path)[0] == '.' {
						// Ignore dot-files
						return nil
					}
					if path == dirPath {
						return nil
					}
					return watchFsRecursive(path)
				})
			}
			go func() {
				triggerCtx, cancelTrigger := context.WithCancel(ctx)
				defer cancelTrigger()
				trigger := func(triggerCtx context.Context) {
					select {
					case <-triggerCtx.Done():
					case <-time.After(300 * time.Millisecond):
						changed, err := buildImages(triggerCtx)
						if ctx.Err() != nil {
							return
						}
						if err == nil {
							if changed {
								fmt.Println("Build finished. Changes detected")
							} else {
								fmt.Println("Build finished. No changes detected")
							}
						} else {
							fmt.Println("Build finished. Error:", err.Error())
						}
					}
				}
				for {
					select {
					case event, ok := <-fsWatcher.Events:
						if !ok {
							return
						}
						fileinfo, err := os.Stat(event.Name)
						if err != nil {
							continue
						}
						if fileinfo.IsDir() {
							if event.Has(fsnotify.Create) {
								if err = watchFsRecursive(event.Name); err != nil {
									fmt.Println("failed to watch", event.Name)
								}
							}
							continue
						}
						if !strings.HasSuffix(event.Name, ".go") {
							continue
						}
						if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) && !event.Has(fsnotify.Remove) {
							continue
						}
						fmt.Println("File changed:", event.Name)

						cancelTrigger()
						triggerCtx, cancelTrigger = context.WithCancel(ctx)
						go trigger(triggerCtx)
					case err, ok := <-fsWatcher.Errors:
						if !ok {
							return
						}
						fmt.Println("Filesystem watcher error:", err.Error())
					}
				}
			}()
			err = watchFsRecursive(filepath.Clean(toolkitImageSource + "/../../.."))
			if err != nil {
				ui.Fail(err)
			}
			defer fsWatcher.Close()
			fmt.Println("Watching", filepath.Clean(toolkitImageSource+"/../../.."), "for changes")

			<-ctx.Done()

			// DESTROYING

			PrintActionHeader("Cleaning up...")
			endSpinner = PrintSpinner(
				"namespace", "Deleting cluster namespace",
				"environment", "Deleting environment",
				"interceptor", "Deleting interceptor",
			)

			wg := sync.WaitGroup{}
			wg.Add(3)

			// Destroy the namespace
			go func() {
				defer wg.Done()
				if err = ns.Destroy(); err != nil {
					endSpinner("namespace", err)
				} else {
					endSpinner("namespace")
				}
			}()

			// Destroy the environment
			go func() {
				defer wg.Done()
				if err = cloud.DeleteEnvironment(env.Id); err != nil {
					endSpinner("environment", err)
				} else {
					endSpinner("environment")
				}
			}()

			// Destroy the interceptor
			go func() {
				defer wg.Done()
				if err = interceptor.Disable(); err != nil {
					endSpinner("interceptor", err)
				} else {
					endSpinner("interceptor")
				}
			}()

			wg.Wait()
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
