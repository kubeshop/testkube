import Tabs from "@theme/Tabs";
import TabItem from "@theme/TabItem";

# Test Workflows - Content

Often you need to provide some input data for your tests - whether it's Tests source code,
or some fixtures.

:::note

All the examples here that are using `content` are using it directly under the `spec`.
Please note, that you can also use it under a specific step - this way, such files won't be available in different steps.

:::

## Shared Directories

### /data

By default, the `/data` directory has an empty volume mounted that can be shared across the steps.

:::info

Please note, that this directory is shared across steps of a single execution. To share data across multiple executions, see [**Shared Directory Between Executions**](#shared-directory-between-executions).

:::

### Custom Volumes

You may create different directories that will share data across steps using Kubernetes. To do so, create and mount [**Empty Dir**](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir) volume.
This may be useful for sharing a dependencies across steps, for example.

```yaml
spec:
  pod:
    volumes:
    - name: golang-deps
      emptyDir: {}
  container:
    env:
    - name: GOCACHE
      value: /go-cache
    volumeMounts:
    - name: golang-deps
      mountPath: /go-cache
```

### Shared Directory Between Executions

:::info

We are planning to prepare a built-in caching mechanism that will allow you to share data more easily.

:::

If you want to cache some data between executions, i.e., to speed up execution, you may consider mounting [**Host's file system**](https://kubernetes.io/docs/concepts/storage/volumes/#hostpath) with `hostPath`.

```yaml
spec:
  pod:
    volumes:
    - name: golang-deps
      hostPath:
        path: '/tmp/go-cache-{{ workflow.name }}'
  container:
    env:
    - name: GOCACHE
      value: /go-cache
    volumeMounts:
    - name: golang-deps
      mountPath: /go-cache
```

This way, you will have a cache for each execution of the Test Workflow on the same node.

:::tip

If you want to cache data across multiple nodes, you may consider:
* using an Object Storage API like [**Minio**](https://min.io/), [**AWS S3**](https://aws.amazon.com/s3/) or [**GCP Cloud Storage**](https://cloud.google.com/storage),
  and save/read files using it, or
* creating a [**Persistent Volume**](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) and attaching it with `pod.volumes` and `container.volumeMounts`.

:::

## Static Files

To mount a static file, you can use `content.files`.

```yaml
spec:
  content:
    files:
    - path: /etc/nginx/nginx.conf
      content: |
        events {
          worker_connections 1024;
        }
        http {
          server {
            listen 8888;
            location / {
              root /www;
            }
          }
        }
    - path: /www/index.html
      content: "hello-there"
```

### Mounting From a Secret or ConfigMap

If you need to mount the secret as a file, you can use `contentFrom` clause in `content.files`. It has the same interface as `valueFrom` in [**native EnvVar**](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvarsource-v1-core).

```yaml
spec:
  content:
    files:
    - path: /some/path/secret.key
      contentFrom:
        secretKeyRef: # Mount from Secret
          name: secret-name
          key: key-name-in-secret
    - path: /some/path/nginx.conf
      contentFrom:
        configMapKeyRef: # Mount from ConfigMap
          name: secret-name
          key: key-name-in-secret
```

### Mounting Secret or ConfigMap as a Directory

As an alternative, you may use native Kubernetes volumes for mounting [**Secrets**](https://kubernetes.io/docs/concepts/storage/volumes/#secret) or [**ConfigMaps**](https://kubernetes.io/docs/concepts/storage/volumes/#configmap).

```yaml
spec:
  pod:
    volumes:
    # Create volume from the Secret
    - name: example-secret-volume
      secret:
        secretName: secret-name
    # Create volume from the ConfigMap
    - name: example-configmap-volume
      configMap:
        name: configmap-name
  container:
    volumeMounts:
    # Mount files from the Secret
    - name: example-secret-volume
      mountPath: /some/secret
    # Mount files from the ConfigMap
    - name: example-configmap-volume
      mountPath: /some/config
```

## Git Repository

Testkube allows you to easily fetch the Git repository using `content.git`.

### Custom Mount Path

By default, the Git repository contents are mounted at `/data/repo` directory.
If you want to mount it in a different directory, you can use `mountPath` property:

```yaml
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      mountPath: /custom/mount/path
  steps:
  - shell: tree /custom/mount/path
```

### Public Repositories

To use a public repository, simply pass the URL of the repository and it will be automatically mounted in `/data/repo`. 

```yaml
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      # You may also fetch different revision:
      # revision: main
      # revision: v1.7.30
      # revision: 2457ba80c1e0ade682b202fbec7062d82107e12f
  steps:
  - shell: tree /data/repo
```

### Using Username and Password/Token

Private repositories may need authentication with username and password/token.
To authenticate that way, you can use `username`/`usernameFrom` and `token`/`tokenFrom` properties.

The `username` and `token` properties take plain-text arguments. If you want to take username and password/token from the Secret or ConfigMap,
you can use `usernameFrom` and/or `tokenFrom` properties. They have the same interface as `valueFrom` in [**native EnvVar**](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvarsource-v1-core).

```yaml
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      username: my-username
      # usernameFrom:
      #   secretKeyRef:
      #     name: git-credentials
      #     key: username
      token: my-token
      # tokenFrom:
      #   secretKeyRef:
      #     name: git-credentials
      #     key: token
```

### Using Authorization Header

The example above is using Basic Authorization. Some Git providers may require Bearer Authentication instead.
To provide the bearer token, you need to pass `authType: header` property.

```yaml
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      authType: header
      token: my-token
      # tokenFrom:
      #   secretKeyRef:
      #     name: git-credentials
      #     key: token
```

### Using SSH Key

You can use a private SSH key to access your repository, using either plain-text `sshKey` property,
or `sshKeyFrom` that has the same interface as `valueFrom` in [**native EnvVar**](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvarsource-v1-core).

```yaml
spec:
  content:
    git:
      uri: ssh://git@github.com/kubeshop/testkube.git
      sshKeyFrom:
        secretKeyRef:
          name: git-credentials
          key: ssh-key
      # sshKey: |
      #   -----BEGIN OPENSSH PRIVATE KEY-----
      #   b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
      #   XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
      #   XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
      #   XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
      #   Py55nRNObUBBaG/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX=
      #   -----END OPENSSH PRIVATE KEY-----
```

### Pulling Selected Files

You may pass file patterns ([**cone pattern set**](https://git-scm.com/docs/git-sparse-checkout#_internalscone_pattern_set)) in `content.git.paths` which should be fetched from the Git repository.
It will use [**sparse checkout**](https://git-scm.com/docs/git-sparse-checkout), so you will avoid transferring unnecessary files.

```yaml
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      paths:
      - test/cypress/executor-tests/cypress-12
```

### Customizable Git Branch

Often you may want to run the Test Workflow for different code revisions (i.e. running it on Pull Requests with [**GitHub Action**](https://github.com/marketplace/actions/testkube-action)).

You can use [**Test Workflow Expressions**](./test-workflows-expressions.md) there, so it may be customized with [**configuration variables**](./test-workflows-examples-configuration.md):

```yaml
spec:
  config:
    branch:
      type: string
      default: main
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      revision: '{{ config.branch }}'
```

## Fetching Tarballs

Testkube have a simple built-in way to fetch tarball archives for your tests. You may use `content.tarball` to specify the files to download and unpack:

```yaml
spec:
  content:
    tarball:
    - url: https://github.com/kubeshop/testkube/releases/download/v1.17.53/testkube_1.17.53_Linux_arm64.tar.gz
      path: /data/bin
  steps:
  - shell: tree /data/bin
```

## Fetching OCI Artifacts

Sometimes, especially in air-gapped environment, you may want to provide the test data via OCI Registries.

You may use [**ORAS**](https://oras.land/) to fetch artifacts from the OCI registry.
To do so, run a step with Docker image that has ORAS installed and pull the artifact into `/data` (or other known volume):

```yaml
spec:
  steps:
  - run:
      image: bitnami/oras:latest
      workingDir: /data
      shell: |
        oras pull 12345678901234.dkr.ecr.us-west-2.amazonaws.com/some/artifact:latest
  - shell: tree /data
```

### OCI Images

Sometimes the registry won't support the OCI Artifacts, and, instead, they are available as regular images.
To fetch the data from such an image, simply run a step using that image,
and copy all the expected contents into `/data` (or other known volume):

```yaml
spec:
  steps:
  - run:
      image: '12345678901234.dkr.ecr.us-west-2.amazonaws.com/some/image:latest'
      shell: 'cp -rf /some/test/content /data'
  - shell: tree /data
```

:::tip

Test Workflows automatically have access to some binaries to use (i.e. shell) even on distroless images.
Thanks to that, you can use general shell commands (i.e. `cp`, `tar` or `tree`) even on images where they are not available.

:::

## Object Storage

To download your test data from the Object Storage you may simply use a Docker image with CLI that allows that,
and fetch the data.

```yaml
spec:
  steps:
  - run:
      image: minio/mc:latest
      env:
      - name: STORAGE_URL
        value: https://s3.us-south.cloud-object-storage.appdomain.cloud
      - name: STORAGE_ACCESS_KEY
        value: AKIAIOSFODNN7EXAMPLE
      - name: STORAGE_SECRET_KEY
        value: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
      - name: STORAGE_BUCKET
        value: artifacts
      shell: |
        /usr/bin/mc config host add storage "$STORAGE_URL" "$STORAGE_ACCESS_KEY" "$STORAGE_SECRET_KEY"
        /usr/bin/mc cp "storage/$STORAGE_BUCKET/some-artifact.tar.gz" /data/some-artifact.tar.gz
        tar -zxf /data/some-artifact.tar.gz
        rm /data/some-artifact.tar.gz
  - shell: tree /data
```

## Custom Sources

Similarly to the Object Storage example above, you can perform any other action in first step,
copy the results to shared directory, and then perform any work on that.

## Reusing Content Between Multiple Test Workflows

You may have multiple Test Workflows that needs access to the same test content. It's a good practice to abstract that with [**Test Workflow Templates**](./test-workflows-examples-templates.md).

<Tabs>
<TabItem value="template" label="Template" default>

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflowTemplate
metadata:
  name: git-testkube
spec:
  config:
    revision:
      type: string
      default: main
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      revision: '{{ config.revision }}'
      sshKeyFrom:
        secretKeyRef:
          name: git-credentials
          key: ssh-key
```

</TabItem>
<TabItem value="workflow" label="Test Workflow">

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflowTemplate
metadata:
  name: example-workflow
spec:
  use:
  - name: git-testkube
    config:
      revision: develop
  steps:
  - shell: tree /data/repo
```

</TabItem>
</Tabs>
