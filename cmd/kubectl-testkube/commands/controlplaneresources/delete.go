package controlplaneresources

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDeleteCmds returns a `testkube delete <kind> <name>` subcommand per
// Control Plane resource kind.
func NewDeleteCmds(newClient ClientFactory) []*cobra.Command {
	cmds := make([]*cobra.Command, 0, len(declarative.Kinds()))
	for _, kind := range declarative.Kinds() {
		cmds = append(cmds, newDeleteCmd(kind, newClient))
	}
	return cmds
}

func newDeleteCmd(kind declarative.Kind, newClient ClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:     kind.Name() + " <name>",
		Aliases: kind.Names[1:],
		Short:   fmt.Sprintf("Delete %s resources from the Control Plane", kind.Kind),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ignoreNotFound := false
			if flag := cmd.Flag("ignore-not-found"); flag != nil {
				ignoreNotFound = flag.Value.String() == "true"
			}
			c, err := newClient(cmd)
			ui.ExitOnError("connecting to the Control Plane", err)
			ui.ExitOnError("deleting "+kind.Name(), Delete(cmd.Context(), c, kind, args[0], ignoreNotFound, cmd.OutOrStdout()))
		},
	}
}

// Delete removes the named resource. With ignoreNotFound, a resource that
// does not exist is not an error.
func Delete(ctx context.Context, c client.Client, kind declarative.Kind, name string, ignoreNotFound bool, out io.Writer) error {
	err := c.Delete(ctx, kind, name)
	if errors.Is(err, client.ErrNotFound) && ignoreNotFound {
		fmt.Fprintf(out, "%s/%s not found\n", kind.Name(), name)
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s/%s deleted\n", kind.Name(), name)
	return nil
}
