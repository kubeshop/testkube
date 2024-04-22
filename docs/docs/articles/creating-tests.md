# Creating Tests

Tests are single executor oriented objects. Tests can have different types, depending on which executors are installed in your cluster.

Testkube includes `postman/collection`, `cypress/project`, `curl/test`, `k6/script` and `soapui/xml` test types which are auto registered during the Testkube install by default.

As Testkube was designed with flexibility in mind, you can add your own executors to handle additional test types.

## Test Source

Tests can be currently created from multiple sources:

1. A simple `file` with the test content. For example, with Postman collections, we're exporting the collection as a JSON file. For cURL executors, we're passing a JSON file with the configured cURL command.
2. String - We can also define the content of the test as a string.
3. A Git directory - We can pass `repository`, `path` and `branch` where our tests are stored. This is used in the Cypress executor as Cypress tests are more like npm-based projects which can have a lot of files. We are handling sparse checkouts which are fast even in the case of huge mono-repos.
4. A Git file - Similarly to Git directories, we can use files located on Git by specifying `git-uri` and `branch`.

:::note
Not all executors support all input types. Please refer to the individual executors' documentation to see which options are available.
:::

## Create a Test

### Create Your First Test from a File (Postman Collection Test)

To create your first Postman collection in Testkube, export your collection into a file.

Right click on your collection name:

![create postman collection step 1](../img/test-create-1.png)

Click the **Export** button:

![create postman collection step 2](../img/test-create-2.png)

Save in a convenient location. In this example, we are using `~/Downloads/TODO.postman_collection.json` path.

![create postman collection step 3](../img/test-create-3.png)

Create a Testkube test using the exported JSON and give it a unique and fitting name. For simplicity's sake we used `test` in this example.

```sh
testkube create test --file ~/Downloads/TODO.postman_collection.json --name test
```

```sh title="Expected output:"
Detected test type postman/collection
Test created test 🥇
```

Test created! Now we have a reusable test.

### Updating Tests

If you need to update your test after change in Postman, re-export it to a file and run the update command:

```sh
testkube update test --file ~/Downloads/TODO.postman_collection.json --name test
```

To check if the test was created correctly, look at the `Test Custom Resource` in your Kubernetes cluster:

```sh title="Expected output:"
Detected test test type postman/collection
Test updated test 🥇
```

Testkube will override all test settings and content with the `update` method.

### Checking Test Content

Let's see what has been created:

```sh
kubectl get tests -n testkube
```

```sh title="Expected output:"
NAME   AGE
test   32s
```

Get the details of a test:

```sh
kubectl get tests -n testkube test-example -oyaml

name: test
type_: postman/collection
description: some test description
content: |-
    {
        "info": {
                "_postman_id": "b40de9fe-9201-4b03-8ca2-3064d9027dd6",
                "name": "TODO",
                "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
        },
        "item": [
                {
                        "name": "Create TODO",
                        "event": [
                                {
                                        "listen": "test",
                                        "script": {
                                                "exec": [
                                                        "pm.test(\"Status code is 201 CREATED\", function () {",
                                                        "    pm.response.to.have.status(201);",
                                                        "});",
                                                        "",
                                                        "",
                                                        "pm.test(\"Check if todo item craeted successfully\", function() {",
                                                        "    var json = pm.response.json();",
                                                        "    pm.environment.set(\"item\", json.url);",
                                                        "    pm.sendRequest(json.url, function (err, response) {",
                                                        "        var json = pm.response.json();",
                                                        "        pm.expect(json.title).to.eq(\"Create video for conference\");",
                                                        "",
                                                        "    });",
                                                        "    console.log(\"creating\", pm.environment.get(\"item\"))",
                                                        "})",
                                                        "",
                                                        ""
                                                ],
                                                "type": "text/javascript"
                                        }
                                },
                                {
                                        "listen": "prerequest",
                                        "script": {
                                                "exec": [
                                                        ""
                                                ],
                                                "type": "text/javascript"
                                        }
                                }
                        ],
                        "protocolProfileBehavior": {
                                "disabledSystemHeaders": {}
                        },
                        "request": {
                                "method": "POST",
                                "header": [
                                        {
                                                "key": "Content-Type",
                                                "value": "application/json",
                                                "type": "text"
                                        }
                                ],
                                "body": {
                                        "mode": "raw",
                                        "raw": "{\"title\":\"Create video for conference\",\"order\":1,\"completed\":false}"
                                },
                                "url": {
                                        "raw": "{{uri}}",
                                        "host": [
                                                "{{uri}}"
                                        ]
                                }
                        },
                        "response": []
                },
                {
                        "name": "Complete TODO item",
                        "event": [
                                {
                                        "listen": "prerequest",
                                        "script": {
                                                "exec": [
                                                        "console.log(\"completing\", pm.environment.get(\"item\"))"
                                                ],
                                                "type": "text/javascript"
                                        }
                                }
                        ],
                        "request": {
                                "method": "PATCH",
                                "header": [
                                        {
                                                "key": "Content-Type",
                                                "value": "application/json",
                                                "type": "text"
                                        }
                                ],
                                "body": {
                                        "mode": "raw",
                                        "raw": "{\"completed\": true}"
                                },
                                "url": {
                                        "raw": "{{item}}",
                                        "host": [
                                                "{{item}}"
                                        ]
                                }
                        },
                        "response": []
                },
                {
                        "name": "Delete TODO item",
                        "event": [
                                {
                                        "listen": "prerequest",
                                        "script": {
                                                "exec": [
                                                        "console.log(\"deleting\", pm.environment.get(\"item\"))"
                                                ],
                                                "type": "text/javatest"
                                        }
                                },
                                {
                                        "listen": "test",
                                        "script": {
                                                "exec": [
                                                        "pm.test(\"Status code is 204 no content\", function () {",
                                                        "    pm.response.to.have.status(204);",
                                                        "});"
                                                ],
                                                "type": "text/javascript"
                                        }
                                }
                        ],
                        "request": {
                                "method": "DELETE",
                                "header": [],
                                "url": {
                                        "raw": "{{item}}",
                                        "host": [
                                                "{{item}}"
                                        ]
                                }
                        },
                        "response": []
                }
        ],
        "event": [
                {
                        "listen": "prerequest",
                        "script": {
                                "type": "text/javascript",
                                "exec": [
                                        ""
                                ]
                        }
                },
                {
                        "listen": "test",
                        "script": {
                                "type": "text/javascript",
                                "exec": [
                                        ""
                                ]
                        }
                }
        ],
        "variable": [
                {
                        "key": "uri",
                        "value": "http://34.74.127.60:8080/todos"
                },
                {
                        "key": "item",
                        "value": null
                }
        ]
    }

```

We can see that the test resource was created with Postman collection JSON content.

You can also check tests with the standard `kubectl` command which will list the tests Custom Resource.

```sh
kubectl get tests -n testkube test -oyaml
```

### Create a Test from Git

Some executors can handle files and some can handle only Git resources. You'll need to follow the particular executor **README** file to be aware which test types the executor handles.

Let's assume that a Cypress project is created in a Git repository - [examples](https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples) - where **examples** is a test directory in the `https://github.com/kubeshop/testkube-executor-cypress.git` repository.

Now we can create our Cypress-based test as shown below. In Git based tests, we need to pass the test type.

```sh
testkube create test --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project
```

```sh title="Expected output:"
Test created kubeshop-cypress 🥇
```

Let's check how the test created by Testkube is defined in the cluster:

```sh
$ kubectl get tests -n testkube kubeshop-cypress -o yaml
apiVersion: tests.testkube.io/v1
kind: Test
metadata:
  creationTimestamp: "2021-11-17T12:29:32Z"
  generation: 1
  name: kubeshop-cypress
  namespace: testkube
  resourceVersion: "225162"
  uid: f0d856aa-04fc-4238-bb4c-156ff82b4741
spec:
  description: some test description
  repository:
    branch: main
    path: examples
    type: git
    uri: https://github.com/kubeshop/testkube-executor-cypress.git
  type: cypress/project
```

As we can see, this test has `spec.repository` with Git repository data. This data can now be used by the executor to download test data.

### Providing Certificates

If the Git repository is using a self-signed certificate, you can provide the certificate using Kubernetes secrets and passing the secret name to the `--git-certificate-secret` flag.

In order to create a secret, use the following command:

```sh
kubectl create secret generic git-cert --from-file=git-cert.crt --from-file=git-cert.key
```

Then you can pass the secret name to the `--git-certificate-secret` flag and, during the test execution, the certificate will be mounted to the test container and added to the trusted authorities.

### Mapping Local Files

Local files can be added into a Testkube Test. This can be set on Test level passing the file in the format `source_path:destination_path` using the flag `--copy-files`. The file will be copied upon execution from the machine running `kubectl`. The files will be then available in the `/data/uploads` folder inside the test container.

```sh
testkube create test --name maven-example-file-test --git-uri https://github.com/kubeshop/testkube-executor-maven.git --git-path examples/hello-maven-settings --type maven/test  --git-branch main --copy-files "/Users/local_user/local_maven_settings.xml:settings.xml"
```

```sh title="Expected output:"
Test created maven-example-file-test 🥇
```

To run this test, refer to `settings.xml` from the `/data/uploads` folder:

```sh
testkube run test maven-example-file-test --args "--settings" --args "/data/uploads/settings.xml" -v "TESTKUBE_MAVEN=true" --args "-e" --args "-X" --env "DEBUG=\"true\""
```

By default, there is a 10 second timeout limit on all requests on the client side, and a 1 GB body size limit on the server side. To update the timeout, use `--upload-timeout` with [Go-compatible duration formats](https://pkg.go.dev/time#ParseDuration).

### Configuration Files

Some of the executors offer the option to set a special file using the flag `--variables-file` on both test creation and test run. For the Postman executor, this expects an environment file, for Maven it is `settings.xml`.
There are many differences between `--variables-file` and `--copy-files`. The former one sets this file directly as the configuration file. With the latter, there is an additional need to set the path explicitly on the arguments level. Another difference is that for variables files smaller than 128KB, this will be set on the CRD level and not uploaded to the object storage. This limitation comes from linux-based systems where this is the default maximum length of arguments.

### Redefining the Prebuilt Executor Command and Arguments

Each of Testkube Prebuilt executors has a default command and arguments it uses to execute the test. They are provided as a part of Executor CRD and can be either overridden, replaced, or appended during test creation or execution, for example:

```sh
testkube create test --name maven-example-test --git-uri https://github.com/kubeshop/testkube-executor-maven.git --git-path examples/hello-maven --type maven/test --git-branch main --command "mvn" --args-mode "override" --executor-args="--settings <settingsFile> <goalName> -Duser.home <mavenHome>"
```

```sh title="Expected output:"
Test created maven-example-test 🥇
```

#### Overriding the Command

As the above example showed, it is possible to override the original command of the Executor, as long as the executable is available in the Executor image. Use the `--command` parameter on test creation with the name of the executable:

```sh
$ testkube create test --help
...
      --command stringArray                        command passed to image in executor
...
```

#### Overriding the Arguments

There are two modes to pass arguments to the executor:

```sh
$ testkube create test --help
...
      --args-mode string                           usage mode for arguments. one of append|override|replace (default "append")
...
```

By default, `--args-mode` is set to `append`, which means that the default list will be kept, and whatever is set in `--executor-args` will be added to the end.

The `override` or `replace` mode will ignore the default arguments and use only what is set in `--executor-args`. If there are default values in between chevrons (`<>`), they can be reused in `--executor-args`.

When using `--args-mode` with `testkube run test ...` pay attention to set the arguments via the `--args` flag, not `--executor-args`.

### Changing the Default Job Template Used for Test Execution

You can always create your own custom executor with its own job template definition used for test execution. But sometimes you just need to adjust an existing job template of a standard Testkube executor with a few parameters. In this case you can use the additional parameter `--job-template` when you create or run the test:

```sh
testkube create test --git-branch main --git-uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name template-test --type cypress/project --job-template job.yaml
```

Where `job.yaml` file contains adjusted job template parts for merging with default job template:

```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      containers:
        - name: "{{ .Name }}"
          image: {{ .Image }}
          imagePullPolicy: Always
          command:
            - "/bin/runner"
            - '{{ .Jsn }}'
          resources:
            limits:
              memory: 128Mi
```

When you run such a test you will face a memory limit for the test executor pod, when the default job template doesn't have any resource constraints.

We also provide special helper methods to use in the job template:
`vartypeptrtostring` is the method to convert a pointer to a variable type to a string type.

Usage example:
```yaml
{{- range $key, $value := .Variables }}
  {{ if eq ($value.Type_ | vartypeptrtostring) "basic" }}
  - name: TEST
    value: "TEST"
  {{- end }}
{{- end }}
```

Add the `imagePullSecrets` option if you use your own Image Registry. This will add the secret for both `init` and `executor` containers.

```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      imagePullSecrets:
        - name: yourSecretName
```

### Changing the Default Slave Pod Template and Resource Specs Used for Slave Pods of Distrubited Executtors

For distributed executors you can adjust an existing slave pod specfication using template or resource fields. In this case you can use the additional parameters `--slave-pod-template` and/or `--slave-pod-requests-XXX`/`--slave-pod-limits-XXX` when you create or run the test:

```sh
testkube create test --git-branch develop --git-uri https://github.com/kubeshop/testkube.git --git-path contrib/executor/jmeterd/examples/gitflow --name distributed-jmeter --type jmeterd/test --slave-pod-template slave-pod.yaml --slave-pod-limits-cpu 20m --slave-pod-limits-memory 30Mi --slave-pod-requests-cpu 40m --slave-pod-requests-memory 50mi
```

Where `slave-pod.yaml` file contains adjusted slave pod template parts for merging with default slave pod template:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: "{{ .Name }}"
  namespace: {{ .Namespace }}
  ownerReferences:
  - apiVersion: batch/v1
    kind: job
    name: {{ .JobName }}
    uid: {{ .JobUID }}
  labels:
    distributed: jmeter
```

### Changing the Default CronJob Template Used for Scheduled Test Execution

Sometimes you need to adjust an existing cron job template for scheduled tests with a few parameters. In this case you can use additional parameter `--cronjob-template` when you create the test:

```sh
testkube create test --git-branch main --git-uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name template-test --type cypress/project --cronjob-template cronjob.yaml --schedule "*/5 * * * *"
```

Where `cronjob.yaml` file contains adjusted cron job template parts for merging with default cron job template:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  annotations:
    test: kube
```

When such a test is created you will see additional annotations for its cron job, if the default cron job template doesn't have any annotations.

### Executing a Prerun/Postrun Script

If you need to provide additional configuration for your executor environment, you can submit a prerun/postrun script to be executed before the test is started and after test is completed. For example, we have a simple shell script stored in `pre_script.sh` and `post_script.sh` files:

```sh
#!/bin/sh

echo "Storing ssl certificate in file from env secret env"
echo "$SSL_CERT" > /data/ssl.crt
```

and

```sh
#!/bin/sh

echo "Sleeping for 30 seconds"
sleep 30
```

Provide the script when you create or run the test using `--prerun-script` and `--postrun-script` parameters:

```sh
testkube create test --file test/postman/LocalHealth.postman_collection.json --name script-test --type postman/collection --prerun-script pre_script.sh --postrun-script post_script.sh --secret-env SSL_CERT=your-k8s-secret
```

For container executors you can define `--source-scripts` flag in order to run both scripts using `source` command in the same shell. 

### Adjusting Scraping Parameters

For any executor type you can specify additional scraping parameters using CLI or CRD definition. For example, below we request to scrape report directories, use a custom bucket to store test artifacts and ask to avoid using separate artifact folders for each test execution

```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: jmeter-smoke-test
  namespace: testkube
spec:
  type: jmeter/test
  content:
    type: git
    repository:
      type: git
      uri: https://github.com/kubeshop/testkube.git
      branch: main
      path: test/jmeter/executor-tests/jmeter-executor-smoke.jmx
  executionRequest:
    artifactRequest:
      dirs:
        - test/reports 
      storageBucket: jmeter-artifacts
      omitFolderPerExecution: true
```

### Changing the Default Scraper Job Template Used for Container Executor Tests

When you use container executor tests generating artifacts for scraping, we launch 2 sequential Kubernetes jobs. One is for test execution and other is for scraping test results. Sometimes you need to adjust an existing scraper job template of a standard Testkube scraper with a few parameters. In this case you can use the additional parameter `--scraper-template` when you create or run the test:

```sh
testkube create test --name scraper-test --type scraper/test --artifact-storage-class-name standard --artifact-volume-mount-path /share --artifact-dir test/files --scraper-template scraper.yaml
```

Where `scraper.yaml` file contains adjusted scraper job template parts for merging with default scraper job template:

```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      containers:
      - name: {{ .Name }}-scraper
        image: {{ .ScraperImage }}
        imagePullPolicy: Always
        command:
          - "/bin/runner"
          - '{{ .Jsn }}'
        {{- if .ArtifactRequest }}
          {{- if .ArtifactRequest.VolumeMountPath }}
        volumeMounts:
          - name: artifact-volume
            mountPath: {{ .ArtifactRequest.VolumeMountPath }}
          {{- end }}
        {{- end }}
        resources:
          limits:
            memory: 512Mi
```

When you run such a test, you will face a memory limit for the scraper pod, if the default scraper job template doesn't have any resource constraints.

### Mounting ConfigMap and Secret to Executor Pod

If you need to mount your ConfigMap and Secret to your executor environment, you can provide them as additional  
parameters when you create or run the test using the `--mount-configmap` and `--mount-secret` options:

```sh
testkube create test --file test/postman/LocalHealth.postman_collection.json --name mount-test --type postman/collection --mount-configmap your_configmap=/pod_mount_path --mount-secret your_secret=/pod_mount_path
```

### Automatically Add All ConfigMap and Secret Keys to Test Variables

You may want to automatcially add all keys from your ConfigMap and Secret to your test. For this, you will need to provide them as additional
parameters when you create or run the test using the `--variable-configmap` and `--variable-secret` options and they will be automatically added during test execution. All ConfigMap keys will be added with `key` as the variable name for Basic variables and all Secret keys will be added with key as variable name for Secret variables:

```sh
testkube create test --file test/postman/LocalHealth.postman_collection.json --name var-test --type postman/collection --variable-configmap your_configmap --variable-secret your_secret
```

### Run the Test in a Different Execution Namespace

When you need to run the test in a namespace different from the Testkube installation one, you can use a special Test CRD field `executionNamespace`, for example: 

```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: jmeter-smoke-test
  namespace: testkube
spec:
  type: jmeter/test
  content:
    type: git
    repository:
      type: git
      uri: https://github.com/kubeshop/testkube.git
      branch: main
      path: test/jmeter/executor-tests/jmeter-executor-smoke.jmx
  executionRequest:
    executionNamespace: default
```

You need to define execution namespaces in your helm chart values. It's possible to generate all required RBAC or just manually supply them.

```yaml
  executionNamespaces: []
       # -- Namespace for test execution  
  #  - namespace: default 
       # -- Whether to generate RBAC for testkube api server or use manually provided 
  #    generateAPIServerRBAC: true
       # -- Job service account name for test jobs 
  #    jobServiceAccountName: tests-job-default
       # -- Whether to generate RBAC for test job or use manually provided 
  #    generateTestJobRBAC: true
```

## Summary

Tests are the main abstractions over test suites in Testkube, they can be created with different sources and used by executors to run on top of a particular test framework.
