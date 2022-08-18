package config

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

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

type ConfigMapConfig struct {
	name   string
	client *configmap.Client
}

func (c *ConfigMapConfig) GetUniqueClusterId(ctx context.Context) (clusterId string, err error) {
	config, err := c.Get(ctx)
	// generate new cluster Id and save if there is not already
	if config.ClusterId == "" {
		return fmt.Sprintf("cluster%s", telemetry.GetMachineID()), err
	}

	return config.ClusterId, nil
}

func (c *ConfigMapConfig) GetTelemetryEnabled(ctx context.Context) (ok bool, err error) {
	config, err := c.Get(ctx)
	if err != nil {
		return true, err
	}

	return config.EnableTelemetry, nil
}

func (c *ConfigMapConfig) Get(ctx context.Context) (result testkube.Config, err error) {
	data, err := c.client.Get(c.name)
	if err != nil {
		return result, fmt.Errorf("reading config map error: %w", err)
	}

	result.ClusterId = data["clusterId"]
	if enableTelemetry, ok := data["enableTelemetry"]; ok {
		result.EnableTelemetry, err = strconv.ParseBool(enableTelemetry)
		if err != nil {
			return result, fmt.Errorf("parsing enable telemetry error: %w", err)
		}
	}

	return
}

func (c *ConfigMapConfig) Upsert(ctx context.Context, result testkube.Config) (err error) {
	data := map[string]string{
		"clusterId":       result.ClusterId,
		"enableTelemetry": fmt.Sprint(result.EnableTelemetry),
	}
	if err = c.client.Apply(c.name, data); err != nil {
		return fmt.Errorf("writing config map error: %w", err)
	}

	return
}
