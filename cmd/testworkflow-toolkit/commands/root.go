package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"

	"golang.org/x/sync/errgroup"
)

func init() {
	RootCmd.AddCommand(NewCloneCmd())
	RootCmd.AddCommand(NewTarballCmd())
	RootCmd.AddCommand(NewTransferCmd())
	RootCmd.AddCommand(NewArtifactsCmd())

	// Pro functionalities
	RootCmd.AddCommand(commands.NewExecuteCmd())
	RootCmd.AddCommand(commands.NewParallelCmd())
	RootCmd.AddCommand(commands.NewServicesCmd())
	RootCmd.AddCommand(commands.NewKillCmd())
}

var RootCmd = &cobra.Command{
	Use:   "testworkflow-toolkit",
	Short: "Orchestrating Testkube workflows",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd:   true,
		DisableNoDescFlag:   true,
		DisableDescriptions: true,
		HiddenDefaultCmd:    true,
	},
}

func Execute() {
	// Run services within an errgroup to propagate errors between services.
	g, ctx := errgroup.WithContext(context.Background())

	// Cancel the errgroup context on SIGINT and SIGTERM,
	// which shuts everything down gracefully.
	// Kill on recurring signal.
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	g.Go(func() error {
		defer signal.Stop(stopSignal)
		select {
		case <-ctx.Done():
			return nil
		case sig := <-stopSignal:
			go func() {
				<-stopSignal
				os.Exit(137)
			}()
			return errors.Errorf("received signal: %v", sig)
		}
	})

	RootCmd.PersistentFlags().BoolVar(&config.UseProxyValue, "proxy", false, "use Kubernetes proxy for TK access")

	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
