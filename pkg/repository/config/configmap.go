package config

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

// NewConfigMapConfig is a constructor for configmap config
func NewConfigMapConfig(name, namespace string) (*ConfigMapConfig, error) {
	client, err := configmap.NewClient(namespace)
	if err != nil {
		return nil, err
	}

	return &ConfigMapConfig{
		name:   name,
		client: client,
	}, nil
}

// ConfigMapConfig contains configmap config properties
type ConfigMapConfig struct {
	name   string
	client *configmap.Client
}

// GetUniqueClusterId gets unique cluster based ID
func (c *ConfigMapConfig) GetUniqueClusterId(ctx context.Context) (clusterId string, err error) {
	config, err := c.Get(ctx)
	// generate new cluster Id
	if config.ClusterId == "" {
		return fmt.Sprintf("cluster%s", telemetry.GetMachineID()), err
	}

	return config.ClusterId, nil
}

// GetTelemetryEnabled get telemetry enabled
func (c *ConfigMapConfig) GetTelemetryEnabled(ctx context.Context) (ok bool, err error) {
	config, err := c.Get(ctx)
	if err != nil {
		return true, err
	}

	return config.EnableTelemetry, nil
}

// Get config
func (c *ConfigMapConfig) Get(ctx context.Context) (result testkube.Config, err error) {
	data, err := c.client.Get(ctx, c.name)
	if err != nil {
		return result, errors.Wrap(err, "reading config map error")
	}

	result.ClusterId = data["clusterId"]
	if enableTelemetry, ok := data["enableTelemetry"]; ok {
		result.EnableTelemetry, err = strconv.ParseBool(enableTelemetry)
		if err != nil {
			return result, errors.Wrap(err, "parsing enable telemetry error")
		}
	}

	return
}

// Upsert inserts record if not exists, updates otherwise
func (c *ConfigMapConfig) Upsert(ctx context.Context, result testkube.Config) (updated testkube.Config, err error) {
	data := map[string]string{
		"clusterId":       result.ClusterId,
		"enableTelemetry": fmt.Sprint(result.EnableTelemetry),
	}
	if err = c.client.Apply(ctx, c.name, data); err != nil {
		return result, errors.Wrap(err, "writing config map error")
	}

	return result, err
}
