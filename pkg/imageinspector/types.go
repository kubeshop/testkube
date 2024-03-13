package imageinspector

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
)

//go:generate mockgen -destination=./mock_inspector.go -package=imageinspector "github.com/kubeshop/testkube/pkg/imageinspector" Inspector
type Inspector interface {
	Inspect(ctx context.Context, registry, image string, pullPolicy corev1.PullPolicy, pullSecretNames []string) (*Info, error)
}

type StorageTransfer interface {
	StoreMany(ctx context.Context, data map[Hash]Info) error
	CopyTo(ctx context.Context, other ...StorageTransfer) error
}

type Storage interface {
	Store(ctx context.Context, request RequestBase, info Info) error
	Get(ctx context.Context, request RequestBase) (*Info, error)
}

//go:generate mockgen -destination=./mock_storage.go -package=imageinspector "github.com/kubeshop/testkube/pkg/imageinspector" StorageWithTransfer
type StorageWithTransfer interface {
	StorageTransfer
	Storage
}

//go:generate mockgen -destination=./mock_secretfetcher.go -package=imageinspector "github.com/kubeshop/testkube/pkg/imageinspector" SecretFetcher
type SecretFetcher interface {
	Get(ctx context.Context, name string) (*corev1.Secret, error)
}

//go:generate mockgen -destination=./mock_infofetcher.go -package=imageinspector "github.com/kubeshop/testkube/pkg/imageinspector" InfoFetcher
type InfoFetcher interface {
	Fetch(ctx context.Context, registry, image string, pullSecrets []corev1.Secret) (*Info, error)
}

type Info struct {
	FetchedAt  time.Time `json:"a,omitempty"`
	Entrypoint []string  `json:"e,omitempty"`
	Cmd        []string  `json:"c,omitempty"`
	Shell      string    `json:"s,omitempty"`
	WorkingDir string    `json:"w,omitempty"`
	User       int64     `json:"u,omitempty"`
	Group      int64     `json:"g,omitempty"`
}

type RequestBase struct {
	Image    string
	Registry string
}

type Request struct {
	RequestBase
	PullPolicy corev1.PullPolicy
}
