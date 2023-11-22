package config

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type GetUniqueClusterIdRequest struct{}

type GetUniqueClusterIdResponse struct {
	ClusterID string `json:"clusterId"`
}

type GetTelemetryEnabledRequest struct{}

type GetTelemetryEnabledResponse struct {
	Enabled bool `json:"enabled"`
}

type GetRequest struct{}

type GetResponse struct {
	Config testkube.Config `json:"config"`
}

type UpsertRequest struct {
	Config testkube.Config `json:"config"`
}

type UpsertResponse struct {
	Config testkube.Config `json:"config"`
}
