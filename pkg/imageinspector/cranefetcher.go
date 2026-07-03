package imageinspector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	insecureRegistries map[string]struct{}
}

func NewCraneFetcher(insecureRegistries ...string) InfoFetcher {
	ir := make(map[string]struct{}, len(insecureRegistries))
	for _, r := range insecureRegistries {
		if r != "" {
			ir[r] = struct{}{}
		}
	}
	return &craneFetcher{insecureRegistries: ir}
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

	amazonKeychain := authn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard)))
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
	if _, ok := c.insecureRegistries[registry]; ok {
		craneOptions = append(craneOptions, crane.Insecure)
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

// stripURLScheme removes a leading "http://" or "https://" from a registry
// host or dockerconfigjson auth key, so entries using the traditional
// scheme-prefixed Docker credential-store format can still be matched
// against a bare registry hostname.
func stripURLScheme(s string) string {
	if rest, ok := strings.CutPrefix(s, "https://"); ok {
		return rest
	}
	if rest, ok := strings.CutPrefix(s, "http://"); ok {
		return rest
	}
	return s
}

// registryHost splits a scheme-stripped auth key into its host and reports
// whether the key refers to a registry as a whole rather than a path-scoped
// mirror entry. Keys with no path, or with only the legacy Docker
// credential-store API suffix (e.g. "index.docker.io/v1/"), are registry keys.
func registryHost(normalizedKey string) (host string, isRegistry bool) {
	host, rest, hasPath := strings.Cut(normalizedKey, "/")
	if !hasPath {
		return host, true
	}
	switch strings.Trim(rest, "/") {
	case "", "v1", "v2":
		return host, true
	}
	return host, false
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

		// Determine which credentials to use for the specified registry, in
		// order of decreasing specificity:
		//   1. an exact match on the registry key,
		//   2. the longest path-scoped key that prefixes the image (mirror auth),
		//   3. a scheme-insensitive match on the registry host, which also covers
		//      the traditional Docker credential-store format (e.g. the key
		//      "https://index.docker.io/v1/" for the "index.docker.io" registry).
		// Keys are visited in sorted order so selection is deterministic when
		// several keys would otherwise match equally.
		creds, ok := auths.Auths[registry]
		if !ok {
			keys := make([]string, 0, len(auths.Auths))
			for key := range auths.Auths {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			bestPathLen := -1
			var hostCreds authn.AuthConfig
			var hostFound bool
			for _, key := range keys {
				normalized := stripURLScheme(key)
				// Path-scoped (mirror) credential: the key prefixes the image path.
				if strings.Contains(normalized, "/") && strings.HasPrefix(image, normalized) {
					if len(normalized) > bestPathLen {
						bestPathLen = len(normalized)
						creds, ok = auths.Auths[key], true
					}
					continue
				}
				// Registry-level credential: the key's host equals the registry.
				if host, isRegistry := registryHost(normalized); isRegistry && host == registry && !hostFound {
					hostCreds, hostFound = auths.Auths[key], true
				}
			}
			// A path-scoped match is more specific, so only fall back to the
			// registry-level credential when no path-scoped key matched.
			if bestPathLen < 0 && hostFound {
				creds, ok = hostCreds, true
			}
		}
		if ok {
			username, password, err := extractRegistryCredentials(creds)
			if err != nil {
				return nil, err
			}

			results = append(results, authn.AuthConfig{Username: username, Password: password})
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
