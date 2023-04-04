package v1

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/secret"
)

var (
	// errWrongRepositoryType is wrong repository type error
	errWrongRepositoryType = errors.New("wrong repository type, only 'git' supported")
)

func (s TestkubeAPI) ValidateRepositoryHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to validate repository"
		var request testkube.Repository
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: unable to parse request: %w", errPrefix, err))
		}

		if request.Type_ != "git" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %w", errPrefix, errWrongRepositoryType))
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
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: unable to get secret client: %w", errPrefix, err))
					}

					data, err := secretClient.Get(item.secretRef.Name)
					if err != nil {
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: unable to get secret from secret client: %w", errPrefix, err))
					}

					if value, ok := data[item.secretRef.Key]; !ok {
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: missed key %s in secret %s/%s", errPrefix,
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
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not create folder for git: %w", errPrefix, err))
			}
			defer os.RemoveAll(dir) // clean up

			fetcher := content.NewFetcher(dir)
			if _, err = fetcher.FetchGit(&request); err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: could not fetch git directory: %w", errPrefix, err))
			}
		}

		return c.SendStatus(http.StatusNoContent)
	}
}
