package slacknotifier

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

var (
	SlackBotClientID     = ""
	SlackBotClientSecret = ""
	oauthConfig          = &oauth2.Config{
		ClientID:     SlackBotClientID,
		ClientSecret: SlackBotClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://slack.com/oauth/v2/authorize",
			TokenURL: "https://slack.com/api/oauth.v2.access",
		},
		Scopes: []string{
			"chat:write",
			"chat:write.public"},
	}
)

// OauthHandler creates a handler for slack authentication
func OauthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		errStr := c.Params("error")
		if errStr != "" {
			c.Status(http.StatusUnauthorized)
			_, err := c.WriteString(errStr)
			return err
		}
		code := c.Params("code")
		if code == "" {
			c.Status(http.StatusBadRequest)
			_, err := c.WriteString("Code was not provided")
			return err
		}

		if _, err := oauthConfig.Exchange(ctx, code); err != nil {
			c.Status(http.StatusInternalServerError)
			_, err := c.WriteString("Unexpected error authorizing on slack")
			return err
		}
		_, err := c.WriteString("Authentification was succesfull!")
		return err
	}
}
