package config

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:generate mockgen -destination=./mock_repository.go -package=config "github.com/kubeshop/testkube/pkg/repository/config" Repository
type Repository interface {
	// GetUniqueClusterId gets unique cluster based ID
	GetUniqueClusterId(ctx context.Context) (string, error)

	// GetTelemetryEnabled get telemetry enabled
	GetTelemetryEnabled(ctx context.Context) (ok bool, err error)

	// Get gets execution result by id
	Get(ctx context.Context) (testkube.Config, error)

	// Upsert inserts record if not exists, updates otherwise
	Upsert(ctx context.Context, config testkube.Config) (testkube.Config, error)
}
