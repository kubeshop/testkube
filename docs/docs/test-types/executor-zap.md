import Admonition from "@theme/Admonition";

# OWASP Zed Attack Proxy Executor

Starting from version 1.12, Testkube has a dedicated executor for running ZAP tests. All you need to do is populate a file with the necessary parameters and create a Testkube test.

Default command for this executor is `<pythonScriptPath>`, which will be calculated based on the test type.

* "zap/baseline": "./zap-baseline.py"
* "zap/full": "./zap-full-scan.py"
* "zap/api":  "./zap-api-scan.py"

Default arguments for this executor command:  &lt;fileArgs&gt;

Parameters in &lt;&gt; are calculated at test execution:

* `<pythonScriptPath>` - calculated based on test type
* `<fileArgs>` - merged list of arguments from file input and `--args` input

[See more at "Redefining the Prebuilt Executor Command and Arguments" on the Creating Test page.](../articles/creating-tests.md#redefining-the-prebuilt-executor-command-and-arguments)

For more information on how the underlying Docker image behaves, please consult the documentation on the [official ZAP website](https://www.zaproxy.org/docs/docker/).

export const ExecutorInfo = () => {
  return (
    <div>
      <Admonition type="info" icon="🎓" title="What is ZAP?">
        <ul>
          <li>OWASP Zed Attack Proxy (ZAP) is a security scanner used to scan web applications.</li>
          <li>It is free and open-source, and also the most widely used web app scanner in the world. It provides a high range of options for security automation.</li>
        </ul>
      </Admonition>
    </div>
  );
}

## **Creating a ZAP Test**

The official ZAP Docker image, on which the executor was built, lets you run three types of tests: baseline, full and API scans. Depending on which of these functionalities you want to leverage, the test creation looks slightly different. Not only the type of the test (`--type`) needs to be specified differently, but the configuration file will also have some parameters that do not work with all types.

An example create test CLI command will look like:

```bash
$ kubectl testkube create test --file contrib/executor/zap/examples/zap-tk-api.yaml --type "zap/api" --name zap-api-test --copy-files contrib/executor/zap/examples/zap-tk-api.conf:zap-tk-api.conf
Test created testkube / zap-api-test 🥇
```

### **Input File**

Note that this command is using files available locally. The input file `--file contrib/executor/zap/examples/zap-tk-api.yaml`, will be the one defining the arguments the executable will run with. Possible values are:

| Variable name  | ZAP parameter  | Default value  | Baseline  | Full | API |
|---|---|---|---|---|---|
| target | -t | "" | &check; | &check; | &check; |
| config | -u for http <br /> -z for api <br /> -c for others | "" | &check; | &check; | &check; |
| debug | -d | false | &check; | &check; | &check; |
| short | -s | false | &check; | &check; | &check; |
| level | -l | "PASS" | &check; | &check; | &check; |
| context | -n | "" | &check; | &check; | &check; |
| user | -U | "" | &check; | &check; | &check; |
| delay | -D | 0 | &check; | &check; | &check; |
| time | -T | 0 | &check; | &check; | &check; |
| zap_options | -z | "" | &check; | &check; | &check; |
| fail_on_warn | -I | true | &check; | &check; | &check; |
| ajax | -j | false | &check; | &check; | |
| minutes | -m | 1 | &check; | &check; | |
| format | -f | "" | | | &check; |
| hostname | -O | "" | | | &check; |
| safe | -S | false | | | &check; |

An example file content would look like:

```bash
api:
  # -t the target API definition
  target: https://www.example.com/openapi.json
  # -f the API format, openapi, soap, or graphql
  format: openapi
  # -O the hostname to override in the (remote) OpenAPI spec
  hostname: https://www.example.com
  # -S safe mode this will skip the active scan and perform a baseline scan
  safe: true
  # -c config file
  config: /data/uploads/zap-tk-api.conf
  # -d show debug messages
  debug: true
  # -s short output
  short: false
  # -l minimum level to show: PASS, IGNORE, INFO, WARN or FAIL
  level: INFO
  # -n context file
  # context: /data/uploads/context.conf
  # username to use for authenticated scans
  user: anonymous
  # delay in seconds to wait for passive scanning
  delay: 5
  # max time in minutes to wait for ZAP to start and the passive scan to run
  time: 60
  # ZAP command line options
  zap_options: -config aaa=bbb
  # -I should ZAP fail on warnings
  fail_on_warn: false
```

The first line always specifies the type of the test. It can be either `baseline`, `full` or `api`.

### **Config File**

Another file that was passed in is `contrib/executor/zap/examples/zap-tk-api.conf`. This one is the rule configuration file. It should look something like:

```bash
# zap-api-scan rule configuration file
# Change WARN to IGNORE to ignore rule or FAIL to fail if rule matches
# Active scan rules set to IGNORE will not be run which will speed up the scan
# Only the rule identifiers are used - the names are just for info
# You can add your own messages to each rule by appending them after a tab on each line.
0   WARN	(Directory Browsing - Active/release)
10010	WARN	(Cookie No HttpOnly Flag - Passive/release)
10011	WARN	(Cookie Without Secure Flag - Passive/release)
10012	WARN	(Password Autocomplete in Browser - Passive/release)
10015	WARN	(Incomplete or No Cache-control and Pragma HTTP Header Set - Passive/release)
10016	WARN	(Web Browser XSS Protection Not Enabled - Passive/release)
10017	WARN	(Cross-Domain JavaScript Source File Inclusion - Passive/release)
10019	WARN	(Content-Type Header Missing - Passive/release)
10020	WARN	(X-Frame-Options Header Scanner - Passive/release)
10021	WARN	(X-Content-Type-Options Header Missing - Passive/release)
.
.
.
```

For additional files that should be passed in as parameter to the executor, use the `--copy-files` feature when developing locally. These files will be uploaded to the path `/data/uploads` by default.

## **Creating a Git-based ZAP Test**

For tests running on production, you should use a Git repository to keep track of the changes the test went through. When running these tests, Testkube will clone the repository every time. An example test creation command would look something like this:

```sh
testkube create test --git-uri https://github.com/kubeshop/testkube.git --type "zap/api" --name git-zap-api-test --executor-args "zap-api.yaml" --git-branch main --git-path contrib/executor/zap/examples
```

```sh title="Output:"
Test created testkube / git-zap-api-test 🥇
```

Please note that for using Git-based tests, the executor is expecting the test file as the last `--executor-args` argument. The path to the config file should be specified in this `yaml` file.

## **Running a ZAP Test**

Running a ZAP test using the CLI is as easy as running any other kind of test in Testkube:

```bash
$ kubectl testkube run test zap-api-test
Testkube will use the following file mappings: contrib/executor/zap/examples/zap-tk-api.conf:zap-tk-api.conf
Type:              zap/api
Name:              zap-api-test
Execution ID:      646f8d4bc5ba4e00f169aafe
Execution name:    zap-api-test-1
Execution number:  1
Status:            running
Start time:        2023-05-25 16:31:07.880658961 +0000 UTC
End time:          0001-01-01 00:00:00 +0000 UTC
Duration:          



Test execution started
Watch test execution until complete:
$ kubectl testkube watch execution zap-api-test-1


Use following command to get test execution details:
$ kubectl testkube get execution zap-api-test-1
```

## **Results**

When the test is finished running, it will have either `passed` or `failed` state. To check it, run:

```bash
testkube get tests
```

To get a more detailed report, you can run:

```bash
testkube get execution zap-api-test-1
```

Another indicator of how the tests completed are the report files created. These are uploaded as artifacts into the object storage used by Testkube, and can be retrieved using both the Testkube CLI and the Testkube UI.

## **References**

* [ZAP Docker documentation](https://www.zaproxy.org/docs/docker)
* [ZAP - Baseline Scan documentation](https://www.zaproxy.org/docs/docker/baseline-scan/)
* [ZAP - Full Scan documentation](https://www.zaproxy.org/docs/docker/full-scan/)
* [ZAP - API Scan documentation](https://www.zaproxy.org/docs/docker/api-scan/)
