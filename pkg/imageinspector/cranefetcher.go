package imageinspector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
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
	serializedImageConfig, err := fetchConfig(image, append(craneOptions, cranePlatformOption)...)

	// Retry again without specifying platform
	if err != nil && (strings.Contains(err.Error(), "no child") || strings.Contains(err.Error(), "not known")) {
		serializedImageConfig, err = fetchConfig(image, craneOptions...)
	}

	if err != nil {
		return nil, registryError(err, len(authConfigs) > 0)
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

// fetchConfig returns the raw configuration of the image. It does what crane.Config does, without
// the prefix that crane adds to an error, because the inspector already names the image.
func fetchConfig(image string, opts ...crane.Option) ([]byte, error) {
	o := crane.GetOptions(opts...)
	ref, err := name.ParseReference(image, o.Name...)
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, o.Remote...)
	if err != nil {
		return nil, err
	}
	return img.RawConfigFile()
}

// registryErr keeps the original error for errors.As and errors.Is, and gives the words for people.
type registryErr struct {
	message string
	err     error
}

func (e *registryErr) Error() string { return e.message }
func (e *registryErr) Unwrap() error { return e.err }

// registryError puts the answer of the registry into words. The error of the library repeats the
// image in the URL and prints the details of each answer as a Go map, so the message keeps the
// host, the code and the text of each answer. An error that did not come from the registry, for
// example a DNS or a TLS error, keeps its text.
func registryError(err error, hasPullSecret bool) error {
	var terr *transport.Error
	if !errors.As(err, &terr) {
		return err
	}
	answers := make([]string, 0, len(terr.Errors))
	hint := ""
	for _, d := range terr.Errors {
		answer := string(d.Code)
		if d.Message != "" {
			answer += ": " + d.Message
		}
		answers = append(answers, answer)
		if hint == "" {
			hint = registryHint(d.Code, hasPullSecret)
		}
	}
	if len(answers) == 0 {
		if terr.StatusCode == 0 {
			return err
		}
		answers = append(answers, fmt.Sprintf("%d %s", terr.StatusCode, http.StatusText(terr.StatusCode)))
		if terr.StatusCode == http.StatusUnauthorized || terr.StatusCode == http.StatusForbidden {
			hint = registryHint(transport.UnauthorizedErrorCode, hasPullSecret)
		}
	}
	registry := "the registry"
	if terr.Request != nil && terr.Request.URL != nil && terr.Request.URL.Host != "" {
		registry += " " + terr.Request.URL.Host
	}
	message := fmt.Sprintf("%s answered %s", registry, strings.Join(answers, "; "))
	if hint != "" {
		message += ". " + hint
	}
	return &registryErr{message: message, err: err}
}

// registryHint returns the next step for one answer of the registry. Docker Hub answers
// UNAUTHORIZED also for a repository that does not exist, so the hint names both causes.
func registryHint(code transport.ErrorCode, hasPullSecret bool) string {
	switch code {
	case transport.UnauthorizedErrorCode, transport.DeniedErrorCode:
		if hasPullSecret {
			return "The pull secret does not give access to this repository."
		}
		return "The repository does not exist, or it is private and the workflow has no pull secret for it."
	case transport.ManifestUnknownErrorCode:
		return "The tag does not exist."
	case transport.NameUnknownErrorCode:
		return "The repository does not exist."
	case transport.TooManyRequestsErrorCode:
		return "The registry limits the pulls. Use a pull secret or a mirror."
	}
	return ""
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
	// The legacy credential-store suffix always carries a trailing slash
	// ("index.docker.io/v1/"), so require it here; otherwise a repository
	// namespace literally named "v1" or "v2" (e.g. "myreg.io/v2") would be
	// misclassified as a whole-registry key and applied registry-wide.
	switch rest {
	case "", "v1/", "v2/":
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
		//   1. the longest path-scoped key that prefixes the image (mirror auth),
		//   2. an exact match on the registry key,
		//   3. a scheme-insensitive match on the registry host, which also covers
		//      the traditional Docker credential-store format (e.g. the key
		//      "https://index.docker.io/v1/" for the "index.docker.io" registry).
		// A path-scoped credential is more specific than a registry-level one, so
		// it takes precedence even when an exact registry key also exists. Keys are
		// visited in sorted order so selection is deterministic when several keys
		// would otherwise match equally.
		creds, ok := auths.Auths[registry]

		keys := make([]string, 0, len(auths.Auths))
		for key := range auths.Auths {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		bestPathLen := -1
		var pathCreds, hostCreds authn.AuthConfig
		var hostKey string
		var pathFound, hostFound bool
		for _, key := range keys {
			normalized := stripURLScheme(key)

			host, isRegistry := registryHost(normalized)
			if isRegistry {
				// Registry-level credential: the key's host equals the registry.
				// Prefer a secure key over an insecure "http://" one so credentials
				// meant for an https endpoint are not sent to an insecure one.
				if host == registry && (!hostFound || (strings.HasPrefix(hostKey, "http://") && !strings.HasPrefix(key, "http://"))) {
					hostCreds, hostKey, hostFound = auths.Auths[key], key, true
				}
				continue
			}

			// Path-scoped (mirror) credential: non-registry keys that prefix the image path.
			if normalized != "" && (image == normalized ||
				strings.HasPrefix(image, normalized+"/") ||
				strings.HasPrefix(image, normalized+":") ||
				strings.HasPrefix(image, normalized+"@")) {
				if len(normalized) > bestPathLen {
					bestPathLen = len(normalized)
					pathCreds, pathFound = auths.Auths[key], true
				}
			}
		}
		switch {
		case pathFound:
			// A path-scoped match overrides a registry-level credential.
			creds, ok = pathCreds, true
		case !ok && hostFound:
			// Fall back to a scheme-insensitive registry match only when no exact
			// registry key was present.
			creds, ok = hostCreds, true
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
