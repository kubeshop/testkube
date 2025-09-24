package imageinspector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	ecr "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/utils"
)

type craneFetcher struct {
}

func NewCraneFetcher() InfoFetcher {
	return &craneFetcher{}
}

func (c *craneFetcher) Fetch(ctx context.Context, registry, image string, pullSecrets []corev1.Secret) (*Info, error) {
	// If registry is not provided, extract it from the image name
	if registry == "" {
		if registry = ExtractRegistry(image); registry == "" {
			registry = utils.DefaultDockerRegistry
		}
	}

	// If registry is provided via config and the image does not start with the registry, prepend it
	if registry != "" && registry != utils.DefaultDockerRegistry && !strings.HasPrefix(image, registry+"/") {
		image = registry + "/" + image
	}

	// Support pull secrets
	authConfigs, err := ParseSecretData(pullSecrets, registry, image)
	if err != nil {
		return nil, err
	}

	amazonKeychain := authn.NewKeychainFromHelper(ecr.NewECRHelper())
	azureKeychain := authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper())
	keychain := authn.NewMultiKeychain(
		authn.DefaultKeychain,
		google.Keychain,
		github.Keychain,
		amazonKeychain,
		azureKeychain,
	)

	// Select the auth
	cranePlatformOption := crane.WithPlatform(&v1.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH})
	craneOptions := []crane.Option{crane.WithContext(ctx), crane.WithAuthFromKeychain(keychain)}
	if len(authConfigs) > 0 {
		craneOptions = append(craneOptions, crane.WithAuth(authn.FromConfig(authConfigs[0])))
	}

	// Fetch the image configuration
	fetchedAt := time.Now()
	serializedImageConfig, err := crane.Config(image, append(craneOptions, cranePlatformOption)...)

	// Retry again without specifying platform
	if err != nil && (strings.Contains(err.Error(), "no child") || strings.Contains(err.Error(), "not known")) {
		serializedImageConfig, err = crane.Config(image, craneOptions...)
	}

	if err != nil {
		return nil, err
	}
	var imageConfig DockerImage
	if err = json.Unmarshal(serializedImageConfig, &imageConfig); err != nil {
		return nil, err
	}

	// Build the required image information
	user, group := determineUserGroupPair(imageConfig.Config.User)
	result := &Info{
		FetchedAt:  fetchedAt,
		Entrypoint: imageConfig.Config.Entrypoint,
		Cmd:        imageConfig.Config.Cmd,
		WorkingDir: imageConfig.Config.WorkingDir,
		User:       user,
		Group:      group,
	}

	// Try to detect optional shell information
	for i := len(imageConfig.History); i > 0; i-- {
		command := imageConfig.History[i-1].CreatedBy
		re, err := regexp.Compile(`/bin/([a-z]*)sh`)
		if err != nil {
			return nil, err
		}

		result.Shell = re.FindString(command)
		if result.Shell != "" {
			break
		}
	}

	return result, nil
}

// DockerImage contains definition of docker image
type DockerImage struct {
	Config struct {
		User       string   `json:"User"`
		Entrypoint []string `json:"Entrypoint"`
		Cmd        []string `json:"Cmd"`
		WorkingDir string   `json:"WorkingDir"`
	} `json:"config"`
	History []struct {
		Created   time.Time `json:"created"`
		CreatedBy string    `json:"created_by"`
	} `json:"history"`
}

// ExtractRegistry takes a container image string and returns the registry part.
func ExtractRegistry(image string) string {
	parts := strings.Split(image, "/")
	// If the image is just a name, return the default registry.
	if len(parts) == 1 {
		return ""
	}
	// If the first part contains '.' or ':', it's likely a registry.
	if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
		return parts[0]
	}
	return ""
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

// DockerAuths contains an embedded DockerAuthConfigs
type DockerAuths struct {
	Auths map[string]authn.AuthConfig `json:"auths"`
}

// ParseSecretData parses secret data for docker auth config
func ParseSecretData(imageSecrets []corev1.Secret, registry, image string) ([]authn.AuthConfig, error) {
	var results []authn.AuthConfig
	for _, imageSecret := range imageSecrets {
		auths := DockerAuths{}
		if jsonData, ok := imageSecret.Data[".dockerconfigjson"]; ok {
			if err := json.Unmarshal(jsonData, &auths); err != nil {
				return nil, err
			}
		} else if configData, ok := imageSecret.Data[".dockercfg"]; ok {
			if err := json.Unmarshal(configData, &auths.Auths); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("imagePullSecret %s contains neither .dockercfg nor .dockerconfigjson", imageSecret.Name)
		}

		// Determine if there is a secret for the specified registry
		if creds, ok := auths.Auths[registry]; ok {
			username, password, err := extractRegistryCredentials(creds)
			if err != nil {
				return nil, err
			}

			results = append(results, authn.AuthConfig{Username: username, Password: password})
		} else {
			var slice []struct {
				Path  string
				Creds authn.AuthConfig
			}

			for path, creds := range auths.Auths {
				slice = append(slice, struct {
					Path  string
					Creds authn.AuthConfig
				}{
					Path:  path,
					Creds: creds,
				})
			}

			sort.Slice(slice, func(i, j int) bool {
				return slice[i].Path > slice[j].Path
			})

			for _, item := range slice {
				if strings.HasPrefix(image, item.Path) {
					username, password, err := extractRegistryCredentials(item.Creds)
					if err != nil {
						return nil, err
					}

					results = append(results, authn.AuthConfig{Username: username, Password: password})
					break
				}
			}
		}
	}

	return results, nil
}

func extractRegistryCredentials(creds authn.AuthConfig) (username, password string, err error) {
	if creds.Auth == "" {
		return creds.Username, creds.Password, nil
	}

	decoder := base64.StdEncoding
	if !strings.HasSuffix(strings.TrimSpace(creds.Auth), "=") {
		// Modify the decoder to be raw if no padding is present
		decoder = decoder.WithPadding(base64.NoPadding)
	}

	base64Decoded, err := decoder.DecodeString(creds.Auth)
	if err != nil {
		return "", "", err
	}

	splitted := strings.SplitN(string(base64Decoded), ":", 2)
	if len(splitted) != 2 {
		return creds.Username, creds.Password, nil
	}

	return splitted[0], splitted[1], nil
}
