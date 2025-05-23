package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/pkg/ui"
)

func init() {
	// New commands
	RootCmd.AddCommand(NewDnsCmd())
	RootCmd.AddCommand(NewTLSCmd())
}

var RootCmd = &cobra.Command{
	Use:   "diagnostics",
	Short: "Testkube entrypoint for kubectl plugin",

	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		err := cmd.Usage()
		ui.PrintOnError("Displaying usage", err)
		cmd.DisableAutoGenTag = true
	},
}

func Execute() {
	// Run services within an errgroup to propagate errors between services.
	g, ctx := errgroup.WithContext(context.Background())

	// Cancel the errgroup context on SIGINT and SIGTERM,
	// which shuts everything down gracefully.
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	g.Go(func() error {
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

	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
