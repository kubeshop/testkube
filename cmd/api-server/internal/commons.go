package internal

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
)

func exitOnError(title string, err error) {
	if err != nil {
		log.DefaultLogger.Errorw(title, "error", err)
		os.Exit(1)
	}
}

func MustGetConfig() *config.Config {
	cfg, err := config.Get()
	exitOnError("error getting application config", err)
	cfg.CleanLegacyVars()
	return cfg
}

func MustGetFeatureFlags() featureflags.FeatureFlags {
	features, err := featureflags.Get()
	exitOnError("error getting application feature flags", err)
	log.DefaultLogger.Infow("Feature flags configured", "ff", features)
	return features
}

func MustFreePort(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	exitOnError("Checking if port "+port+" is free", err)
	_ = ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", port)
}

func MustGetConfigMapConfig(ctx context.Context, name string, namespace string, defaultTelemetryEnabled bool) *configRepo.ConfigMapConfig {
	if name == "" {
		name = fmt.Sprintf("testkube-api-server-config-%s", namespace)
	}
	configMapConfig, err := configRepo.NewConfigMapConfig(name, namespace)
	exitOnError("Getting config map config", err)

	// Load the initial data
	err = configMapConfig.Load(ctx, defaultTelemetryEnabled)
	if err != nil {
		log.DefaultLogger.Warn("error upserting config ConfigMap", "error", err)
	}
	return configMapConfig
}

func GetEnvironmentVariables() map[string]string {
	list := os.Environ()
	envs := make(map[string]string, len(list))
	for _, env := range list {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		envs[pair[0]] += pair[1]
	}
	return envs
}

func HandleCancelSignal(g *errgroup.Group, ctx context.Context) {
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
			// Returning an error cancels the errgroup.
			return fmt.Errorf("received signal: %v", sig)
		}
	})
}
