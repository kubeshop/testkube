package config

import (
	"context"
	"fmt"
	"strconv"
	"sync"

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

	data *testkube.Config
	mu   sync.Mutex
}

func (c *ConfigMapConfig) getDefaultClusterId() string {
	return fmt.Sprintf("cluster%s", telemetry.GetMachineID())
}

// GetUniqueClusterId gets unique cluster based ID
func (c *ConfigMapConfig) GetUniqueClusterId(_ context.Context) (clusterId string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		return "", errors.New("config not loaded yet")
	}
	return c.data.ClusterId, nil
}

// GetTelemetryEnabled get telemetry enabled
func (c *ConfigMapConfig) GetTelemetryEnabled(_ context.Context) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		return false, errors.New("config not loaded yet")
	}
	return c.data.EnableTelemetry, nil
}

// Get config
func (c *ConfigMapConfig) Get(_ context.Context) (result testkube.Config, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		return result, errors.New("config not loaded yet")
	}
	return *c.data, nil
}

func (c *ConfigMapConfig) load(ctx context.Context, defaultTelemetryEnabled bool) error {
	// Load configuration from the ConfigMap
	data, _ := c.client.Get(ctx, c.name)
	c.data = &testkube.Config{}
	if len(data) > 0 {
		c.data.ClusterId = data["clusterId"]
		if enableTelemetry, ok := data["enableTelemetry"]; ok {
			c.data.EnableTelemetry, _ = strconv.ParseBool(enableTelemetry)
		} else {
			c.data.EnableTelemetry = defaultTelemetryEnabled
		}
	}

	// Create new configuration if it doesn't exist
	if c.data.ClusterId != "" {
		c.data.ClusterId = c.getDefaultClusterId()
		c.data.EnableTelemetry = defaultTelemetryEnabled
		_, err := c.upsert(ctx, *c.data)
		return err
	}

	return nil
}

func (c *ConfigMapConfig) Load(ctx context.Context, defaultTelemetryEnabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.load(ctx, defaultTelemetryEnabled)
}

func (c *ConfigMapConfig) upsert(ctx context.Context, result testkube.Config) (updated testkube.Config, err error) {
	c.data = &testkube.Config{
		ClusterId:       result.ClusterId,
		EnableTelemetry: result.EnableTelemetry,
	}
	if c.data.ClusterId == "" {
		c.data.ClusterId = c.getDefaultClusterId()
	}
	data := map[string]string{
		"clusterId":       c.data.ClusterId,
		"enableTelemetry": fmt.Sprint(c.data.EnableTelemetry),
	}
	if err = c.client.Apply(ctx, c.name, data); err != nil {
		return result, errors.Wrap(err, "writing config map error")
	}

	return result, err
}

// Upsert inserts record if not exists, updates otherwise
func (c *ConfigMapConfig) Upsert(ctx context.Context, result testkube.Config) (updated testkube.Config, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.upsert(ctx, result)
}
