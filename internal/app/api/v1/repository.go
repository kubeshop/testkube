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
		var request testkube.Repository
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if request.Type_ != "git" {
			return s.Error(c, http.StatusBadRequest, errWrongRepositoryType)
		}

		if request.CertificateSecret == "" && request.Username == "" && request.Token == "" {
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
						return s.Error(c, http.StatusBadGateway, fmt.Errorf("missed key %s in secret %s/%s",
							item.secretRef.Key, item.secretRef.Namespace, item.secretRef.Name))
					} else {
						*item.field = value
					}
				}
			}

			dir, err := os.MkdirTemp("", "checkout")
			if err != nil {
				return s.Error(c, http.StatusBadGateway, err)
			}
			defer os.RemoveAll(dir) // clean up

			fetcher := content.NewFetcher(dir)
			if _, err = fetcher.FetchGitDir(&request); err != nil {
				return s.Error(c, http.StatusBadGateway, err)
			}
		}

		return c.SendStatus(http.StatusNoContent)
	}
}
