package localrunner

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cache"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/kubernetesworker"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"
)

const DefaultMaxSourceBytes int64 = 100 << 20

// localSourceURLPattern catches source-relay URLs emitted by the TestWorkflow
// runtime itself. Those messages are outside the CLI's SourcePlan plumbing, so
// retaining only the caller-known URL would risk printing the bearer-like
// random path on a reconnect or a reconstructed notification stream.
var localSourceURLPattern = regexp.MustCompile(`https?://local-source-[a-z0-9-]+(?::[0-9]+)?/[a-f0-9]+\.tar\.gz`)

// redactedLocalSourceError preserves errors.Is/errors.As while ensuring a
// temporary source relay URL or token can never reach Cobra's terminal error
// renderer. The original error remains available to callers through Unwrap.
type redactedLocalSourceError struct {
	err        error
	redactions []string
}

func (e *redactedLocalSourceError) Error() string {
	return redactLogValues(e.err.Error(), e.redactions)
}

func (e *redactedLocalSourceError) Unwrap() error { return e.err }

func redactLocalSourceError(err error, redactions ...string) error {
	if err == nil {
		return nil
	}
	return &redactedLocalSourceError{err: err, redactions: redactions}
}

func localKubeconfigFlag(kubeconfig string) string {
	if kubeconfig == "" {
		return ""
	}
	return " --kubeconfig " + shellQuote(kubeconfig)
}

func localInspectHint(runID, namespace, contextName, kubeconfig string) string {
	return fmt.Sprintf(
		"kubectl%s --context %s -n %s get job,pod,service,secret,configmap,pvc -l %s=%s",
		localKubeconfigFlag(kubeconfig),
		shellQuote(contextName),
		shellQuote(namespace),
		LocalRunIDLabel,
		shellQuote(runID),
	)
}

func localCleanHint(runID, namespace, contextName, kubeconfig string) string {
	return fmt.Sprintf(
		"testkube local clean %s --namespace %s --context %s%s",
		shellQuote(runID),
		shellQuote(namespace),
		shellQuote(contextName),
		localKubeconfigFlag(kubeconfig),
	)
}

// Options is the complete dependency-free contract used by `testkube local
// run`. It intentionally contains Kubernetes selection rather than any
// Testkube API configuration.
type Options struct {
	FilePath       string
	SourceDir      string
	SourceMount    string
	SourceIncludes []string
	SourceExcludes []string
	MaxSourceBytes int64
	Config         map[string]string
	Variables      map[string]string
	Namespace      string
	Kubeconfig     string
	ContextName    string
	AllowNonLocal  bool
	Interactive    bool
	AutoContinue   bool
	Keep           bool
	DryRun         bool
	In             io.Reader
	Out            io.Writer
	ErrOut         io.Writer
}

// PreparedRun holds only local state and direct Kubernetes configuration. It
// has not mutated the selected cluster yet.
type PreparedRun struct {
	RunID      string
	Workflow   *testworkflowsv1.TestWorkflow
	Runtime    *executionworkertypes.Runtime
	Target     KubeTarget
	Namespace  string
	Kubeconfig string
	Source     *SourcePlan
}

// Result is returned for an executed workflow. A non-passing result is also
// accompanied by an ExecutionError so Cobra exits with status 1.
type Result struct {
	RunID     string
	Status    string
	Passed    bool
	Namespace string
}

// Prepare validates all user-controlled workflow and source inputs before any
// namespace, relay, Secret, ConfigMap, PVC, Job, or Pod is created.
func Prepare(ctx context.Context, opts Options) (*PreparedRun, error) {
	_ = ctx // preparation is deliberately local apart from kubeconfig parsing.
	if opts.Namespace == "" {
		opts.Namespace = DefaultNamespace
	}
	if err := ValidateNamespace(opts.Namespace); err != nil {
		return nil, err
	}
	target, err := ResolveKubeTarget(opts.Kubeconfig, opts.ContextName, opts.AllowNonLocal)
	if err != nil {
		return nil, err
	}
	workflow, _, err := LoadWorkflow(opts.FilePath)
	if err != nil {
		return nil, err
	}
	// Reject API-backed templates and every other unsupported structural field
	// before applying configuration. Configuration resolution is local today,
	// but this ordering keeps the no-Control-Plane contract explicit and avoids
	// ever trying to resolve a template reference from a local invocation.
	if err = ValidateSupportedInNamespace(workflow, opts.Namespace, opts.SourceDir != "", opts.Interactive, opts.AutoContinue); err != nil {
		return nil, err
	}
	workflow, err = ApplyConfig(workflow, opts.Config)
	if err != nil {
		return nil, err
	}
	// Re-run the structural gate after configuration in case a parameterized
	// value resolved to a local-runner-incompatible field.
	if err = ValidateSupportedInNamespace(workflow, opts.Namespace, opts.SourceDir != "", opts.Interactive, opts.AutoContinue); err != nil {
		return nil, err
	}
	workflow = workflow.DeepCopy()
	if workflow.Spec.Job == nil {
		workflow.Spec.Job = &testworkflowsv1.JobConfig{}
	} else {
		workflow.Spec.Job = workflow.Spec.Job.DeepCopy()
	}
	// Explicitly pin the rendered Job to the selected local namespace. This
	// avoids accepting a workflow's production namespace as hidden authority.
	workflow.Spec.Job.Namespace = opts.Namespace

	prepared := &PreparedRun{
		RunID:      newRunID(workflow.Name),
		Workflow:   workflow,
		Runtime:    &executionworkertypes.Runtime{Variables: copyStringMap(opts.Variables)},
		Target:     target,
		Namespace:  opts.Namespace,
		Kubeconfig: opts.Kubeconfig,
	}
	if opts.SourceDir != "" {
		root, err := sourceRoot(opts.SourceDir)
		if err != nil {
			return nil, UsageError("validate --source: %v", err)
		}
		maxBytes := opts.MaxSourceBytes
		if maxBytes == 0 {
			maxBytes = DefaultMaxSourceBytes
		}
		if maxBytes < 0 {
			return nil, UsageError("--max-source-bytes must be greater than zero")
		}
		mountPath, err := ResolveSourceMount(workflow, opts.SourceMount)
		if err != nil {
			return nil, err
		}
		if err = ValidateSourceMountAvailable(workflow, mountPath); err != nil {
			return nil, err
		}
		prepared.Source = &SourcePlan{
			Options: SourceOptions{
				Directory: root,
				Includes:  append([]string(nil), opts.SourceIncludes...),
				Excludes:  append([]string(nil), opts.SourceExcludes...),
				MaxBytes:  maxBytes,
			},
			MountPath: mountPath,
		}
	}
	return prepared, nil
}

// Run performs a prepared direct-Kubernetes execution. It deliberately builds
// no Testkube API client and uses the source relay only after all validation is
// complete. Every mutation is protected by exact-label cleanup unless --keep.
func Run(ctx context.Context, opts Options) (result *Result, err error) {
	prepared, err := Prepare(ctx, opts)
	if err != nil {
		return nil, err
	}
	// Source-rewrite and relay errors can contain the temporary bearer-like URL
	// in processor or transport details. Register this before any cleanup defer
	// so it is the final transformation of every returned error.
	defer func() {
		if prepared.Source != nil && prepared.Source.URL != "" {
			err = redactLocalSourceError(err, prepared.Source.URL)
		}
	}()
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}
	fmt.Fprintf(out, "local run: %s (%s, namespace %s)\n", prepared.RunID, KubeTargetDescription(prepared.Target), prepared.Namespace)
	if opts.DryRun {
		fmt.Fprintln(out, "dry run: workflow and local-source inputs are valid; no Kubernetes resources were created")
		return &Result{RunID: prepared.RunID, Status: "validated", Passed: true, Namespace: prepared.Namespace}, nil
	}

	client, err := kubernetes.NewForConfig(prepared.Target.RESTConfig)
	if err != nil {
		return nil, ExecutionError("create Kubernetes client for %s: %v", KubeTargetDescription(prepared.Target), err)
	}
	resources := NewResourceManager(client, prepared.Namespace)
	if err = resources.EnsureNamespace(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, InterruptedError(ctx.Err())
		}
		return nil, ExecutionError("prepare local namespace: %v", err)
	}
	forceCleanup := false
	if stale, staleErr := resources.StaleRunIDs(ctx, time.Now()); staleErr != nil {
		if ctx.Err() != nil {
			return nil, InterruptedError(ctx.Err())
		}
		fmt.Fprintf(errOut, "warning: could not check retained local runs: %v\n", staleErr)
	} else {
		for _, runID := range stale {
			fmt.Fprintf(errOut, "warning: retained local run %s is older than 24h; clean it with: %s\n", runID, localCleanHint(runID, prepared.Namespace, prepared.Target.ContextName, prepared.Kubeconfig))
		}
	}

	if !opts.Keep {
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupWaitTimeout)
			defer cancel()
			cleanupErr := resources.Clean(cleanupCtx, prepared.RunID)
			if cleanupErr == nil {
				fmt.Fprintf(out, "cleanup: removed exact local resources for %s\n", prepared.RunID)
				return
			}
			if err == nil {
				err = ExecutionError("clean local run %s: %v", prepared.RunID, cleanupErr)
				return
			}
			fmt.Fprintf(errOut, "warning: cleanup local run %s: %v\n", prepared.RunID, cleanupErr)
		}()
	} else {
		defer func() {
			if forceCleanup || IsInterruptedError(err) {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupWaitTimeout)
				defer cancel()
				cleanupErr := resources.Clean(cleanupCtx, prepared.RunID)
				if cleanupErr == nil {
					fmt.Fprintf(out, "cleanup: removed exact local resources for %s\n", prepared.RunID)
					return
				}
				if err == nil {
					err = ExecutionError("clean local run %s: %v", prepared.RunID, cleanupErr)
					return
				}
				fmt.Fprintf(errOut, "warning: cleanup local run %s: %v\n", prepared.RunID, cleanupErr)
				return
			}
			fmt.Fprintf(out, "kept local run %s; inspect with: %s\n", prepared.RunID, localInspectHint(prepared.RunID, prepared.Namespace, prepared.Target.ContextName, prepared.Kubeconfig))
			fmt.Fprintf(out, "clean it with: %s\n", localCleanHint(prepared.RunID, prepared.Namespace, prepared.Target.ContextName, prepared.Kubeconfig))
		}()
	}

	if prepared.Source != nil {
		relay, relayErr := CreateSourceRelay(ctx, client, prepared.Target.RESTConfig, prepared.Namespace, prepared.RunID)
		if relayErr != nil {
			if ctx.Err() != nil {
				return nil, InterruptedError(ctx.Err())
			}
			return nil, ExecutionError("create local source relay: %v", relayErr)
		}
		// Assign the URL before upload so every subsequent returned error is
		// redacted, including a remotecommand error that echoes the shell command.
		prepared.Source.URL = relay.URL
		summary, uploadErr := relay.Upload(ctx, prepared.Source.Options)
		if uploadErr != nil {
			if errors.Is(uploadErr, context.Canceled) || ctx.Err() != nil {
				return nil, InterruptedError(ctx.Err())
			}
			return nil, ExecutionError("upload local source: %v", uploadErr)
		}
		prepared.Workflow, err = RewriteWorkflowWithSource(prepared.Workflow, relay.URL, prepared.Source.MountPath)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(out, "source: local archive (%d files, %d bytes) -> %s\n", summary.Files, summary.Bytes, prepared.Source.MountPath)
	}

	workflowLabels, err := Labels(prepared.RunID, "workflow")
	if err != nil {
		return nil, err
	}
	worker := newLocalWorker(client, prepared.Namespace, prepared.RunID)
	now := time.Now().UTC()
	ttl := int32(defaultJobTTL / time.Second)
	execution := testworkflowconfig.ExecutionConfig{
		Id:              prepared.RunID,
		GroupId:         prepared.RunID,
		Name:            prepared.RunID,
		ScheduledAt:     now,
		DisableWebhooks: true,
	}
	deployed, err := worker.Execute(ctx, executionworkertypes.ExecuteRequest{
		ResourceId:              prepared.RunID,
		GroupId:                 prepared.RunID,
		Workflow:                *prepared.Workflow,
		Labels:                  workflowLabels,
		TTLSecondsAfterFinished: &ttl,
		ScheduledAt:             &now,
		Runtime:                 prepared.Runtime,
		Execution:               execution,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, InterruptedError(ctx.Err())
		}
		return nil, ExecutionError("deploy local workflow: %v", err)
	}
	fmt.Fprintf(out, "workflow: %s deployed\n", prepared.Workflow.Name)
	control := NewLocalControl(client, prepared.Target.RESTConfig, prepared.Namespace)
	abort := func() error {
		forceCleanup = true
		abortCtx, cancel := context.WithTimeout(context.Background(), localAbortTimeout)
		defer cancel()
		abortErr := worker.Abort(abortCtx, prepared.RunID, executionworkertypes.DestroyOptions{Namespace: prepared.Namespace})
		if apierrors.IsNotFound(abortErr) {
			return nil
		}
		return abortErr
	}
	return followWorkflow(ctx, client, worker, deployed, prepared, control, abort, opts, out, errOut)
}

const (
	localAbortTimeout          = 15 * time.Second
	notificationDrainTimeout   = 2 * time.Second
	localJobStatusPollInterval = 250 * time.Millisecond
)

func newLocalWorker(client kubernetes.Interface, namespace, runID string) executionworkertypes.Worker {
	secretClient := secret.NewClientFor(client, namespace)
	inspector := imageinspector.NewInspector(
		"",
		imageinspector.NewCraneFetcher(),
		imageinspector.NewSecretFetcher(secretClient, cache.NewInMemoryCache[*corev1.Secret]()),
		imageinspector.NewMemoryStorage(),
	)
	return executionworker.NewKubernetes(client, presets.NewOpenSource(inspector), kubernetesworker.Config{
		Cluster: kubernetesworker.ClusterConfig{
			Id:               "local-" + runID,
			DefaultNamespace: namespace,
			Namespaces:       map[string]kubernetesworker.NamespaceConfig{namespace: {}},
		},
		RunnerId:               "local-" + runID,
		Connection:             testworkflowconfig.WorkerConnectionConfig{},
		DisableResourceMetrics: true,
		AllowLowSecurityFields: false,
	})
}

func followWorkflow(
	ctx context.Context,
	client kubernetes.Interface,
	worker executionworkertypes.Worker,
	deployed *executionworkertypes.ExecuteResult,
	prepared *PreparedRun,
	control *LocalControl,
	abort func() error,
	opts Options,
	out, errOut io.Writer,
) (*Result, error) {
	watchCtx, cancelWatch := context.WithCancel(ctx)
	defer cancelWatch()
	watcher := worker.Notifications(watchCtx, prepared.RunID, executionworkertypes.NotificationsOptions{
		Hints: executionworkertypes.Hints{
			Namespace:   prepared.Namespace,
			ScheduledAt: &deployed.ScheduledAt,
			Signature:   deployed.Signature,
		},
	})
	names := signatureNames(deployed.Signature)
	lastStepStatuses := map[string]string{}
	lastPause := ""
	finish := func(result *Result) (*Result, error) {
		fmt.Fprintf(out, "result: %s\n", result.Status)
		if !result.Passed {
			return result, ExecutionError("local workflow %s finished with %s", prepared.RunID, result.Status)
		}
		return result, nil
	}

	// The production controller follows structured logs. Kubernetes Job status
	// remains a bounded fallback, but it is deliberately not an immediate
	// return: a fast Job may become terminal shortly before the watcher delivers
	// buffered logs or its final structured result.
	jobPoll := time.NewTicker(localJobStatusPollInterval)
	defer jobPoll.Stop()
	notifications := watcher.Channel()
	var polledTerminal *Result
	var drainTimer *time.Timer
	var drain <-chan time.Time
	defer func() {
		if drainTimer != nil {
			if !drainTimer.Stop() {
				select {
				case <-drainTimer.C:
				default:
				}
			}
		}
	}()
	resetDrain := func() {
		if drainTimer == nil {
			return
		}
		if !drainTimer.Stop() {
			select {
			case <-drainTimer.C:
			default:
			}
		}
		drainTimer.Reset(notificationDrainTimeout)
	}

	// handleNotification keeps one rendering path for normal notification
	// delivery and the final nonblocking drain after a terminal Job poll.
	handleNotification := func(notification *testkube.TestWorkflowExecutionNotification) (*Result, error, bool) {
		if notification == nil || isProtocolNotification(notification.EventType) {
			return nil, nil, false
		}
		if notification.Log != "" {
			renderLogRedacted(out, notification.Log, localLogRedactions(prepared.Source))
		}
		if notification.Output != nil {
			fmt.Fprintf(out, "output: %s\n", notification.Output.Name)
		}
		if notification.Result == nil {
			return nil, nil, false
		}
		renderStepStatus(out, notification.Result, names, lastStepStatuses)
		if notification.Result.IsFinished() {
			result := &Result{RunID: prepared.RunID, Status: resultStatus(notification.Result), Passed: notification.Result.IsPassed(), Namespace: prepared.Namespace}
			result, err := finish(result)
			return result, err, true
		}
		// Once Kubernetes has observed the Job terminal state, keep rendering
		// trailing structured output but do not surface a stale pause prompt.
		if polledTerminal != nil || !notification.Result.IsPaused() {
			return nil, nil, false
		}
		pauseRef := latestPauseRef(notification.Result)
		if pauseRef == "" || pauseRef == lastPause {
			return nil, nil, false
		}
		lastPause = pauseRef
		label := names[pauseRef]
		if label == "" {
			label = pauseRef
		}
		fmt.Fprintf(out, "breakpoint: %s paused\n", label)
		if opts.AutoContinue {
			fmt.Fprintln(out, "breakpoint: auto-continuing")
			if err := control.Resume(ctx, prepared.RunID); err != nil {
				if ctx.Err() != nil {
					if abortErr := abort(); abortErr != nil {
						fmt.Fprintf(errOut, "warning: abort local workflow %s after interrupt: %v\n", prepared.RunID, abortErr)
					}
					return nil, InterruptedError(ctx.Err()), true
				}
				return nil, ExecutionError("resume local workflow breakpoint: %v", err), true
			}
			return nil, nil, false
		}
		if !opts.Interactive {
			return nil, nil, false
		}
		if err := promptBreakpoint(ctx, control, prepared.RunID, opts, out, errOut); err != nil {
			if errors.Is(err, errBreakpointAbort) {
				if abortErr := abort(); abortErr != nil {
					return nil, ExecutionError("abort local workflow %s: %v", prepared.RunID, abortErr), true
				}
				return nil, ExecutionError("local workflow aborted by developer"), true
			}
			if IsInterruptedError(err) {
				if abortErr := abort(); abortErr != nil {
					fmt.Fprintf(errOut, "warning: abort local workflow %s after interrupt: %v\n", prepared.RunID, abortErr)
				}
			}
			return nil, err, true
		}
		return nil, nil, false
	}

	for {
		select {
		case <-ctx.Done():
			if abort != nil {
				if abortErr := abort(); abortErr != nil {
					fmt.Fprintf(errOut, "warning: abort local workflow %s after interrupt: %v\n", prepared.RunID, abortErr)
				}
			}
			return nil, InterruptedError(ctx.Err())
		case <-jobPoll.C:
			if polledTerminal != nil {
				continue
			}
			finished, passed, status, pollErr := localJobOutcome(ctx, client, prepared.Namespace, prepared.RunID)
			if pollErr != nil {
				fmt.Fprintf(errOut, "warning: read local workflow Job status: %v\n", pollErr)
				continue
			}
			if finished {
				polledTerminal = &Result{RunID: prepared.RunID, Status: status, Passed: passed, Namespace: prepared.Namespace}
				if notifications == nil {
					return finish(polledTerminal)
				}
				drainTimer = time.NewTimer(notificationDrainTimeout)
				drain = drainTimer.C
			}
		case <-drain:
			// Do not let a ready timer win a select race over buffered trailing
			// notifications. Drain every currently ready item before using the
			// Kubernetes terminal status as the fallback result.
			for notifications != nil {
				select {
				case notification, ok := <-notifications:
					if !ok {
						notifications = nil
						break
					}
					if result, err, done := handleNotification(notification); done {
						return result, err
					}
				default:
					return finish(polledTerminal)
				}
			}
			return finish(polledTerminal)
		case notification, ok := <-notifications:
			if !ok {
				notifications = nil
				if watchErr := watcher.Err(); watchErr != nil {
					fmt.Fprintf(errOut, "warning: structured local workflow notification stream ended: %v; using Kubernetes Job status\n", watchErr)
				}
				if polledTerminal != nil {
					return finish(polledTerminal)
				}
				continue
			}
			resetDrain()
			if result, err, done := handleNotification(notification); done {
				return result, err
			}
		}
	}
}

var errBreakpointAbort = errors.New("local workflow abort requested")

func localJobOutcome(ctx context.Context, client kubernetes.Interface, namespace, runID string) (finished, passed bool, status string, err error) {
	job, err := client.BatchV1().Jobs(namespace).Get(ctx, runID, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, false, "", nil
	}
	if err != nil {
		return false, false, "", err
	}
	if job.Status.Succeeded > 0 {
		return true, true, "passed", nil
	}
	if job.Status.Failed > 0 {
		return true, false, "failed", nil
	}
	for _, condition := range job.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			continue
		}
		switch condition.Type {
		case "Complete":
			return true, true, "passed", nil
		case "Failed":
			return true, false, "failed", nil
		}
	}
	return false, false, "", nil
}

func promptBreakpoint(ctx context.Context, control *LocalControl, runID string, opts Options, out, errOut io.Writer) error {
	in := opts.In
	if in == nil {
		return UsageError("local workflow breakpoint needs interactive standard input; pass --auto-continue for noninteractive use")
	}
	reader := bufio.NewReader(in)
	for {
		fmt.Fprint(out, "breakpoint command [continue, shell, pod, abort]: ")
		line, err := readBreakpointLine(ctx, reader)
		if ctx.Err() != nil {
			return InterruptedError(ctx.Err())
		}
		if err != nil && len(line) == 0 {
			return ExecutionError("read breakpoint command: %v", err)
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "c", "continue":
			if err := control.Resume(ctx, runID); err != nil {
				if ctx.Err() != nil {
					return InterruptedError(ctx.Err())
				}
				fmt.Fprintf(errOut, "continue unavailable: %v\n", err)
				continue
			}
			return nil
		case "s", "shell":
			// Do not hand the shared terminal directly to remotecommand. Its
			// background stdin copier can otherwise consume the first command
			// typed at this breakpoint after `exit`. The line-aware adapter ends
			// its stream after an explicit shell exit and leaves later input in
			// this prompt's buffered reader.
			shellIn := &breakpointShellInput{reader: reader}
			if err := control.Shell(ctx, runID, IOStreams{In: shellIn, Out: out, Err: errOut, TTY: opts.Interactive}); err != nil {
				fmt.Fprintf(errOut, "shell unavailable: %v\n", err)
			}
		case "p", "pod":
			pod, events, err := control.PodDetails(ctx, runID)
			if err != nil {
				fmt.Fprintf(errOut, "pod unavailable: %v\n", err)
				continue
			}
			fmt.Fprintf(out, "pod: %s phase=%s node=%s\n", pod.Name, pod.Status.Phase, pod.Spec.NodeName)
			if len(events) == 0 {
				fmt.Fprintln(out, "events: none")
				continue
			}
			for _, event := range events {
				occurredAt := podEventTime(event)
				timestamp := occurredAt.UTC().Format(time.RFC3339)
				if occurredAt.IsZero() {
					timestamp = "unknown-time"
				}
				fmt.Fprintf(out, "event: %s %s %s: %s\n", timestamp, event.Type, event.Reason, event.Message)
			}
		case "a", "abort":
			return errBreakpointAbort
		default:
			fmt.Fprintln(errOut, "unknown breakpoint command; choose continue, shell, pod, or abort")
		}
	}
}

type breakpointLineResult struct {
	line string
	err  error
}

// readBreakpointLine lets Ctrl-C stop a paused workflow even when a terminal
// read is waiting for Enter. The CLI process exits after the caller handles
// cancellation, so an unavoidable blocked stdin read cannot retain cluster
// resources or delay the result.
func readBreakpointLine(ctx context.Context, reader *bufio.Reader) (string, error) {
	result := make(chan breakpointLineResult, 1)
	go func() {
		line, err := reader.ReadString('\n')
		result <- breakpointLineResult{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-result:
		return result.line, result.err
	}
}

// breakpointShellInput passes complete command lines to the remote shell. An
// explicit `exit`/`logout` ends the remote stdin stream after that line, so the
// remotecommand copy goroutine cannot steal the next local breakpoint choice.
// It is only used by the breakpoint prompt; standalone `local shell` remains
// a normal unrestricted terminal session.
type breakpointShellInput struct {
	reader     *bufio.Reader
	pending    []byte
	closeAfter bool
	closed     bool
}

func (s *breakpointShellInput) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	if s.closed {
		return 0, io.EOF
	}
	if len(s.pending) == 0 {
		line, err := s.reader.ReadString('\n')
		if len(line) == 0 {
			s.closed = true
			return 0, err
		}
		s.pending = []byte(line)
		command := strings.TrimSpace(line)
		s.closeAfter = command == "exit" || command == "logout"
	}
	written := copy(destination, s.pending)
	s.pending = s.pending[written:]
	if len(s.pending) == 0 && s.closeAfter {
		s.closed = true
	}
	return written, nil
}

func renderLogRedacted(out io.Writer, logs string, redactions []string) {
	for _, line := range strings.SplitAfter(logs, "\n") {
		if line == "" {
			continue
		}
		trimmed := trimLocalTimestamp(strings.TrimSuffix(line, "\n"))
		if trimmed == "" {
			continue
		}
		fmt.Fprintln(out, redactLogValues(trimmed, redactions))
	}
}

func localLogRedactions(source *SourcePlan) []string {
	if source == nil || source.URL == "" {
		return nil
	}
	token := source.URL[strings.LastIndex(source.URL, "/")+1:]
	token = strings.TrimSuffix(token, ".tar.gz")
	return []string{source.URL, token}
}

func redactLogValues(value string, redactions []string) string {
	value = localSourceURLPattern.ReplaceAllString(value, "<local-source>")
	for _, secret := range redactions {
		if secret == "" {
			continue
		}
		value = strings.ReplaceAll(value, secret, "<local-source>")
	}
	return value
}

func trimLocalTimestamp(line string) string {
	if len(line) > 11 && line[10] == 'T' {
		if index := strings.IndexByte(line, ' '); index >= 0 && index+1 < len(line) {
			return line[index+1:]
		}
	}
	return line
}

func renderStepStatus(out io.Writer, result *testkube.TestWorkflowResult, names, previous map[string]string) {
	if result.Initialization != nil && result.Initialization.Status != nil {
		status := string(*result.Initialization.Status)
		if previous["__init__"] != status {
			previous["__init__"] = status
			fmt.Fprintf(out, "step: initialization %s\n", status)
		}
	}
	for ref, step := range result.Steps {
		if step.Status == nil {
			continue
		}
		status := string(*step.Status)
		if previous[ref] == status {
			continue
		}
		previous[ref] = status
		name := names[ref]
		if name == "" {
			name = ref
		}
		fmt.Fprintf(out, "step: %s %s\n", name, status)
	}
}

func signatureNames(signature []testkube.TestWorkflowSignature) map[string]string {
	names := map[string]string{}
	var walk func([]testkube.TestWorkflowSignature)
	walk = func(items []testkube.TestWorkflowSignature) {
		for _, item := range items {
			names[item.Ref] = item.Name
			walk(item.Children)
		}
	}
	walk(signature)
	return names
}

func latestPauseRef(result *testkube.TestWorkflowResult) string {
	for index := len(result.Pauses) - 1; index >= 0; index-- {
		pause := result.Pauses[index]
		if pause.ResumedAt.IsZero() {
			return pause.Ref
		}
	}
	return ""
}

func resultStatus(result *testkube.TestWorkflowResult) string {
	if result == nil || result.Status == nil {
		return "unknown"
	}
	return string(*result.Status)
}

func isProtocolNotification(eventType string) bool {
	return eventType == "ready" || eventType == "heartbeat" || eventType == "resume_unavailable"
}

func newRunID(name string) string {
	base := strings.ToLower(name)
	base = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, base)
	// A human-readable workflow name can contain consecutive separators (for
	// example, spaces around a slash). Collapse them so the generated run ID is
	// stable, readable, and leaves as much of its 63-character budget as
	// possible for the random suffix.
	for strings.Contains(base, "--") {
		base = strings.ReplaceAll(base, "--", "-")
	}
	base = strings.Trim(base, "-")
	if base == "" {
		base = "workflow"
	}
	if len(base) > 42 {
		base = strings.TrimRight(base[:42], "-")
	}
	suffixBytes := make([]byte, 6)
	if _, err := rand.Read(suffixBytes); err != nil {
		// crypto/rand failure is extraordinarily rare. A nanosecond suffix is
		// still unique enough for the local label scope and preserves progress.
		fallback := fmt.Sprintf("local-%s-%x", base, time.Now().UnixNano())
		if len(fallback) > 63 {
			return fallback[:63]
		}
		return fallback
	}
	return "local-" + base + "-" + hex.EncodeToString(suffixBytes)
}
