# Integrating with Slack

In order to receive notifications in slack about the status of the testing process, Testkube provides integration with slack, for this to work bellow are the steps to configure it.

## Install Testkube bot to your Slack Workspace

Teskube bot
<a href="https://slack.com/oauth/v2/authorize?client_id=1943550956369.3416932538629&scope=chat:write,chat:write.public,groups:read,channels:read&user_scope="><img alt="Add Testkube bot to your Slack workspace" height="40" width="139" src="https://platform.slack-edge.com/img/add_to_slack.png" srcSet="https://platform.slack-edge.com/img/add_to_slack.png 1x, https://platform.slack-edge.com/img/add_to_slack@2x.png 2x" /></a>

![img.gif](img/add-testkube-bot-to-workspace.gif)

## Add Testkube bot to a channel

![img.gif](img/add-testkube-bot-to-channel.gif)
## Configure Testkube to use the slack bot token

Populate slackToken in the helm values file, then install testkube using helm install, see [Manual Testkube Helm Charts Installation](installing.md) for more information.
