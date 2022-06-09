# Telemetry

To improve the end-user experience, Testkube collects anonymous telemetry data about usage.

Participation in this program is optional. You may [opt-out](#how-to-opt-out) if you'd prefer not to share any information.

The data collected is always anonymous, not traceable to the source, and only used in aggregate form. 

Telemetry collects and scrambles information about the host when the API server is bootstrapped for the first time. 

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

## **What We Collect**

Analytics tracked include:
- The unique id generated from the MAC address.
- The testkube version.
- The command being executed.

## **How to Opt Out?**

To *opt out* of the Testkube telemetry data gathering:
```
kubectl testkube disable telemetry
```

To *opt in*:
```
kubectl testkube enable telemetry
```

To check the current *status*:
``` 
kubectl testkube status
```
