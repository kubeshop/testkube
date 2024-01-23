![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to Testkube kubepug Executor

Testkube kubepug Executor is a [kubepug](https://github.com/kubepug/kubepug) executor running directly in your Kubernetes cluster via [Testkube](https://testkube.io).
Its main function is to make sure your Kubernetes clusters are up to date, and using the latest APIs.

This Executor is automatically installed together with Testkube, there is no need to build it or register it to your local environment.

## Example usages

This Executor should be used to compare the [Kuberenetes API Swagger definitions](https://raw.githubusercontent.com/kubernetes/kubernetes/master/api/openapi-spec/swagger.json) with existing manifests.

### Test manifests against latest Kubernetes version

```yaml
apiVersion: v1
conditions:
- message: '{"health":"true"}'
  status: "True"
  type: Healthy
kind: ComponentStatus
metadata:
  creationTimestamp: null
  name: etcd-1
```

```bash
$ kubectl testkube create test --file file_name.yaml --type kubepug/yaml --name kubepug-example-test-1
Test created testkube / kubepug-example-test-1 🥇


$ kubectl testkube run test kubepug-example-test-1
Type          : kubepug/yaml
Name          : kubepug-example-test-1
Execution ID  : 62b59ae1657713ea1b003a25
Execution name: completely-helped-fowl



Test execution started

Watch test execution until complete:
$ kubectl testkube watch execution 62b59ae1657713ea1b003a25


Use following command to get test execution details:
$ kubectl testkube get execution 62b59ae1657713ea1b003a25


$ kubectl testkube get execution 62b59ae1657713ea1b003a25
ID:        62b59ae1657713ea1b003a25
Name:      completely-helped-fowl
Type:      kubepug/yaml
Duration:  00:00:05

Status test execution failed:

⨯
{"DeprecatedAPIs":[{"Description":"ComponentStatus (and ComponentStatusList) holds the cluster validation info. Deprecated: This API is deprecated in v1.19+","Group":"","Kind":"ComponentStatus","Version":"v1","Name":"","Deprecated":true,"Items":[{"Scope":"OBJECT","ObjectName":"etcd-1","Namespace":"","location":"/tmp/test-content4075001618"}]}],"DeletedAPIs":null}
```

### Test manifests against previous Kubernetes version

```yaml
apiVersion: v1
conditions:
- message: '{"health":"true"}'
  status: "True"
  type: Healthy
kind: ComponentStatus
metadata:
  creationTimestamp: null
  name: etcd-1
```

```bash
$ kubectl testkube run test kubepug-example-test-1 --args '--k8s-version=v1.18.0'
Type          : kubepug/yaml
Name          : kubepug-example-test-1
Execution ID  : 62b59d52657713ea1b003a2d
Execution name: notably-healthy-cricket



Test execution started

Watch test execution until complete:
$ kubectl testkube watch execution 62b59d52657713ea1b003a2d

Use following command to get test execution details:
$ kubectl testkube get execution 62b59d52657713ea1b003a2d


$ kubectl testkube get execution 62b59d52657713ea1b003a2d
ID:        62b59d52657713ea1b003a2d
Name:      sincerely-real-marten
Type:      kubepug/yaml
Duration:  00:00:05
Args:     --k8s-version=v1.18.0

{"DeprecatedAPIs":null,"DeletedAPIs":null}
Status Test execution completed with success 🥇
```

# Known limitations 

- large input files break it - at least when I tried to feed it the output of kubectl api-resources --verbs=list -o name | xargs -n 1 kubectl get --show-kind --ignore-not-found --all-namespaces -o json > all-namespaces-pug.json
- helm-charts with template variables don’t work. It seems to be a limitation from kubepug, but that’s just a hunch

# Issues and enchancements

Please follow the main [TestKube repository](https://github.com/kubeshop/testkube) for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

# Testkube

For more info go to [main Testkube repo](https://github.com/kubeshop/testkube)

![Release](https://img.shields.io/github/v/release/kubeshop/testkube) [![Releases](https://img.shields.io/github/downloads/kubeshop/testkube/total.svg)](https://github.com/kubeshop/testkube/tags?label=Downloads) ![Go version](https://img.shields.io/github/go-mod/go-version/kubeshop/testkube)

![Docker builds](https://img.shields.io/docker/automated/kubeshop/testkube-api-server) ![Code build](https://img.shields.io/github/workflow/status/kubeshop/testkube/Code%20build%20and%20checks) ![Release date](https://img.shields.io/github/release-date/kubeshop/testkube)

![Twitter](https://img.shields.io/twitter/follow/thekubeshop?style=social) ![Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email)

#### [Documentation](https://docs.testkube.io) | [Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email)
