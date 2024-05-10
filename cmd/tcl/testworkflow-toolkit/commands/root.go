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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"

	"golang.org/x/sync/errgroup"
)

func init() {
	RootCmd.AddCommand(NewCloneCmd())
	RootCmd.AddCommand(NewTarballCmd())
	RootCmd.AddCommand(NewExecuteCmd())
	RootCmd.AddCommand(NewArtifactsCmd())
	RootCmd.AddCommand(NewParallelCmd())
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

	RootCmd.PersistentFlags().BoolVar(&env.UseProxyValue, "proxy", false, "use Kubernetes proxy for TK access")

	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
