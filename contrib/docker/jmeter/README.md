# JMeter

This repository contains Dockerfiles for JMeter builds which are used by the Testkube JMeter Executor.

Currently supported builds:
* JMeter 5.5 with OpenJDK 17 built on RHEL UBI 8.8 (minimal)

## Development

Use the following `make` targets to build and push the images:

To build the JMeter Docker image use:
```bash
make build
```

To do a quick test run of the JMeter Docker image use:
```bash
make test
```

To push the JMeter Docker image to the registry use:
```bash
make push
```