package v1

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/secret"
)

var (
	// errWrongRepositoryType is wrong repository type error
	errWrongRepositoryType = errors.New("type: Invalid repository type. Only 'git' supported")
	// errAuthFailed is auth failed error
	errAuthFailed = errors.New("username or token: Authentication failed. The provided credentials are not valid")
	// errWrongAuthType is wrong auth type error
	errWrongAuthType = errors.New("auth type: Invalid auth type. Only 'basic' or 'header' supported")
	// errRepositoryNotFound is repository not found error
	errRepositoryNotFound = errors.New("uri: The repository could not be found")
	// errBranchNotFound is branch not found error
	errBranchNotFound = errors.New("branch: The specified branch could not be found")
	// errCommitNotFound is commit not found error
	errCommitNotFound = errors.New("commit: The specified commit could not be found")
	// errPathNotFound is path not found error
	errPathNotFound = errors.New("path: The specified path could not be found")
)

func (s TestkubeAPI) ValidateRepositoryHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to validate repository"
		var request testkube.Repository
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: Unable to parse request: %w", errPrefix, err))
		}

		if request.Type_ != "git" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %w", errPrefix, errWrongRepositoryType))
		}

		switch testkube.GitAuthType(request.AuthType) {
		case testkube.GitAuthTypeBasic, testkube.GitAuthTypeHeader, testkube.GitAuthTypeEmpty:
		default:
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %w", errPrefix, errWrongAuthType))
		}

		if request.Username == "" && request.Token == "" {
			var items = []struct {
				secretRef *testkube.SecretRef
				field     *string
				name      string
			}{
				{
					request.UsernameSecret,
					&request.Username,
					"username secret",
				},
				{
					request.TokenSecret,
					&request.Token,
					"token secret",
				},
			}

			for _, item := range items {
				if item.secretRef != nil {
					secretClient, err := secret.NewClient(item.secretRef.Namespace)
					if err != nil {
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %s: Unable to get secret client: %w", errPrefix, item.name, err))
					}

					data, err := secretClient.Get(item.secretRef.Name)
					if err != nil {
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %s: Unable to get secret from secret client: %w", errPrefix, item.name, err))
					}

					if value, ok := data[item.secretRef.Key]; !ok {
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %s: Missed key %s in secret %s/%s", errPrefix, item.name,
							item.secretRef.Key, item.secretRef.Namespace, item.secretRef.Name))
					} else {
						*item.field = value
					}
				}
			}
		}

		if request.CertificateSecret == "" {
			dir, err := os.MkdirTemp("", "checkout")
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: Could not create folder for git: %w", errPrefix, err))
			}
			defer os.RemoveAll(dir) // clean up

			if request.Path != "" {
				request.WorkingDir = "." // skip partial checkout for path validation
			}

			fetcher := content.NewFetcher(dir)
			if _, err = fetcher.FetchGit(&request); err != nil {
				message := strings.ToLower(err.Error())
				switch {
				case strings.Contains(message, "remote: not found"):
					err = errRepositoryNotFound
				case strings.Contains(message, "could not find remote branch"):
					err = errBranchNotFound
				case strings.Contains(message, "did not match any file") ||
					strings.Contains(message, "couldn't find remote"):
					err = errCommitNotFound
				case strings.Contains(message, "authentication failed") ||
					strings.Contains(message, "could not read username"):
					err = errAuthFailed
				case strings.Contains(message, "no such file or directory"):
					err = errPathNotFound
				}

				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %w", errPrefix, err))
			}
		}

		return c.SendStatus(http.StatusNoContent)
	}
}
