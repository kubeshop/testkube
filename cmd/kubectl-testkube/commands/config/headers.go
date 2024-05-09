package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewConfigureHeadersCmd is headers config command
func NewConfigureHeadersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "headers <name1=value1> <name2=value2> ...",
		Short: "Set headers for testkube client",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)
			headers := make(map[string]string)
			for _, arg := range args {
				values := strings.SplitN(arg, "=", 2)
				if len(values) != 2 {
					ui.ExitOnError("parsing headers", fmt.Errorf("wrong value %s", arg))
				}

				headers[values[0]] = values[1]
			}

			cfg.Headers = headers
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New headers set to", testkube.MapToString(cfg.Headers))
		},
	}

	return cmd
}
