package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"

	thttp "github.com/kubeshop/testkube/pkg/http"
)

const slackAccessUrl = "https://slack.com/api/oauth.v2.access"

var (
	SlackBotClientID     = ""
	SlackBotClientSecret = ""
)

type oauthResponse struct {
	Ok         bool   `json:"ok"`
	AppID      string `json:"app_id"`
	AuthedUser struct {
		ID string `json:"id"`
	} `json:"authed_user"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	BotUserID   string `json:"bot_user_id"`
	Team        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	Enterprise          interface{} `json:"enterprise"`
	IsEnterpriseInstall bool        `json:"is_enterprise_install"`
}

// OauthHandler creates a handler for slack authentication
func (s TestkubeAPI) OauthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		errStr := c.Query("error", "")
		if errStr != "" {
			c.Status(http.StatusUnauthorized)
			_, err := c.WriteString(errStr)
			return err
		}
		code := c.Query("code", "")
		if code == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("Code was not provided"))
		}

		if SlackBotClientID == "" && SlackBotClientSecret == "" {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("\nSlack secrets are not set\n"))
		}

		var slackClient = thttp.NewClient()

		req, err := http.NewRequest(http.MethodGet, slackAccessUrl, nil)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("\nFailed to create request: %+v\n", err))
		}

		req.SetBasicAuth(SlackBotClientID, SlackBotClientSecret)
		q := req.URL.Query()
		q.Add("code", code)
		req.URL.RawQuery = q.Encode()
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, err := slackClient.Do(req)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("\nFailed to get access token: %+v\n", err))
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("\nInvalid format for access token: %+v", err))
		}

		oResp := oauthResponse{}
		err = json.Unmarshal(body, &oResp)

		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("\nUnable to unmarshal the response: %+v", err))
		}

		if len(oResp.AccessToken) == 0 {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("Unable to get the response from the slack oauth endpoint"))
		}

		_, err = c.WriteString(fmt.Sprintf("Authentification was succesfull!\nPlease use the following token in the helm values for slackToken : %s", oResp.AccessToken))
		return err
	}
}
