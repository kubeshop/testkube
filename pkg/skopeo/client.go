package skopeo

import (
	"encoding/json"
	"math/rand"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/credentialprovider"
	"k8s.io/kubernetes/pkg/credentialprovider/secrets"

	"github.com/kubeshop/testkube/pkg/process"
)

// DockerImage contains definition of docker image
type DockerImage struct {
	Config struct {
		Entrypoint []string `json:"Entrypoint"`
		Cmd        []string `json:"Cmd"`
	} `json:"config"`
	History []struct {
		Created   time.Time `json:"created"`
		CreatedBy string    `json:"created_by"`
	} `json:"history"`
	Shell string `json:"-"`
}

type Inspector interface {
	Inspect(image string) (*DockerImage, error)
}

type client struct {
	keyring credentialprovider.DockerKeyring
}

func NewClient() *client {
	return &client{}
}

func NewClientFromSecret(imageSecrets []corev1.Secret) (*client, error) {
	keyring, err := secrets.MakeDockerKeyring(imageSecrets, &credentialprovider.FakeKeyring{})
	if err != nil {
		return nil, err
	}

	return &client{keyring: keyring}, nil
}

func (c *client) Inspect(image string) (*DockerImage, error) {
	args := []string{
		"--override-os",
		"linux",
		"inspect",
		"--config",
		"docker://" + image,
	}

	if c.keyring != nil {
		if authConfigs, ok := c.keyring.Lookup(image); ok && len(authConfigs) != 0 {
			rand.Seed(time.Now().UnixNano())
			i := 1 + rand.Intn(len(authConfigs))
			args = append(args, "--creds", authConfigs[i].Username+":"+authConfigs[i].Password)
		}
	}

	result, err := process.Execute("skopeo", args...)
	if err != nil {
		return nil, err
	}

	var dockerImage DockerImage
	if err = json.Unmarshal(result, &dockerImage); err != nil {
		return nil, err
	}

	var shell string
	for i := len(dockerImage.History); i > 0; i-- {
		command := dockerImage.History[i-1].CreatedBy
		re, err := regexp.Compile(`/bin/([a-z]*)sh`)
		if err != nil {
			return nil, err
		}

		shell = re.FindString(command)
		if shell != "" {
			break
		}
	}

	dockerImage.Shell = shell
	return &dockerImage, nil
}

var _ Inspector = (*client)(nil)
