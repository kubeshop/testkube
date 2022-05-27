package slacknotifier

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
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
func OauthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		errStr := c.Query("error", "")
		if errStr != "" {
			c.Status(http.StatusUnauthorized)
			_, err := c.WriteString(errStr)
			return err
		}
		code := c.Query("code", "")
		if code == "" {
			c.Status(http.StatusBadRequest)
			_, err := c.WriteString("Code was not provided")
			return err
		}
		//fmt.Printf("\n---->code is |%s|\n", code)

		var slackClient = &http.Client{Timeout: 10 * time.Second}

		req, err := http.NewRequest(http.MethodGet, slackAccessUrl, nil)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			_, err = c.WriteString(fmt.Sprintf("\nFailed to create request: %+v\n", err))
			return err
		}

		req.SetBasicAuth(SlackBotClientID, SlackBotClientSecret)
		q := req.URL.Query()
		q.Add("code", code)
		req.URL.RawQuery = q.Encode()
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, err := slackClient.Do(req)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			_, err = c.WriteString(fmt.Sprintf("\nFailed to get access token: %+v\n", err))
			return err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			c.Status(http.StatusInternalServerError)
			_, err = c.WriteString(fmt.Sprintf("\nInvalid format for access token: %+v", err))
			return err
		}
		oResp := oauthResponse{}
		err = json.Unmarshal(body, &oResp)

		if err != nil {
			c.Status(http.StatusInternalServerError)
			_, err = c.WriteString(fmt.Sprintf("\nUnable to unmarshal the response: %+v", err))
			return err
		}

		_, err = c.WriteString(fmt.Sprintf("Authentification was succesfull!\nPlease use the following token to configure testkube-bot: %s", oResp.AccessToken))
		return err
	}
}
