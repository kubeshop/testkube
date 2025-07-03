package imageinspector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
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
	authConfigs, err := ParseSecretData(pullSecrets, registry)
	if err != nil {
		return nil, err
	}

	// Select the auth
	cranePlatformOption := crane.WithPlatform(&v1.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH})
	craneOptions := []crane.Option{crane.WithContext(ctx)}
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
func ParseSecretData(imageSecrets []corev1.Secret, registry string) ([]authn.AuthConfig, error) {
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
		}
	}

	// If registry is an AWS ECR private registry, fetch the auth token, e.g. <ID>.dkr.ecr.<REGION>.amazonaws.com
	if strings.HasSuffix(registry, "amazonaws.com") && strings.Contains(registry, ".ecr.") {
		// Generate token for AWS ECR
		token, err := getAWSAuthToken()
		if err != nil {
			// If we fail to get the token, print error message but continue
			fmt.Printf("Failed to get AWS ECR auth token: %v", err)
		} else {
			// Append the AWS ECR auth token to the results
			// AWS ECR uses "AWS" as the username and the token as the password
			fmt.Printf("Using AWS ECR auth token for registry %s\n", registry)
			fmt.Printf("Using AWS ECR auth token: %s\n", token)
			results = append(results, authn.AuthConfig{
				Username: token[:strings.Index(token, ":")], // Extract username from token
				Password: token[strings.Index(token, ":")+1:], // Extract password from token
			})
		}
	}

	return results, nil
}

func getAWSAuthToken() (string, error) {
	// Load the AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	// Create an ECR client
	ecrClient := ecr.NewFromConfig(cfg)
	// Get the authorization token from ECR
	input := &ecr.GetAuthorizationTokenInput{}
	result, err := ecrClient.GetAuthorizationToken(context.TODO(), input)
    if err != nil {
		return "", fmt.Errorf("failed to get ECR authorization token: %w", err)
	}
	// Check if we have authorization data
	if len(result.AuthorizationData) > 0 {
		// Decode the authorization token
    	authData := result.AuthorizationData[0]
    	decodedToken, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
    	if err != nil {
			return "", fmt.Errorf("failed to decode ECR authorization token: %w", err)
		}
		// The decoded token is in the format "username:password", we return it as a string
		return string(decodedToken), nil
	} else {
		return "", fmt.Errorf("no authorization data found in ECR response")
	}
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
