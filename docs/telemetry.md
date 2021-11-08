# Telemetry

To improve the end-user experience TestKube collects anonymous data about usage and sends it to us.

Telemetry collects and scrambles information about the host when the API server is bootstrapped for the first time. 

The collected data looks like this.
```
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