![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to Testkube Gradle Executor

Testkube Gradle Executor is a test executor to run Gradle tasks with [Testkube](https://testkube.io).  

## Usage

You need to register and deploy the executor in your cluster.

```bash
kubectl apply -f examples/gradle-executor.yaml
```

Issue the following commands to create and start a Gradle test for a given Git repository:

```bash
kubectl testkube create test --git-uri https://github.com/lreimer/hands-on-testkube.git --git-branch main --type "gradle/test" --name gradle-test
kubectl testkube run test --watch gradle-test
```

# Issues and enchancements

Please follow the main [Testkube repository](https://github.com/kubeshop/testkube) for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

# Testkube

For more info go to [main testkube repo](https://github.com/kubeshop/testkube)

![Release](https://img.shields.io/github/v/release/kubeshop/testkube) [![Releases](https://img.shields.io/github/downloads/kubeshop/testkube/total.svg)](https://github.com/kubeshop/testkube/tags?label=Downloads) ![Go version](https://img.shields.io/github/go-mod/go-version/kubeshop/testkube)

![Docker builds](https://img.shields.io/docker/automated/kubeshop/testkube-api-server) ![Code build](https://img.shields.io/github/workflow/status/kubeshop/testkube/Code%20build%20and%20checks) ![Release date](https://img.shields.io/github/release-date/kubeshop/testkube)

![Twitter](https://img.shields.io/twitter/follow/thekubeshop?style=social) ![Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email)
 #### [Documentation](https://docs.testkube.io) | [Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email) 
