// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/transfer"
)

func NewTransferCmd() *cobra.Command {
	var (
		addr string
	)

	cmd := &cobra.Command{
		Use:   "transfer <idPathPairs...>",
		Short: "Read files from the transfer server",
		Args:  cobra.MinimumNArgs(1),

		Run: func(cmd *cobra.Command, idPathPairs []string) {
			if addr == "" {
				fmt.Println("--addr is required")
				os.Exit(1)
			}
			client := transfer.NewClient(addr)

			for _, pair := range idPathPairs {
				mountPath, id, found := strings.Cut(pair, "=")
				if mountPath == "" || id == "" || !found {
					fmt.Printf("invalid pair: %s\n", pair)
					os.Exit(1)
				}

				fmt.Printf("Writing into %s...\n", mountPath)
				err := client.Fetch(id, mountPath)
				if err != nil {
					fmt.Printf("failed to transfer %s into %s: %s\n", id, mountPath, err.Error())
					os.Exit(1)
				}
			}
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "", "transfer server to take data from")

	return cmd
}
