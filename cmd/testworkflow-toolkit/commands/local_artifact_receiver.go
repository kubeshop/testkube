package commands

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/pkg/testworkflows/localartifacts"
)

const defaultLocalArtifactReceiverMaxBytes int64 = 100 << 20

// NewLocalArtifactReceiverCmd creates the private local-run artifact relay.
// It is hidden because only testkube local owns the receiver's lifecycle and
// Secret; regular TestWorkflow artifact stages continue to use Cloud storage.
func NewLocalArtifactReceiverCmd() *cobra.Command {
	var (
		root     string
		listen   string
		tokenEnv string
		maxBytes int64
	)

	cmd := &cobra.Command{
		Use:    "local-artifact-receiver",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			receiver, err := artifacts.NewLocalReceiver(root, os.Getenv(tokenEnv), maxBytes)
			if err != nil {
				return fmt.Errorf("configure local artifact receiver: %w", err)
			}
			server := &http.Server{
				Addr:              listen,
				Handler:           receiver.Handler(),
				ReadHeaderTimeout: 5 * time.Second,
				IdleTimeout:       time.Minute,
				MaxHeaderBytes:    8 << 10,
			}
			go func() {
				<-cmd.Context().Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdownCtx)
			}()
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("serve local artifact receiver: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&root, "root", "/srv", "artifact storage root")
	cmd.Flags().StringVar(&listen, "listen", ":8080", "HTTP listen address")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", defaultLocalArtifactReceiverMaxBytes, "maximum total artifact bytes")
	cmd.Flags().StringVar(&tokenEnv, "token-env", localartifacts.TokenEnvName, "environment variable containing the relay token")
	_ = cmd.Flags().MarkHidden("token-env")

	return cmd
}
