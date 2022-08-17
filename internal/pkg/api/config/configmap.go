package config

import (
	"os"
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

func NewConfigMap(path string) *ConfigMap {
	return &ConfigMap{
		path: path,
	}
}

type ConfigMap struct {
	path string
}

func (c *ConfigMap) GetUniqueClusterId(ctx context.Context) (clusterId string, err error) {
	config, err := c.Get(ctx)
	if err != nil {
		return clusterId, err
	}

	// generate new cluster Id and save if there is not already
	if config.ClusterId == "" {
		config.ClusterId = fmt.Sprintf("cluster%s", telemetry.GetMachineID())
		err := c.Upsert(ctx, config)
		return config.ClusterId, err
	}

	return config.ClusterId, nil
}

func (c *ConfigMap) GetTelemetryEnabled(ctx context.Context) (ok bool, err error) {
	config, err := c.Get(ctx)
	return config.EnableTelemetry, err
}

func (c *ConfigMap) Get(ctx context.Context) (result testkube.Config, err error) {
	data, err := os.ReadFile(filepath.Join(c.path, "clusterId"))
	if err != nil {
		return result, fmt.Errorf("reading cluster id error: %w", err)
	}
	result.ClusterId = string(data)

	data, err = os.ReadFile(filepath.Join(c.path, "enableTelemetry"))
	if err != nil {
		return result, fmt.Errorf("reading enable telemetry error: %w", err)
	}
	if len(data) != 0  {
		result.EnableTelemetry, err = strconv.ParseBool(string(data))
		if err != nil {
			return result, fmt.Errorf("parsing enable telemetry error: %w", err)
		}
	}

	return
}

func (c *ConfigMap) Upsert(ctx context.Context, result testkube.Config) (err error) {
	if err = os.WriteFile(filepath.Join(c.path, "clusterId"), []byte(result.ClusterId), 0666); err != nil {
		return fmt.Errorf("writing cluster id error: %w", err)
	}

	if err = os.WriteFile(filepath.Join(c.path, "enableTelemetry"), []byte(fmt.Sprint(result.EnableTelemetry)), 0666); err != nil {
		return fmt.Errorf("writing enable telemetry error: %w", err)
	}	

	return
}
