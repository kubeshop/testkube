package imageinspector

import (
	"context"
	"strconv"
	"strings"
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

func determineUserGroupPair(userGroupStr string) (int64, int64) {
	if userGroupStr == "" {
		userGroupStr = "0"
	}
	userStr, groupStr, _ := strings.Cut(userGroupStr, ":")
	if groupStr == "" {
		groupStr = userStr
	}
	user, _ := strconv.Atoi(userStr)
	group, _ := strconv.Atoi(groupStr)
	return int64(user), int64(group)
}
