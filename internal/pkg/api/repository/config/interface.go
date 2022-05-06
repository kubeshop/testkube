package config

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Repository interface {
	// GetUniqueClusterId gets unique cluster based ID
	GetUniuqueClusterId(ctx context.Context) (string, error)

	// Get gets execution result by id
	Get(ctx context.Context) (testkube.Config, error)

	// Upserts inserts record if not exists, updates otherwise
	Upsert(ctx context.Context, config testkube.Config) (testkube.Config, error)
}
