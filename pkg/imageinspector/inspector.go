package imageinspector

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/log"
)

type inspector struct {
	defaultRegistry string
	fetcher         InfoFetcher
	secrets         SecretFetcher
	storage         []Storage
}

func NewInspector(defaultRegistry string, infoFetcher InfoFetcher, secretFetcher SecretFetcher, storage ...Storage) Inspector {
	return &inspector{
		defaultRegistry: defaultRegistry,
		fetcher:         infoFetcher,
		secrets:         secretFetcher,
		storage:         storage,
	}
}

func (i *inspector) get(ctx context.Context, registry, image string) *Info {
	for _, s := range i.storage {
		v, err := s.Get(ctx, RequestBase{Registry: registry, Image: image})
		if err != nil && !errors.Is(err, context.Canceled) {
			log.DefaultLogger.Warnw("error while getting image details from cache", "registry", registry, "image", image, "error", err)
		}
		if v != nil {
			return v
		}
	}
	return nil
}

func (i *inspector) fetch(ctx context.Context, registry, image string, pullSecretNames []string) (*Info, error) {
	// Fetch the secrets
	secrets := make([]corev1.Secret, len(pullSecretNames))
	for idx, name := range pullSecretNames {
		secret, err := i.secrets.Get(ctx, name)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("fetching '%s' pull secret", name))
		}
		secrets[idx] = *secret
	}

	// Load the image details
	info, err := i.fetcher.Fetch(ctx, registry, image, secrets)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("fetching '%s' image from '%s' registry", image, registry))
	} else if info == nil {
		return nil, fmt.Errorf("unknown problem with fetching '%s' image from '%s' registry", image, registry)
	}
	if info.Shell != "" && !filepath.IsAbs(info.Shell) {
		info.Shell = ""
	}
	return info, err
}

func (i *inspector) save(ctx context.Context, registry, image string, info *Info) {
	if info == nil {
		return
	}
	for _, s := range i.storage {
		if err := s.Store(ctx, RequestBase{Registry: registry, Image: image}, *info); err != nil {
			log.DefaultLogger.Warnw("error while saving image details in the cache", "registry", registry, "image", image, "error", err)
		}
	}
}

func (i *inspector) Inspect(ctx context.Context, registry, image string, pullPolicy corev1.PullPolicy, pullSecretNames []string) (*Info, error) {
	// Load from cache
	if pullPolicy != corev1.PullAlways {
		value := i.get(ctx, registry, image)
		if value != nil {
			return value, nil
		}
	}

	// Fetch the data
	value, err := i.fetch(ctx, registry, image, pullSecretNames)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("inspecting image: '%s' at '%s' registry", image, registry))
	}
	if value == nil {
		return nil, fmt.Errorf("not found image details for: '%s' at '%s' registry", image, registry)
	}

	// Save asynchronously
	go i.save(context.Background(), registry, image, value)

	return value, nil
}
