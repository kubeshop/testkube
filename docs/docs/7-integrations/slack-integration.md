---
sidebar_position: 2
---
# Integrating with Slack

In order to receive notifications in Slack about the status of the testing process, Testkube provides integration with Slack. Below are the steps to configure it.

## **Install Testkube bot to Your Slack Workspace**

Testkube bot
<a href="https://slack.com/oauth/v2/authorize?client_id=1943550956369.3416932538629&scope=chat:write,chat:write.public,groups:read,channels:read&user_scope="><img alt="Add Testkube bot to your Slack workspace" height="40" width="139" src="https://platform.slack-edge.com/img/add_to_slack.png" srcSet="https://platform.slack-edge.com/img/add_to_slack.png 1x, https://platform.slack-edge.com/img/add_to_slack@2x.png 2x" /></a>

![img.gif](../img/add-testkube-bot-to-workspace.gif)

## **Configure Testkube to Use the Slack bot Token**

Populate slackToken in the Helm values file, then install Testkube using Helm install, see [Manual Testkube Helm Charts Installation](../1-installing.md) for more information.

## **Adjust Slack Config File**

By default the configuration [/charts/testkube-api/slack-config.json](https://github.com/kubeshop/helm-charts/blob/704c71fa3b8f0138f983ea9a2fa598ecbe3868ae/charts/testkube-api/slack-config.json) looks like below; it will send notification for all events and all test or test suite names with any labels.
If the channel is left empty, it will send to the first channel that Testkube bot is member of.

It is an array of config objects and can use any config combinations:
```json
[
    {
      "ChannelID": "",
      "selector": {},
      "testName": [],
      "testSuiteName": [],
      "events": [
        "start-test",
        "end-test-success",
        "end-test-failed",
        "end-test-aborted",
        "end-test-timeout",
        "start-testsuite",
        "end-testsuite-success",
        "end-testsuite-failed",
        "end-testsuite-aborted",
        "end-testsuite-timeout"
      ]
    }
  ]
```
e.g.

```json
[
    {
      "ChannelID": "C01234567",
      "selector": {"label1":"value1"},
      "testName": ["sanity", "testName2"],
      "testSuiteName": ["test-suite1", "test-suite2"],
      "events": [
        "end-test-failed",
        "end-test-timeout",
        "end-testsuite-failed",
        "end-testsuite-timeout"
      ]
    },
    {
      "ChannelID": "C07654342",
      "selector": {"label3":"value4"},
      "testName": ["integration-test1", "integration-test2"],
      "testSuiteName": ["integration-test-suite1", "integration-test-suite2"],
      "events": [
        "start-test",
        "end-test-success",
        "end-test-failed",
        "end-test-aborted",
        "end-test-timeout",
        "start-testsuite",
        "end-testsuite-success",
        "end-testsuite-failed",
        "end-testsuite-aborted",
        "end-testsuite-timeout"
      ]
    },
]
```

This will send notifications to the channel with the id `C01234567` for the test and test suites with labels `label1:value1`, tests "sanity", "testName2" and test suites "test-suite1", "test-suite2" on events "end-test-failed", "end-test-timeout", "end-testsuite-failed", "end-testsuite-timeout" and to the channel with the id `C07654342` for test with labels `label3:value4`, tests "integration-test1", "integration-test2" and test suites "integration-test-suite1", "integration-test-suite2" on all events.

## **Configure Message Template**

The default message is [/charts/testkube-api/slack-template.json](https://github.com/kubeshop/helm-charts/blob/311ff9f6fc38dfb5196b91a6f63ee7d3f59f7f4b/charts/testkube-api/slack-template.json) and is written using [Slack block kit builder](https://app.slack.com/block-kit-builder) and Golang templates, you can customize it depending on your needs. The following structure is referenced in the template where it is getting the data to show:

```go
type MessageArgs struct {
	ExecutionName string
	EventType     string
	Namespace     string
	Labels        string
	TestName      string
	TestType      string
	Status        string
	FailedSteps   int
	TotalSteps    int
	StartTime     string
	EndTime       string
	Duration      string
}
```

## **Add the Testkube Bot to a Channel**

If the goal is to receive all the notifications in one channel, add the Testkube bot to the channel and leave the `ChannelID` field empty in the `slack-config.json` file.
![img.gif](../img/add-testkube-bot-to-channel.gif)