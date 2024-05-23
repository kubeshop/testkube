# Integrating with Slack

In order to receive notifications in Slack about the status of the testing process, Testkube provides integration with Slack. Below are the configuration steps. Click [here](#video-tutorial) for a video tutorial.

## Step 1 - Install the Testkube bot to Your Slack Workspace

Testkube bot:
<a href="https://slack.com/oauth/v2/authorize?client_id=1943550956369.3416932538629&scope=chat:write,chat:write.public,groups:read,channels:read&user_scope="><img alt="Add Testkube bot to your Slack workspace" height="40" width="139" src="https://platform.slack-edge.com/img/add_to_slack.png" srcSet="https://platform.slack-edge.com/img/add_to_slack.png 1x, https://platform.slack-edge.com/img/add_to_slack@2x.png 2x" /></a>

Once you add Slack bot to your workspace, note the bot token, you will need it in the next step.

```
Authentification was succesfull!
Please use the following token in the Helm values for slackToken:
xoxb-123456789012-1234567890123-123456789012345678901234
```

## Step 2 - Configure Testkube to Use the Slack bot Token

Populate `slackToken` in the Helm values or apply it directly with:

```sh
helm upgrade \
  --install \
  --namespace testkube \
  --create-namespace \
  --set testkube-api.slackToken="YOUR_SLACK_TOKEN" \
  kubeshop/testkube
```

To see all the values that can be configured, check the Helm chart documentation [here](./helm-chart.md). 

## Step 3 - Add the Testkube bot to the Slack Channel

Add the Testkube bot to the Slack channel where you want to receive notifications. Do this by inviting the bot to a specific channel. By default, the Testkube bot will send to the first channel that the Testkube bot is member of. You can update the config in the next step.

![Add Testkube bot to Slack channel](../img/slack-running-example.png)

## Step 4 - (Optional) Adjust Slack Config File

By default, the configuration [/charts/testkube-api/slack-config.json](https://github.com/kubeshop/helm-charts/blob/704c71fa3b8f0138f983ea9a2fa598ecbe3868ae/charts/testkube-api/slack-config.json) looks like below; it will send notifications for all events and all test or test suite names with any labels.
If the channel is left empty, it will send to the first channel that the Testkube bot is member of.

It is an array of config objects and can use any config combinations:

```yaml title="config.yaml"
- ChannelID: C058AGQ25D2
  selector: {}
  testName: []
  testSuiteName: []
  testWorkflowName: []
  events:
    - start-test
    - end-test-success
    - end-test-failed
    - end-test-aborted
    - end-test-timeout
    - start-testsuite
    - end-testsuite-success
    - end-testsuite-failed
    - end-testsuite-aborted
    - end-testsuite-timeout
    - start-testworkflow
    - queue-testworkflow    
    - end-testworkflow-success
    - end-testworkflow-failed
    - end-testworkflow-aborted
```

To apply, pass the file to the Helm values:

```sh
helm upgrade \
  --install \
  --create-namespace \
  --namespace testkube \
  testkube \
  kubeshop/testkube \
  --set testkube-api.slackToken="$SLACK_BOT_TOKEN" \
  --set testkube-api.slackConfig="$(cat config.yaml)"
```

For example:

```yaml title="config.yaml"
- ChannelID: C01234567
  selector:
    label1: value1
  testName:
    - sanity
    - testName2
  testSuiteName:
    - test-suite1
    - test-suite2
  tesrWorkflowName:
    - test-workflow1
    - test-workflow2  
  events:
    - end-test-failed
    - end-test-timeout
    - end-testsuite-failed
    - end-testsuite-timeout
    - end-testworkflow-failed
- ChannelID: C07654342
  selector:
    label3: value4
  testName:
    - integration-test1
    - integration-test2
  testSuiteName:
    - integration-test-suite1
    - integration-test-suite2
   testWorkflowName:
    - integration-test-workflow1
    - integration-test-workflow2 
  events:
    - start-test
    - end-test-success
    - end-test-failed
    - end-test-aborted
    - end-test-timeout
    - start-testsuite
    - end-testsuite-success
    - end-testsuite-failed
    - end-testsuite-aborted
    - end-testsuite-timeout
    - start-testworkflow
    - queue-testworkflow    
    - end-testworkflow-success
    - end-testworkflow-failed
    - end-testworkflow-aborted
```

This will send notifications to the channel with the id `C01234567` for the test, test suites and test workflows with labels `label1:value1` as well as tests with the names "sanity" and "testName2", test suites with the names "test-suite1" and "test-suite2" and test workflows with the names "test-workflow1" and "test-workflow2", on events with the labels "end-test-failed", "end-test-timeout", "end-testsuite-failed", "end-testsuite-timeout" and "end-testworkflow-failed", and to the channel with the id `C07654342` for tests with labels `label3:value4`, tests with the names "integration-test1" and "integration-test2", test suites with the names "integration-test-suite1" and "integration-test-suite2" and test workflows with the names "integration-test-workflow1" and "integration-test-workflow2" on all events.

### Configure the Message Template

The default message is [/charts/testkube-api/slack-template.json](https://github.com/kubeshop/helm-charts/blob/311ff9f6fc38dfb5196b91a6f63ee7d3f59f7f4b/charts/testkube-api/slack-template.json) and is written using [Slack block kit builder](https://app.slack.com/block-kit-builder) and Golang templates. You can customize the message depending on your needs. The following structure is referenced in the template to show where the data to show is coming from:

```go
type MessageArgs struct {
	ExecutionID   string
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

## Video Tutorial

<iframe width="100%" height="315" src="https://www.youtube.com/embed/iaiiDilAyMY" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>
