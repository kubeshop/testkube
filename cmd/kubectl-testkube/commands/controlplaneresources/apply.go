package controlplaneresources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

// ApplyOptions configure Apply.
type ApplyOptions struct {
	// DryRun validates the objects without sending them.
	DryRun bool
	// Output is "" for a line per object, or yaml or json to print the
	// objects: as stored, or as they would be sent in a dry run.
	Output string
}

// NewApplyCmd returns `testkube apply`, which connects with newClient.
func NewApplyCmd(newClient ClientFactory) *cobra.Command {
	var (
		files []string
		opts  ApplyOptions
	)
	cmd := &cobra.Command{
		Use:   "apply -f <file|dir|->",
		Short: "Apply Control Plane resources from YAML",
		Long: `Create or update Control Plane resources from YAML or JSON.

Each file may hold several documents. A directory applies every .yaml, .yml and .json file in it, and "-" reads from stdin.
Every document is validated locally before anything is sent; nothing is sent if any document is invalid.
Use --dry-run to validate without sending anything; with -o yaml it prints the objects as they would be sent, defaults filled in.`,
		Example: `  testkube apply -f board.yaml
  testkube apply -f boards/ --dry-run -o yaml
  testkube get board release-health -o yaml | testkube apply -f -`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			if len(files) == 0 {
				ui.Failf("at least one -f <file|dir|-> is required")
			}
			objects, err := ReadInputs(files, cmd.InOrStdin())
			ui.ExitOnError("reading input", err)

			var c client.Client
			if !opts.DryRun {
				c, err = newClient(cmd)
				ui.ExitOnError("connecting to the Control Plane", err)
			}
			ui.ExitOnError("applying", Apply(cmd.Context(), c, objects, opts, cmd.OutOrStdout()))
		},
	}
	cmd.Flags().StringArrayVarP(&files, "filename", "f", nil, "file, directory or - for stdin; repeatable")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "validate without sending anything")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "print the applied objects: yaml|json")
	return cmd
}

// ReadInputs decodes every object of the given files, directories (not
// recursively, in name order) and "-" for stdin.
func ReadInputs(paths []string, stdin io.Reader) ([]declarative.Object, error) {
	var objects []declarative.Object
	read := func(name string, r io.Reader) error {
		decoded, err := declarative.Decode(r)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		objects = append(objects, decoded...)
		return nil
	}
	for _, path := range paths {
		if path == "-" {
			if err := read("stdin", stdin); err != nil {
				return nil, err
			}
			continue
		}
		files, err := expandPath(path)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			if err := read(file, bytes.NewReader(data)); err != nil {
				return nil, err
			}
		}
	}
	if len(objects) == 0 {
		return nil, errors.New("no objects found in the input")
	}
	return objects, nil
}

func expandPath(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !e.IsDir() && (ext == ".yaml" || ext == ".yml" || ext == ".json") {
			files = append(files, filepath.Join(path, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// Apply validates every object, then sends them in order. Nothing is sent
// when any object is invalid. A failure to apply one object does not stop
// the others; every failure is returned together.
func Apply(ctx context.Context, c client.Client, objects []declarative.Object, opts ApplyOptions, out io.Writer) error {
	if opts.Output != "" && opts.Output != "yaml" && opts.Output != "json" {
		return fmt.Errorf("unsupported output %q (use yaml or json)", opts.Output)
	}
	seen := map[string]bool{}
	var invalid []error
	normalized := make([]declarative.Object, 0, len(objects))
	for _, obj := range objects {
		ref := describe(obj)
		if seen[ref] {
			invalid = append(invalid, fmt.Errorf("%s is declared more than once", ref))
			continue
		}
		seen[ref] = true
		// Prepared as the client prepares a PUT, so a dry run prints what
		// would be sent, without the status and cluster metadata of an export.
		copied, err := declarative.PrepareForApply(obj)
		if err != nil {
			invalid = append(invalid, fmt.Errorf("%s: %w", ref, err))
			continue
		}
		if err := declarative.Normalize(copied); err != nil {
			invalid = append(invalid, fmt.Errorf("%s: %w", ref, err))
			continue
		}
		normalized = append(normalized, copied)
	}
	if len(invalid) > 0 {
		return errors.Join(invalid...)
	}

	if opts.DryRun {
		if opts.Output != "" {
			return write(out, normalized, opts.Output)
		}
		for _, obj := range normalized {
			fmt.Fprintf(out, "%s valid (dry run)\n", describe(obj))
		}
		return nil
	}

	var failed []error
	var stored []declarative.Object
	for _, obj := range normalized {
		result, err := c.Apply(ctx, obj)
		if err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", describe(obj), err))
			continue
		}
		stored = append(stored, result)
		if opts.Output == "" {
			fmt.Fprintf(out, "%s applied\n", describe(obj))
		}
	}
	if opts.Output != "" && len(stored) > 0 {
		if err := write(out, stored, opts.Output); err != nil {
			failed = append(failed, err)
		}
	}
	return errors.Join(failed...)
}

func describe(obj declarative.Object) string {
	kind, err := declarative.KindOf(obj)
	if err != nil {
		return obj.GetName()
	}
	return kind.Name() + "/" + obj.GetName()
}

func write(out io.Writer, objects []declarative.Object, format string) error {
	data, err := declarative.Encode(objects, format)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}
