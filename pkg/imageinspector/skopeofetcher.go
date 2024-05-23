package imageinspector

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/utils"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/skopeo"
)

type skopeoFetcher struct {
}

func NewSkopeoFetcher() InfoFetcher {
	return &skopeoFetcher{}
}

func (s *skopeoFetcher) Fetch(ctx context.Context, registry, image string, pullSecrets []corev1.Secret) (*Info, error) {
	// If registry is not provided, extract it from the image name
	if registry == "" {
		registry = extractRegistry(image)
	}
	client, err := skopeo.NewClientFromSecrets(pullSecrets, registry)
	if err != nil {
		return nil, err
	}
	info, err := client.Inspect(registry, image) // TODO: Support passing context
	if err != nil {
		return nil, err
	}
	user, group := determineUserGroupPair(info.Config.User)
	return &Info{
		FetchedAt:  time.Now(),
		Entrypoint: info.Config.Entrypoint,
		Cmd:        info.Config.Cmd,
		Shell:      info.Shell,
		WorkingDir: info.Config.WorkingDir,
		User:       user,
		Group:      group,
	}, nil
}

// extractRegistry takes a container image string and returns the registry part.
// It defaults to "docker.io" if no registry is specified.
func extractRegistry(image string) string {
	parts := strings.Split(image, "/")
	// If the image is just a name, return the default registry.
	if len(parts) == 1 {
		return utils.DefaultDockerRegistry
	}
	// If the first part contains '.' or ':', it's likely a registry.
	if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
		return parts[0]
	}
	return utils.DefaultDockerRegistry
}

func determineUserGroupPair(userGroupStr string) (int64, int64) {
	if userGroupStr == "" {
		userGroupStr = "0"
	}
	userStr, groupStr, _ := strings.Cut(userGroupStr, ":")
	if groupStr == "" {
		groupStr = "0"
	}
	user, _ := strconv.Atoi(userStr)
	group, _ := strconv.Atoi(groupStr)
	return int64(user), int64(group)
}
