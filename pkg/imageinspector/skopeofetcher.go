package imageinspector

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/skopeo"
)

type skopeoFetcher struct {
}

func NewSkopeoFetcher() InfoFetcher {
	return &skopeoFetcher{}
}

func (s *skopeoFetcher) Fetch(ctx context.Context, registry, image string, pullSecrets []corev1.Secret) (*Info, error) {
	client, err := skopeo.NewClientFromSecrets(pullSecrets, registry)
	if err != nil {
		return nil, err
	}
	info, err := client.Inspect(image) // TODO: Support passing context
	if err != nil {
		return nil, err
	}
	return &Info{
		FetchedAt:  time.Now(),
		Entrypoint: info.Config.Entrypoint,
		Cmd:        info.Config.Cmd,
		Shell:      info.Shell,
		WorkingDir: info.Config.WorkingDir,
	}, nil
}
