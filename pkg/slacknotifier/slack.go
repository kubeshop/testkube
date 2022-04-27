package slacknotifier

import (
	"os"

	"github.com/slack-go/slack"
)

type client struct {
	SlackClient *slack.Client
	ChannelId   string
}

var c *client

func init() {
	if id, ok := os.LookupEnv("SLACK_CHANNEL_ID"); ok {
		c = &client{ChannelId: id}
		if token, ok := os.LookupEnv("SLACK_TOKEN"); ok {
			c.SlackClient = slack.New(token, slack.OptionDebug(true))
		}
	}
}

func SendMessage(message string) error {
	if c != nil && c.SlackClient != nil {
		_, _, err := c.SlackClient.PostMessage(c.ChannelId, slack.MsgOptionText(message, false))
		if err != nil {
			return err
		}
	}
	return nil
}
