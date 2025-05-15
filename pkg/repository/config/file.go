package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

func NewFileConfig(filePath string) (*FileConfig, error) {
	return &FileConfig{filePath: filePath}, nil
}

// FileConfig contains configmap config properties
type FileConfig struct {
	filePath string
	data     *testkube.Config
	mu       sync.Mutex
}

func (c *FileConfig) getDefaultClusterId() string {
	return fmt.Sprintf("cluster%s", telemetry.GetMachineID())
}

// GetUniqueClusterId gets unique cluster based ID
func (c *FileConfig) GetUniqueClusterId(_ context.Context) (clusterId string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		return "", errors.New("config not loaded yet")
	}
	return c.data.ClusterId, nil
}

// GetTelemetryEnabled get telemetry enabled
func (c *FileConfig) GetTelemetryEnabled(_ context.Context) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		return false, errors.New("config not loaded yet")
	}
	return c.data.EnableTelemetry, nil
}

// Get config
func (c *FileConfig) Get(_ context.Context) (result testkube.Config, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		return result, errors.New("config not loaded yet")
	}
	return *c.data, nil
}

func (c *FileConfig) load(ctx context.Context, defaultTelemetryEnabled bool) error {
	// Load configuration from the File
	var data map[string]string
	content, _ := os.ReadFile(c.filePath)
	if len(content) > 0 {
		_ = json.Unmarshal(content, &data)
	}
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

func (c *FileConfig) Load(ctx context.Context, defaultTelemetryEnabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.load(ctx, defaultTelemetryEnabled)
}

func (c *FileConfig) upsert(_ context.Context, result testkube.Config) (updated testkube.Config, err error) {
	c.data = &testkube.Config{
		ClusterId:       result.ClusterId,
		EnableTelemetry: result.EnableTelemetry,
	}
	if c.data.ClusterId == "" {
		c.data.ClusterId = c.getDefaultClusterId()
	}
	serializedData, _ := json.Marshal(map[string]string{
		"clusterId":       c.data.ClusterId,
		"enableTelemetry": fmt.Sprint(c.data.EnableTelemetry),
	})
	if err = os.WriteFile(c.filePath, serializedData, 0644); err != nil {
		return result, errors.Wrap(err, "writing config map error")
	}

	return result, err
}

// Upsert inserts record if not exists, updates otherwise
func (c *FileConfig) Upsert(ctx context.Context, result testkube.Config) (updated testkube.Config, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.upsert(ctx, result)
}
