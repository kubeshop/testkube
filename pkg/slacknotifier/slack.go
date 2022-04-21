package slacknotifier

import (
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

type client struct {
	SlackClient *slack.Client
	ChannelId   string
}

var c *client

func Init() {

	if id, ok := os.LookupEnv("SLACK_CHANNEL_ID"); ok {
		c = &client{ChannelId: id}
		if token, ok := os.LookupEnv("SLACK_TOKEN"); ok {
			c.SlackClient = slack.New(token, slack.OptionDebug(true))
		}
	}
}

func SendMessage(message string) {
	if c == nil {
		Init()
	}
	if c != nil {
		_, _, err := c.SlackClient.PostMessage(c.ChannelId, slack.MsgOptionText(message, false))
		if err != nil {
			fmt.Printf("Error: %s", err)
		}
	}
}
