// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
	"github.com/kubeshop/testkube/pkg/ui"
)

// ServiceInstance contains all configuration needed to deploy and monitor a service.
type ServiceInstance struct {
	Index          int64
	Name           string
	Description    string
	Timeout        *time.Duration
	RestartPolicy  corev1.RestartPolicy
	ReadinessProbe *corev1.Probe
	Spec           testworkflowsv1.TestWorkflowSpec
}

// ServiceState tracks IP assignment and description for service discovery.
type ServiceState struct {
	Ip          string `json:"ip"`
	Description string `json:"description"`
}

type ServiceStatus string

const (
	ServiceStatusQueued  ServiceStatus = "queued"
	ServiceStatusRunning ServiceStatus = "running"
	ServiceStatusReady   ServiceStatus = "passed"
	ServiceStatusFailed  ServiceStatus = "failed"
)

// ServiceInfo is serialized as JSON and sent via PrintOutput for UI updates.
type ServiceInfo struct {
	Group       string        `json:"group"`
	Index       int64         `json:"index"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Logs        string        `json:"logs,omitempty"`
	Status      ServiceStatus `json:"status,omitempty"`
	Done        bool          `json:"done,omitempty"`
}

func (s ServiceInfo) AsMap() (v map[string]interface{}) {
	serialized, _ := json.Marshal(s)
	_ = json.Unmarshal(serialized, &v)
	return
}

// ServiceExecutionResult contains the outcome of monitoring a single service.
type ServiceExecutionResult struct {
	Started bool
	Ready   bool
	Failed  bool
	Error   error
}

const (
	UnlimitedParallelism = math.MaxInt64

	// How long to wait for Kubernetes to report a container exit after startup.
	// The kubelet syncs pod status every 10s by default.
	serviceFailureCheckTimeout = 15 * time.Second
)

type ServicesDependencies struct {
	Config          *testworkflowconfig.InternalConfig
	BaseMachine     expressions.Machine
	TransferSrv     transfer.Server
	ExecutionWorker executionworkertypes.Worker
	Ref             string
	Namespace       string
}

// ServicesExecutor handles the full services lifecycle:
// parsing, preparation, execution, and result reporting.
type ServicesExecutor struct {
	groupRef      string
	base64Encoded bool
	deps          ServicesDependencies
}

func NewServicesExecutor(groupRef string, base64Encoded bool, deps ServicesDependencies) *ServicesExecutor {
	return &ServicesExecutor{
		groupRef:      groupRef,
		base64Encoded: base64Encoded,
		deps:          deps,
	}
}

func NewServicesCmd() *cobra.Command {
	var (
		groupRef      string
		base64Encoded bool
	)
	cmd := &cobra.Command{
		Use:   "services <ref>",
		Short: "Start accompanying service(s)",

		Run: func(cmd *cobra.Command, args []string) {
			// Initialize dependencies from singletons in the command handler only
			deps := ServicesDependencies{
				Config:          config.Config(),
				BaseMachine:     spawn.CreateBaseMachineWithoutEnv(),
				TransferSrv:     transfer.NewServer(constants.DefaultTransferDirPath, config.IP(), constants.DefaultTransferPort),
				ExecutionWorker: spawn.ExecutionWorker(),
				Ref:             config.Ref(),
				Namespace:       config.Namespace(),
			}

			executor := NewServicesExecutor(groupRef, base64Encoded, deps)
			if err := executor.Execute(args); err != nil {
				ui.Fail(err)
			}
		},
	}

	cmd.Flags().StringVarP(&groupRef, "group", "g", "", "services group reference")
	cmd.Flags().BoolVar(&base64Encoded, "base64", false, "input is base64 encoded")

	return cmd
}

// RunServicesWithOptions executes services with the provided configuration.
// This is the testable entry point for integration tests.
func RunServicesWithOptions(specContent string, cfg *config.ConfigV2, base64Encoded bool, groupRef string) error {
	internalCfg := cfg.Internal()

	deps := ServicesDependencies{
		Config:          internalCfg,
		BaseMachine:     spawn.CreateBaseMachineWithoutEnv(),
		TransferSrv:     transfer.NewServer(constants.DefaultTransferDirPath, cfg.IP(), constants.DefaultTransferPort),
		ExecutionWorker: spawn.ParallelExecutionWorker(cfg),
		Ref:             internalCfg.Resource.Id,
		Namespace:       cfg.Namespace(),
	}

	executor := NewServicesExecutor(groupRef, base64Encoded, deps)
	return executor.Execute([]string{specContent})
}

// Execute runs all services and returns an error if any fail.
func (e *ServicesExecutor) Execute(args []string) error {
	if e.groupRef == "" {
		return errors.New("missing required --group for starting the services")
	}

	services, err := e.parseServices(args)
	if err != nil {
		return err
	}

	e.initializeServiceDefaults(services)

	instances, namespaces, state, svcParams, err := e.prepareInstances(services)
	if err != nil {
		return err
	}

	e.notifyQueuedInstances(instances)

	if err := e.startTransferServer(); err != nil {
		return err
	}

	if len(instances) == 0 {
		fmt.Println("nothing to run")
		return nil
	}

	failed := e.runServices(instances, namespaces, state, svcParams)
	e.reportFinalState(state)

	if failed == 0 {
		fmt.Printf("Successfully started %d workers.\n", len(instances))
		return nil
	}
	fmt.Printf("Failed to start %d out of %d expected workers.\n", failed, len(instances))
	return fmt.Errorf("%d services failed to start", failed)
}

// parseServices supports two input formats:
//   - Base64 encoded JSON (--base64 flag): prevents premature expression resolution
//   - Raw JSON key=value pairs: legacy format for backward compatibility
func (e *ServicesExecutor) parseServices(args []string) (map[string]testworkflowsv1.ServiceSpec, error) {
	services := make(map[string]testworkflowsv1.ServiceSpec)

	if e.base64Encoded && len(args) > 0 {
		// The processor base64-encodes service specs to prevent testworkflow-init
		// from prematurely resolving expressions like {{ matrix.browser.driver }}.
		var servicesMap map[string]json.RawMessage
		if err := expressionstcl.DecodeBase64JSON(args[0], &servicesMap); err != nil {
			return nil, errors.Wrap(err, "decoding services")
		}
		for name, raw := range servicesMap {
			var svc testworkflowsv1.ServiceSpec
			if err := json.Unmarshal(raw, &svc); err != nil {
				return nil, errors.Wrapf(err, "parsing service spec for %s", name)
			}
			services[name] = svc
		}
	} else {
		// Legacy format: name=spec pairs (kept for backward compatibility)
		for i := range args {
			name, v, found := strings.Cut(args[i], "=")
			if !found {
				return nil, fmt.Errorf("invalid service declaration: %s", args[i])
			}
			var svc *testworkflowsv1.ServiceSpec
			if err := json.Unmarshal([]byte(v), &svc); err != nil {
				return nil, errors.Wrap(err, "parsing service spec")
			}
			services[name] = *svc
		}
	}

	return services, nil
}

// initializeServiceDefaults applies default values to service specs.
func (e *ServicesExecutor) initializeServiceDefaults(services map[string]testworkflowsv1.ServiceSpec) {
	for name := range services {
		if services[name].Pod == nil {
			svc := services[name]
			svc.Pod = &testworkflowsv1.PodConfig{}
			services[name] = svc
		}
		if services[name].Pod.ServiceAccountName == "" {
			services[name].Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
		}
		// Initialize empty array of details for each service
		instructions.PrintHintDetails(e.deps.Ref, data.ServicesPrefix+name, []ServiceState{})
	}
}

// prepareInstances analyzes services and creates instances for execution.
func (e *ServicesExecutor) prepareInstances(services map[string]testworkflowsv1.ServiceSpec) (
	[]ServiceInstance,
	[]string,
	map[string][]ServiceState,
	map[string]*commontcl.ParamsSpec,
	error,
) {
	state := make(map[string][]ServiceState)
	instances := make([]ServiceInstance, 0)
	namespaces := make([]string, 0)
	svcParams := make(map[string]*commontcl.ParamsSpec)

	for name, svc := range services {
		params, err := commontcl.GetParamsSpec(svc.Matrix, svc.Shards, svc.Count, svc.MaxCount, e.deps.BaseMachine)
		if err != nil {
			return nil, nil, nil, nil, errors.Wrapf(err, "%s: compute matrix and sharding", commontcl.ServiceLabel(name))
		}
		svcParams[name] = params

		if params.Count == 0 {
			fmt.Printf("%s: 0 instances requested (combinations=%d, count=%d), skipping\n",
				commontcl.ServiceLabel(name), params.MatrixCount, params.ShardCount)
			continue
		}

		fmt.Printf("%s: %s\n", commontcl.ServiceLabel(name), params.String(UnlimitedParallelism))

		svcInstances, svcNamespaces, err := e.buildServiceInstances(name, svc, params)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		instances = append(instances, svcInstances...)
		namespaces = append(namespaces, svcNamespaces...)

		state[name] = make([]ServiceState, len(svcInstances))
		for i := range svcInstances {
			state[name][i].Description = svcInstances[i].Description
		}
		instructions.PrintHintDetails(e.deps.Ref, data.ServicesPrefix+name, state)
	}

	return instances, namespaces, state, svcParams, nil
}

// buildServiceInstances creates instances for a single service definition.
func (e *ServicesExecutor) buildServiceInstances(
	name string,
	svc testworkflowsv1.ServiceSpec,
	params *commontcl.ParamsSpec,
) ([]ServiceInstance, []string, error) {
	svcInstances := make([]ServiceInstance, params.Count)
	svcNamespaces := make([]string, params.Count)

	for index := int64(0); index < params.Count; index++ {
		machines := []expressions.Machine{e.deps.BaseMachine, params.MachineAt(index)}

		svcSpec := svc.DeepCopy()
		if err := expressions.Simplify(&svcSpec, machines...); err != nil {
			return nil, nil, errors.Wrapf(err, "%s: %d: simplify", commontcl.ServiceLabel(name), index)
		}

		spec := testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content:   svcSpec.Content,
				Container: common.Ptr(svcSpec.ContainerConfig),
				Pod:       svcSpec.Pod,
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Run: common.Ptr(svcSpec.StepRun)}},
			},
			Pvcs: svcSpec.Pvcs,
		}
		spec.Steps[0].Run.ContainerConfig = testworkflowsv1.ContainerConfig{}
		spec.Container.Env = testworkflowresolver.DedupeEnvVars(append(e.deps.Config.Execution.GlobalEnv, spec.Container.Env...))

		if spec.Content == nil {
			spec.Content = &testworkflowsv1.Content{}
		}
		tarballs, err := spawn.ProcessTransfer(e.deps.TransferSrv, svcSpec.Transfer, machines...)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "%s: %d: transfer", commontcl.ServiceLabel(name), index)
		}
		spec.Content.Tarball = append(spec.Content.Tarball, tarballs...)

		svcInstances[index] = ServiceInstance{
			Index:          index,
			Name:           name,
			Description:    svcSpec.Description,
			RestartPolicy:  corev1.RestartPolicy(svcSpec.RestartPolicy),
			ReadinessProbe: svcSpec.ReadinessProbe,
			Spec:           spec,
		}
		svcNamespaces[index] = e.deps.Namespace
		if spec.Job != nil && spec.Job.Namespace != "" {
			svcNamespaces[index] = spec.Job.Namespace
		}

		if svcSpec.Timeout != "" {
			v, err := expressions.EvalTemplate(svcSpec.Timeout, machines...)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "%s: %d: timeout expression", commontcl.ServiceLabel(name), index)
			}
			d, err := time.ParseDuration(strings.ReplaceAll(v, " ", ""))
			if err != nil {
				return nil, nil, errors.Wrapf(err, "%s: %d: invalid timeout: %s", commontcl.ServiceLabel(name), index, v)
			}
			svcInstances[index].Timeout = &d
		}
	}

	return svcInstances, svcNamespaces, nil
}

// notifyQueuedInstances sends queued status for each instance.
func (e *ServicesExecutor) notifyQueuedInstances(instances []ServiceInstance) {
	for _, instance := range instances {
		instructions.PrintOutput(e.deps.Ref, "service", ServiceInfo{
			Group:       e.groupRef,
			Index:       instance.Index,
			Name:        instance.Name,
			Description: instance.Description,
			Status:      ServiceStatusQueued,
		})
	}
}

// startTransferServer starts the file transfer server if needed.
func (e *ServicesExecutor) startTransferServer() error {
	if e.deps.TransferSrv.Count() == 0 && e.deps.TransferSrv.RequestsCount() == 0 {
		return nil
	}

	infos := make([]string, 0)
	if e.deps.TransferSrv.Count() > 0 {
		infos = append(infos, fmt.Sprintf("sending %d tarballs", e.deps.TransferSrv.Count()))
	}
	if e.deps.TransferSrv.RequestsCount() > 0 {
		infos = append(infos, fmt.Sprintf("fetching %d requests", e.deps.TransferSrv.RequestsCount()))
	}

	fmt.Printf("Starting transfer server for %s...\n", strings.Join(infos, " and "))
	if _, err := e.deps.TransferSrv.Listen(); err != nil {
		return errors.Wrap(err, "failed to start transfer server")
	}
	fmt.Printf("Transfer server started.\n")
	return nil
}

// runServices executes all service instances in parallel.
func (e *ServicesExecutor) runServices(
	instances []ServiceInstance,
	namespaces []string,
	state map[string][]ServiceState,
	svcParams map[string]*commontcl.ParamsSpec,
) int64 {
	run := func(_ int64, _ string, instance *ServiceInstance) bool {
		runner := NewServiceRunner(instance, e.groupRef, e.deps, svcParams[instance.Name], state)
		return runner.Run()
	}

	return spawn.ExecuteParallel(context.Background(), run, instances, namespaces, int64(len(instances)))
}

// reportFinalState reports the final state of all services.
func (e *ServicesExecutor) reportFinalState(state map[string][]ServiceState) {
	for k := range state {
		instructions.PrintHintDetails(e.deps.Ref, data.ServicesPrefix+k, state[k])
	}
}

// ServiceRunner executes and monitors a single service instance.
type ServiceRunner struct {
	instance *ServiceInstance
	groupRef string
	deps     ServicesDependencies
	params   *commontcl.ParamsSpec
	state    map[string][]ServiceState
	log      func(...string)
	info     ServiceInfo
}

func NewServiceRunner(
	instance *ServiceInstance,
	groupRef string,
	deps ServicesDependencies,
	params *commontcl.ParamsSpec,
	state map[string][]ServiceState,
) *ServiceRunner {
	return &ServiceRunner{
		instance: instance,
		groupRef: groupRef,
		deps:     deps,
		params:   params,
		state:    state,
		log:      spawn.CreateLogger(instance.Name, instance.Description, instance.Index, params.Count),
		info: ServiceInfo{
			Group:       groupRef,
			Index:       instance.Index,
			Name:        instance.Name,
			Description: instance.Description,
			Status:      ServiceStatusQueued,
		},
	}
}

// Run executes the service and returns true if successful.
func (r *ServiceRunner) Run() bool {
	namespace := r.deps.Namespace
	if r.instance.Spec.Job != nil && r.instance.Spec.Job.Namespace != "" {
		namespace = r.instance.Spec.Job.Namespace
	}

	cfg := *r.deps.Config
	cfg.Resource = spawn.CreateResourceConfig(r.instance.Name+"-", r.instance.Index)
	cfg.Worker.Namespace = namespace
	machine := expressions.CombinedMachines(
		testworkflowconfig.CreateResourceMachine(&cfg.Resource),
		testworkflowconfig.CreateWorkerMachine(&cfg.Worker),
		r.deps.BaseMachine,
		testworkflowconfig.CreatePvcMachine(cfg.Execution.PvcNames),
		r.params.MachineAt(r.instance.Index),
	)

	_ = expressions.Simplify(&r.instance.Spec, machine)

	result, err := r.deployService(cfg)
	if err != nil {
		r.log("failed to prepare resources", err.Error())
		return false
	}

	execResult := r.monitorService(cfg, result)
	return r.evaluateResult(execResult)
}

// deployService deploys the service to the cluster.
func (r *ServiceRunner) deployService(cfg testworkflowconfig.InternalConfig) (*executionworkertypes.ServiceResult, error) {
	scheduledAt := time.Now()
	return r.deps.ExecutionWorker.Service(context.Background(), executionworkertypes.ServiceRequest{
		ResourceId:          cfg.Resource.Id,
		GroupId:             r.groupRef,
		Execution:           cfg.Execution,
		Workflow:            testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: cfg.Workflow.Name, Labels: cfg.Workflow.Labels}, Spec: r.instance.Spec},
		ScheduledAt:         &scheduledAt,
		RestartPolicy:       string(r.instance.RestartPolicy),
		ReadinessProbe:      common.MapPtr(r.instance.ReadinessProbe, testworkflows.MapProbeKubeToAPI),
		ControlPlane:        cfg.ControlPlane,
		ArtifactsPathPrefix: cfg.Resource.FsPrefix,
	})
}

// monitorService monitors the service execution and returns the result.
func (r *ServiceRunner) monitorService(cfg testworkflowconfig.InternalConfig, result *executionworkertypes.ServiceResult) ServiceExecutionResult {
	signatureSeq := stage.MapSignatureToSequence(stage.MapSignatureList(result.Signature))
	mainRef := signatureSeq[len(signatureSeq)-1].Ref()

	timeoutCtx, timeoutCtxCancel := context.WithCancel(context.Background())
	defer timeoutCtxCancel()
	if r.instance.Timeout != nil {
		go func() {
			select {
			case <-timeoutCtx.Done():
			case <-time.After(*r.instance.Timeout):
				r.log("timed out", r.instance.Timeout.String()+" elapsed")
				timeoutCtxCancel()
			}
		}()
	}

	ctx, ctxCancel := context.WithCancel(timeoutCtx)
	defer ctxCancel()

	notifications := r.deps.ExecutionWorker.StatusNotifications(ctx, cfg.Resource.Id, executionworkertypes.StatusNotificationsOptions{
		Hints: executionworkertypes.Hints{
			Namespace:   result.Namespace,
			Signature:   result.Signature,
			ScheduledAt: &result.ScheduledAt,
		},
	})
	if notifications.Err() != nil {
		r.log("error", "failed to connect to the service", notifications.Err().Error())
		return ServiceExecutionResult{Error: notifications.Err()}
	}
	r.log("created")

	execResult := r.processNotifications(notifications, mainRef)

	// For services without readiness probe, do a brief post-startup check
	// to catch services that fail immediately after starting
	if r.instance.ReadinessProbe == nil && execResult.Started && !execResult.Failed {
		execResult = r.checkForImmediateFailure(cfg, result, execResult)
	}

	return execResult
}

// checkForImmediateFailure polls to detect services that fail immediately after starting.
// Services without readiness probes exit the monitoring loop on startup, so we need
// a secondary check to catch fast failures before declaring success.
func (r *ServiceRunner) checkForImmediateFailure(
	cfg testworkflowconfig.InternalConfig,
	result *executionworkertypes.ServiceResult,
	execResult ServiceExecutionResult,
) ServiceExecutionResult {
	ctx, cancel := context.WithTimeout(context.Background(), serviceFailureCheckTimeout)
	defer cancel()

	notifications := r.deps.ExecutionWorker.StatusNotifications(ctx, cfg.Resource.Id, executionworkertypes.StatusNotificationsOptions{
		Hints: executionworkertypes.Hints{
			Namespace:   result.Namespace,
			Signature:   result.Signature,
			ScheduledAt: &result.ScheduledAt,
		},
	})
	if notifications.Err() != nil {
		return execResult
	}

	for v := range notifications.Channel() {
		if v.Result != nil && v.Result.IsFinished() {
			if !v.Result.IsPassed() {
				execResult.Failed = true
				r.log("service failed immediately after starting")
			}
			break
		}
	}

	return execResult
}

// processNotifications processes status notifications and tracks service state.
func (r *ServiceRunner) processNotifications(
	notifications executionworkertypes.StatusNotificationsWatcher,
	mainRef string,
) ServiceExecutionResult {
	execResult := ServiceExecutionResult{
		Ready: r.instance.ReadinessProbe == nil,
	}
	var lastWorkflowResult *testkube.TestWorkflowResult

	scheduled := false
	index := r.instance.Index

	for v := range notifications.Channel() {
		if !scheduled && v.NodeName != "" {
			scheduled = true
			r.log(fmt.Sprintf("assigned to %s node", ui.LightBlue(v.NodeName)))
		}

		if r.state[r.instance.Name][index].Ip == "" && v.PodIp != "" {
			r.state[r.instance.Name][index].Ip = v.PodIp
			r.log(fmt.Sprintf("assigned to %s IP", ui.LightBlue(v.PodIp)))
			r.info.Status = ServiceStatusRunning
			instructions.PrintOutput(r.deps.Ref, "service", r.info)
		}

		if v.Result != nil {
			lastWorkflowResult = v.Result
		}

		execResult.Ready = v.Ready || r.instance.ReadinessProbe == nil

		if !execResult.Started && v.Ref == mainRef && r.state[r.instance.Name][index].Ip != "" {
			execResult.Started = true
			if r.instance.ReadinessProbe == nil {
				r.log("container started")
			} else {
				r.log("container started, waiting for readiness")
			}
		}

		if lastWorkflowResult != nil && lastWorkflowResult.IsFinished() {
			if !lastWorkflowResult.IsPassed() {
				execResult.Failed = true
				r.log("service execution failed")
			}
			break
		}

		if execResult.Started && execResult.Ready && r.instance.ReadinessProbe != nil {
			break
		}

		// For services without readiness probe, break once started
		if execResult.Started && r.instance.ReadinessProbe == nil {
			break
		}
	}

	if notifications.Err() != nil {
		r.log("error", notifications.Err().Error())
		execResult.Error = notifications.Err()
	}

	return execResult
}

// evaluateResult determines the final status and returns success/failure.
func (r *ServiceRunner) evaluateResult(execResult ServiceExecutionResult) bool {
	success := true

	// Check for failures in order of priority
	if execResult.Error != nil {
		r.info.Status = ServiceStatusFailed
		r.log("error during monitoring")
		success = false
	} else if execResult.Failed {
		// Service execution finished with non-PASSED status
		r.info.Status = ServiceStatusFailed
		r.log("service failed")
		success = false
	} else if !execResult.Started {
		// Container never started
		r.info.Status = ServiceStatusFailed
		r.log("container failed to start")
		success = false
	} else if !execResult.Ready && r.instance.ReadinessProbe != nil {
		// Container started but never became ready (only relevant for services with readiness probes)
		r.info.Status = ServiceStatusFailed
		r.log("container did not reach readiness")
		success = false
	} else {
		// All checks passed - service is ready
		r.info.Status = ServiceStatusReady
		r.log("container ready")
	}

	instructions.PrintOutput(r.deps.Ref, "service", r.info)
	return success
}
