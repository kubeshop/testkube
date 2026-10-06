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

func (i *inspector) rawGet(ctx context.Context, registry, image string) *Info {
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

// resolvedCacheVersion marks the cache key of an image that resolves to another name. Entries
// stored under the resolved name without it can hold the data of the Docker Hub image, because
// the fetch did not use the resolved name, so they must not be read.
const resolvedCacheVersion = "v2"

// cacheKey is the key of the data of an image in the cache.
func (i *inspector) cacheKey(registry, image string) RequestBase {
	if resolvedName := i.ResolveName(registry, image); resolvedName != image {
		return RequestBase{Registry: resolvedCacheVersion, Image: resolvedName}
	}
	return RequestBase{Registry: registry, Image: image}
}

func (i *inspector) get(ctx context.Context, registry, image string) *Info {
	key := i.cacheKey(registry, image)
	return i.rawGet(ctx, key.Registry, key.Image)
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

	// Load the image details. Inspect names the image, so an error here keeps only the cause.
	info, err := i.fetcher.Fetch(ctx, registry, image, secrets)
	if err != nil {
		return nil, err
	} else if info == nil {
		return nil, errors.New("the registry returned no details")
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
	key := i.cacheKey(registry, image)
	for _, s := range i.storage {
		if err := s.Store(ctx, key, *info); err != nil {
			log.DefaultLogger.Warnw("error while saving image details in the cache", "registry", key.Registry, "image", key.Image, "error", err)
		}
	}
}

func (i *inspector) ResolveName(registry, image string) string {
	if ExtractRegistry(image) != "" {
		return image
	}
	if registry == "" {
		registry = i.defaultRegistry
	}
	if registry == "" {
		return image
	}
	return fmt.Sprintf("%s/%s", registry, image)
}

func (i *inspector) Inspect(ctx context.Context, registry, image string, pullPolicy corev1.PullPolicy, pullSecretNames []string) (*Info, error) {
	// Load from cache
	if pullPolicy != corev1.PullAlways {
		value := i.get(ctx, registry, image)
		if value != nil {
			return value, nil
		}
	}

	// Fetch the data of the resolved name. The pod pulls that name, and the cache stores the data
	// under it, so an image without a registry must not be read from the Docker Hub instead.
	resolvedName := i.ResolveName(registry, image)
	value, err := i.fetch(ctx, "", resolvedName, pullSecretNames)
	if err != nil {
		return nil, errors.Wrapf(err, "inspecting the image %q", resolvedName)
	}

	// Save asynchronously
	go i.save(context.Background(), registry, image, value)

	return value, nil
}
