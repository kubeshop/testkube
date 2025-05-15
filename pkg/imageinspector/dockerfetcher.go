package imageinspector

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/client"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/utils"
)

type dockerFetcher struct {
	client *client.Client
	crane  InfoFetcher
}

func NewDockerFetcher(dockerClient *client.Client) InfoFetcher {
	return &dockerFetcher{
		client: dockerClient,
		crane:  NewCraneFetcher(),
	}
}

func (c *dockerFetcher) Fetch(ctx context.Context, registry, image string, pullSecrets []corev1.Secret) (*Info, error) {
	// If registry is not provided, extract it from the image name
	if registry == "" {
		registry = extractRegistry(image)
	}

	// If registry is provided via config and the image does not start with the registry, prepend it
	if registry != "" && registry != utils.DefaultDockerRegistry && !strings.HasPrefix(image, registry+"/") {
		image = registry + "/" + image
	}

	// Support pull secrets
	// TODO: Use that auth?
	//authConfigs, err := ParseSecretData(pullSecrets, registry)
	//if err != nil {
	//	return nil, err
	//}

	// Fetch the image configuration
	fetchedAt := time.Now()
	inspectConfig, _, err := c.client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		// Fallback to Crane when not found
		return c.crane.Fetch(ctx, registry, image, pullSecrets)
	}

	// Build the required image information
	user, group := determineUserGroupPair(inspectConfig.Config.User)
	result := &Info{
		FetchedAt:  fetchedAt,
		Entrypoint: inspectConfig.Config.Entrypoint,
		Cmd:        inspectConfig.Config.Cmd,
		WorkingDir: inspectConfig.Config.WorkingDir,
		User:       user,
		Group:      group,
	}

	// TODO: detect optional shell information
	//for i := len(inspectConfig.History); i > 0; i-- {
	//	command := inspectConfig.History[i-1].CreatedBy
	//	re, err := regexp.Compile(`/bin/([a-z]*)sh`)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	result.Shell = re.FindString(command)
	//	if result.Shell != "" {
	//		break
	//	}
	//}

	return result, nil
}
