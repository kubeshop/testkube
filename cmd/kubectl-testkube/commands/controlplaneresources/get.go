package controlplaneresources

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewGetCmds returns a `testkube get <kind> [name]` subcommand per Control
// Plane resource kind.
func NewGetCmds(newClient ClientFactory) []*cobra.Command {
	cmds := make([]*cobra.Command, 0, len(declarative.Kinds()))
	for _, kind := range declarative.Kinds() {
		cmds = append(cmds, newGetCmd(kind, newClient))
	}
	return cmds
}

func newGetCmd(kind declarative.Kind, newClient ClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:     kind.Name() + " [name]",
		Aliases: kind.Names[1:],
		Short:   fmt.Sprintf("Get %s resources from the Control Plane", kind.Kind),
		Long: fmt.Sprintf(`Get one %[1]s or list them all. With -o yaml or -o json the output is in the
form "testkube apply" takes, so what is built in the dashboard can be exported and kept in Git.`, kind.Kind),
		Example: fmt.Sprintf("  testkube get %[1]s\n  testkube get %[1]s <name> -o yaml > %[1]s.yaml", kind.Name()),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			output := cmd.Flag("output").Value.String()
			c, err := newClient(cmd)
			ui.ExitOnError("connecting to the Control Plane", err)
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			ui.ExitOnError("getting "+kind.Name(), Get(cmd.Context(), c, kind, name, output, cmd.OutOrStdout()))
		},
	}
}

// Get prints the named resource, or every resource of the kind when name is
// empty, as yaml, json or a table ("pretty").
func Get(ctx context.Context, c client.Client, kind declarative.Kind, name, output string, out io.Writer) error {
	if output != "pretty" && output != "yaml" && output != "json" {
		return fmt.Errorf("unsupported output %q (use pretty, yaml or json)", output)
	}
	var objects []declarative.Object
	if name == "" {
		list, err := c.List(ctx, kind)
		if err != nil {
			return err
		}
		objects = list
	} else {
		obj, err := c.Get(ctx, kind, name)
		if err != nil {
			return err
		}
		objects = []declarative.Object{obj}
	}
	if output == "pretty" {
		ui.Table(table(objects), out)
		return nil
	}
	return write(out, objects, output)
}

type table []declarative.Object

func (t table) Table() ([]string, [][]string) {
	rows := make([][]string, 0, len(t))
	for _, obj := range t {
		status := declarative.StatusOf(obj)
		owner := ""
		if status.GitOpsOwner != nil {
			owner = status.GitOpsOwner.AgentName
			if owner == "" {
				owner = status.GitOpsOwner.AgentID
			}
		}
		rows = append(rows, []string{obj.GetName(), status.ID, owner, status.CreationSource, status.LastModificationSource})
	}
	return []string{"Name", "ID", "GitOps owner", "Created by", "Last modified by"}, rows
}
