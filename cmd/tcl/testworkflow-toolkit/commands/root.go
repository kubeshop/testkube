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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"

	"golang.org/x/sync/errgroup"
)

func init() {
	RootCmd.AddCommand(NewCloneCmd())
	RootCmd.AddCommand(NewTarballCmd())
	RootCmd.AddCommand(NewTransferCmd())
	RootCmd.AddCommand(NewExecuteCmd())
	RootCmd.AddCommand(NewArtifactsCmd())
	RootCmd.AddCommand(NewParallelCmd())
	RootCmd.AddCommand(NewServicesCmd())
	RootCmd.AddCommand(NewKillCmd())
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

func preExecute() {
	cfg := env.Config()

	if cfg.Execution.IstioProxyWait {
		fmt.Println("Waiting for Istio's proxy to become ready...")
		for {
			resp, err := http.Head("http://localhost:15021/healthz/ready")
			if err == nil && resp.StatusCode == http.StatusOK {
				fmt.Println("Istio's proxy is ready")
				break
			}
			fmt.Println("Still waiting for Istio's proxy to become ready...")
			time.Sleep(3 * time.Second)
		}
	}
}

func postExecute(exitCode int) {
	cfg := env.Config()

	if cfg.Execution.IstioProxyExit {
		fmt.Println("Sending exit signal to Istio's proxy...")
		for {
			resp, err := http.Post("http://localhost:15020/quitquitquit", "", nil)
			if err == nil && resp.StatusCode == http.StatusOK {
				fmt.Println("Sent exit signal to Istio's proxy")
				break
			}
			fmt.Println("Still sending exit signal to Istio's proxy...")
			time.Sleep(3 * time.Second)
		}
	}

	os.Exit(exitCode)
}

func Execute() {
	// Run pre-execute hooks
	preExecute()

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
		postExecute(1)
	}

	postExecute(0)
}
