# Telemetry

To improve the end-user experience, Testkube collects anonymous telemetry data about usage.

Participation in this program is optional. You may [opt-out](#how-to-opt-out) if you'd prefer not to share any information.

The data collected is always anonymous, not traceable to the source, and only used in aggregate form. 

## Dashboard telemetry

The webui dashboard collects user consent and then sends telemetry to the following services:
* app.posthog.com
* api.segment.io
* google analytics

## API-server telemetry

API server Telemetry collects and scrambles information about the host when the API server is bootstrapped for the first time.

The collected data looks like this.

```json
{
  "anonymousId": "a4652358effb311a074bf84d2aed5a7d270dee858bff10e847df2a9ea132bb38",
  "context": {
    "library": {
      "name": "analytics-go",
      "version": "3.0.0"
    }
  },
  "event": "testkube-heartbeat",
  "integrations": {},
  "messageId": "2021-11-04 19:54:40.029549 +0100 CET m=+0.148209228",
  "originalTimestamp": "2021-11-04T19:54:40.029571+01:00",
  "receivedAt": "2021-11-04T18:54:41.004Z",
  "sentAt": "2021-11-04T18:54:40.029Z",
  "timestamp": "2021-11-04T18:54:41.004Z",
  "type": "track"
}
```

### What We Collect

The telemetry data we use in our metrics is limited to:

 - The number of CLI installations.
 - The number of unique CLI usages in a day.
 - The number of installations to a cluster.
 - The number of unique active cluster installations.
 - The number of people who disable telemetry.
 - The number of created test suites and the number of steps.
 - The number of tests, their types and their sources.
 - The number of test suite runs and their durations.
 - The number of workflows created and run.
 - The number of workflow templates created and used.
 - Error details resulting from CLI commands.

It is sent to https://analytics.testkube.io and https://www.google-analytics.com

## How to Opt Out?

To opt out of the Testkube telemetry collection:

```sh
testkube disable telemetry
```

To *opt in*:

```sh
testkube enable telemetry
```

To check the current *status*:

```sh
testkube status telemetry
```
