# JMeter

This repository contains Dockerfiles for JMeter builds which are used by the Testkube JMeter Executor.

Currently supported builds:
<<<<<<< HEAD
* JMeter 5.6.1 with OpenJDK 11 built on RHEL UBI 8.8 (minimal)
=======
* JMeter 5.5 with OpenJDK 17 built on RHEL UBI 8.8 (minimal)
>>>>>>> aac7795c53bdcc1bf5840518a7d82413ccfd9508

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