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
	errWrongRepositoryType = errors.New("type: wrong repository type, only 'git' supported")
	// errAuthFailed is auth failed error
	errAuthFailed = errors.New("username or token: authentication failed")
	// errWrongAuthType is wrong auth type error
	errWrongAuthType = errors.New("auth type: wrong auth type, only 'basic' or 'header' supported")
)

func (s TestkubeAPI) ValidateRepositoryHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.Repository
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if request.Type_ != "git" {
			return s.Error(c, http.StatusBadRequest, errWrongRepositoryType)
		}

		switch testkube.GitAuthType(request.AuthType) {
		case testkube.GitAuthTypeBasic, testkube.GitAuthTypeHeader, testkube.GitAuthTypeEmpty:
		default:
			return s.Error(c, http.StatusBadRequest, errWrongAuthType)
		}

		if request.Username == "" && request.Token == "" {
			var items = []struct {
				secretRef *testkube.SecretRef
				field     *string
			}{
				{
					request.UsernameSecret,
					&request.Username,
				},
				{
					request.TokenSecret,
					&request.Token,
				},
			}

			for _, item := range items {
				if item.secretRef != nil {
					secretClient, err := secret.NewClient(item.secretRef.Namespace)
					if err != nil {
						return s.Error(c, http.StatusBadGateway, err)
					}

					data, err := secretClient.Get(item.secretRef.Name)
					if err != nil {
						return s.Error(c, http.StatusBadGateway, err)
					}

					if value, ok := data[item.secretRef.Key]; !ok {
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("username secret or token secret: missed key %s in secret %s/%s",
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
				return s.Error(c, http.StatusBadGateway, err)
			}
			defer os.RemoveAll(dir) // clean up

			fetcher := content.NewFetcher(dir)
			if _, err = fetcher.FetchGit(&request); err != nil {
				message := strings.ToLower(err.Error())
				switch {
				case strings.Contains(message, "remote: not found"):
					err = fmt.Errorf("uri: repository not found %s", request.Uri)

				case strings.Contains(message, "could not find remote branch"):
					err = fmt.Errorf("branch: branch not found %s", request.Branch)

				case strings.Contains(message, "did not match any file") ||
					strings.Contains(message, "couldn't find remote"):
					err = fmt.Errorf("commit: commit not found %s", request.Commit)

				case strings.Contains(message, "authentication failed") ||
					strings.Contains(message, "could not read username"):
					err = errAuthFailed
				}

				return s.Error(c, http.StatusBadGateway, err)
			}
		}

		return c.SendStatus(http.StatusNoContent)
	}
}
