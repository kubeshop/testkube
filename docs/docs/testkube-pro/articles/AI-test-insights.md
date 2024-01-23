# AI Test Insights

export const ProBadge = () => {
  return (
    <span>
      <p class="pro-badge">PRO FEATURE</p>
    </span>
  );
}

<ProBadge />

:::note
The AI Insights feature on Testkube utilizes artificial intelligence to help you debug your failed tests faster. It collects relevant bits of the failed logs and sends them to OpenAI which processes them and gives an assessment on why the test failed.
:::

## Example of Creating a cURL Test

Login to your Testkube pro account and create a test. The test in this example will send an HTTP GET request to an endpoint and validate that the response - an IP address - is received.

Provide the following details: 
Name: `curl-url-test`
Type: `curl/test`
Source: `String`

```json
{
    "command": [
      "curl",
      "http://ip.jsontest.com/",
      "-H",
      "'Accept: application/json'"
    ],
    "expected_status": "200",
    "expected_body": "{\"ip\": \"120.88.40.210\"}"
  }
```

![Create a Test](../../img/create-a-test.png)

## Execute and Validate Tests

Click on `Run Now` to execute the test. After the test has finished executing, you can click on it to view the results. In this case, the test has failed. Let's analyze the logs to understand why the test has failed.

![Log Output](../../img/log-output.png)

It shows that the IP address we are looking for in the request is different; hence, the test has failed. Let's see what the AI Analysis feature has to say on this.

## Using AI Analysis

Navigate to the AI Analysis Tab. Testkube will automatically collect the relevant details from the log and analyze them.

![AI Analysis Results](../../img/AI-analysis-results.png)

As per the AI Analysis, the assessment is “The test execution is failing because the expected result does not match the actual result. The expected result was not received from the API”. This means that the response that we received is different from what is expected, which is spot on. 

AI Analysis also provides you with a list of suggestions like checking the URL, headers, and internet connection, and validating the response. 

:::note
AI Analysis is an experimental feature. The results obtained may be incorrect or misleading and we’re actively working on improving its accuracy. Users are cautioned to refrain from relying upon these results for critical evaluations and should approach them with caution.
:::

Let's update the expected IP address in the test and execute it again.

```json
{
    "command": [
      "curl",
      "http://ip.jsontest.com/",
      "-H",
      "'Accept: application/json'"
    ],
    "expected_status": "200",
    "expected_body": "{\"ip\": \"120.88.40.232\"}"
  }
```
![Passed Test](../../img/passed-test.png)

Now if you execute the test again, it passes. Note that the AI Analysis tab is not present this time. This is because AI Analysis is best suited to analyze failed tests and not otherwise.

 Watch our YouTube hands-on video at [Get AI Insights for Your Tests in Kubernetes](https://www.youtube.com/watch?v=29zVIzMBaow).

This was a simple demo to show you how to use Testkube’s AI Analysis feature to analyze logs and fix failing tests quickly. You can create complex tests to test your applications and infrastructure. 

If you have feedback or concerns using the AI analysis feature, do share them on our [Slack Channel](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email) for faster resolution.




