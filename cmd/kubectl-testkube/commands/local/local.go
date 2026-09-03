// Package local exposes TestWorkflow development commands that operate only on
// a selected Kind or k3d Kubernetes cluster. It intentionally imports no
// ordinary Testkube API command dependencies.
package local

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/testworkflows/localrunner"
)

type targetOptions struct {
	namespace     string
	kubeconfig    string
	contextName   string
	allowNonLocal bool
}

// NewLocalCmd returns the fully offline local TestWorkflow command family.
func NewLocalCmd() *cobra.Command {
	target := &targetOptions{namespace: localrunner.DefaultNamespace}
	cmd := &cobra.Command{
		Use:          "local",
		Short:        "Run an uncommitted TestWorkflow directly on a local Kubernetes cluster",
		SilenceUsage: true,
	}
	addTargetFlags(cmd, target)
	cmd.AddCommand(newRunCmd(target))
	cmd.AddCommand(newPauseCmd(target))
	cmd.AddCommand(newResumeCmd(target))
	cmd.AddCommand(newShellCmd(target))
	cmd.AddCommand(newCleanCmd(target))
	return cmd
}

func addTargetFlags(cmd *cobra.Command, target *targetOptions) {
	// This local persistent flag intentionally shadows the ordinary CLI's
	// config-derived namespace default. Local resources should never inherit an
	// installed Testkube namespace accidentally.
	cmd.PersistentFlags().StringVar(&target.namespace, "namespace", localrunner.DefaultNamespace, "namespace for local workflow resources")
	cmd.PersistentFlags().StringVar(&target.kubeconfig, "kubeconfig", "", "path to the Kubernetes kubeconfig")
	cmd.PersistentFlags().StringVar(&target.contextName, "context", "", "Kubernetes context for this command (does not change current context)")
	cmd.PersistentFlags().BoolVar(&target.allowNonLocal, "allow-non-local-context", false, "allow a context without kind- or k3d- prefix after confirming it is safe")
}

type runOptions struct {
	filePath         string
	sourceDir        string
	sourceMount      string
	sourceIncludes   []string
	sourceExcludes   []string
	maxSourceBytes   int64
	artifactsDir     string
	maxArtifactBytes int64
	config           []string
	variables        []string
	autoContinue     bool
	keep             bool
	dryRun           bool
}

func newRunCmd(target *targetOptions) *cobra.Command {
	options := &runOptions{}
	cmd := &cobra.Command{
		Use:   "run --file <testworkflow.yaml>",
		Short: "Execute a local TestWorkflow without a Testkube API or CRD",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("max-artifact-bytes") && options.maxArtifactBytes <= 0 {
				return localrunner.UsageError("--max-artifact-bytes must be greater than zero")
			}
			config, err := parseAssignments(options.config, "--config")
			if err != nil {
				return err
			}
			variables, err := parseAssignments(options.variables, "--variable")
			if err != nil {
				return err
			}
			interactive := isTerminal(cmd.InOrStdin()) && isTerminal(cmd.OutOrStdout())
			_, err = localrunner.Run(cmd.Context(), localrunner.Options{
				FilePath:         options.filePath,
				SourceDir:        options.sourceDir,
				SourceMount:      options.sourceMount,
				SourceIncludes:   options.sourceIncludes,
				SourceExcludes:   options.sourceExcludes,
				MaxSourceBytes:   options.maxSourceBytes,
				ArtifactsDir:     options.artifactsDir,
				MaxArtifactBytes: options.maxArtifactBytes,
				Config:           config,
				Variables:        variables,
				Namespace:        target.namespace,
				Kubeconfig:       target.kubeconfig,
				ContextName:      target.contextName,
				AllowNonLocal:    target.allowNonLocal,
				Interactive:      interactive,
				AutoContinue:     options.autoContinue,
				Keep:             options.keep,
				DryRun:           options.dryRun,
				In:               cmd.InOrStdin(),
				Out:              cmd.OutOrStdout(),
				ErrOut:           cmd.ErrOrStderr(),
			})
			return err
		},
	}
	cmd.Flags().StringVar(&options.filePath, "file", "", "path to one TestWorkflow YAML file")
	cmd.Flags().StringVar(&options.sourceDir, "source", "", "directory of uncommitted source to relay into the workflow")
	cmd.Flags().StringVar(&options.sourceMount, "source-mount", "", "absolute POSIX path where local source is extracted")
	cmd.Flags().StringArrayVar(&options.sourceIncludes, "source-include", nil, "Testkube-ignore pattern to include (may be repeated)")
	cmd.Flags().StringArrayVar(&options.sourceExcludes, "source-exclude", nil, "Testkube-ignore pattern to exclude (may be repeated)")
	cmd.Flags().Int64Var(&options.maxSourceBytes, "max-source-bytes", localrunner.DefaultMaxSourceBytes, "maximum uncompressed local source bytes")
	cmd.Flags().StringVar(&options.artifactsDir, "artifacts-dir", "", "host directory for private local workflow artifacts")
	cmd.Flags().Int64Var(&options.maxArtifactBytes, "max-artifact-bytes", localrunner.DefaultMaxArtifactBytes, "maximum total local artifact-export bytes")
	cmd.Flags().StringArrayVar(&options.config, "config", nil, "non-sensitive workflow configuration key=value (may be repeated)")
	cmd.Flags().StringArrayVarP(&options.variables, "variable", "v", nil, "runtime variable key=value (may be repeated)")
	cmd.Flags().BoolVar(&options.autoContinue, "auto-continue", false, "continue paused workflow steps automatically")
	cmd.Flags().BoolVar(&options.keep, "keep", false, "keep exact local-run resources after the command exits")
	cmd.Flags().BoolVar(&options.dryRun, "dry-run", false, "validate inputs and kubeconfig without creating Kubernetes resources")
	return cmd
}

func newPauseCmd(target *targetOptions) *cobra.Command {
	return newControlCmd("pause", "Pause an active local TestWorkflow", target, func(ctx context.Context, control *localrunner.LocalControl, runID string, _ *cobra.Command) error {
		return control.Pause(ctx, runID)
	})
}

func newResumeCmd(target *targetOptions) *cobra.Command {
	return newControlCmd("resume", "Resume a paused local TestWorkflow", target, func(ctx context.Context, control *localrunner.LocalControl, runID string, _ *cobra.Command) error {
		return control.Resume(ctx, runID)
	})
}

func newShellCmd(target *targetOptions) *cobra.Command {
	return newControlCmd("shell", "Open the TestWorkflow-provided shell in an active local workflow", target, func(ctx context.Context, control *localrunner.LocalControl, runID string, cmd *cobra.Command) error {
		return control.Shell(ctx, runID, localrunner.IOStreams{
			In:  cmd.InOrStdin(),
			Out: cmd.OutOrStdout(),
			Err: cmd.ErrOrStderr(),
			TTY: isTerminal(cmd.InOrStdin()) && isTerminal(cmd.OutOrStdout()),
		})
	})
}

type controlOperation func(context.Context, *localrunner.LocalControl, string, *cobra.Command) error

func newControlCmd(name, description string, target *targetOptions, operation controlOperation) *cobra.Command {
	return &cobra.Command{
		Use:   name + " <run-id>",
		Short: description,
		Args:  exactlyOneRunID,
		RunE: func(cmd *cobra.Command, args []string) error {
			control, err := loadControl(target)
			if err != nil {
				return err
			}
			return operation(cmd.Context(), control, args[0], cmd)
		},
	}
}

func newCleanCmd(target *targetOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "clean <run-id>",
		Short: "Remove only Kubernetes resources labelled for one local run",
		Args:  exactlyOneRunID,
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := localrunner.ResolveKubeTarget(target.kubeconfig, target.contextName, target.allowNonLocal)
			if err != nil {
				return err
			}
			if err = localrunner.ValidateNamespace(target.namespace); err != nil {
				return err
			}
			client, err := kubernetes.NewForConfig(selection.RESTConfig)
			if err != nil {
				return localrunner.ExecutionError("create Kubernetes client for %s: %v", localrunner.KubeTargetDescription(selection), err)
			}
			manager := localrunner.NewResourceManager(client, target.namespace)
			if err = manager.Clean(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "cleanup: removed exact local resources for %s\n", args[0])
			return nil
		},
	}
}

func loadControl(options *targetOptions) (*localrunner.LocalControl, error) {
	if err := localrunner.ValidateNamespace(options.namespace); err != nil {
		return nil, err
	}
	target, err := localrunner.ResolveKubeTarget(options.kubeconfig, options.contextName, options.allowNonLocal)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(target.RESTConfig)
	if err != nil {
		return nil, localrunner.ExecutionError("create Kubernetes client for %s: %v", localrunner.KubeTargetDescription(target), err)
	}
	return localrunner.NewLocalControl(client, target.RESTConfig, options.namespace), nil
}

func parseAssignments(values []string, flag string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, contents, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, localrunner.UsageError("%s expects key=value, got %q", flag, value)
		}
		if _, exists := result[key]; exists {
			return nil, localrunner.UsageError("%s repeats key %q", flag, key)
		}
		result[key] = contents
	}
	return result, nil
}

func noArgs(_ *cobra.Command, args []string) error {
	if len(args) != 0 {
		return localrunner.UsageError("local run does not accept positional arguments")
	}
	return nil
}

func exactlyOneRunID(_ *cobra.Command, args []string) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return localrunner.UsageError("expected exactly one local run ID")
	}
	if _, err := localrunner.Labels(args[0], "workflow"); err != nil {
		return err
	}
	return nil
}

// isTerminal supports Cobra's normal os.File streams while remaining false for
// injected buffers in tests and CI, which makes pause validation deterministic.
func isTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}
