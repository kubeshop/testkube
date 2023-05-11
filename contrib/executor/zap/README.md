![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to TestKube ZAP Executor

TestKube ZAP Executor is a test executor to run ZED attack proxy scans with [Testkube](https://testkube.io).  

## Usage

Your Testkube installation comes directly with the ZAP executor.

In case you want to build and deploy the executor yourself, you need to register and deploy it in your cluster.
```bash
kubectl apply -f examples/zap-executor.yaml
```

Issue the following commands to create and start a ZAP test for a given YAML configuration file:
```bash
kubectl testkube create test --file examples/zap-api.yaml --type "zap/api" --name api-test
kubectl testkube run test --watch api-test

kubectl testkube create test --file examples/zap-baseline.yaml --type "zap/baseline" --name baseline-test
kubectl testkube run test --watch baseline-test

kubectl testkube create test --file examples/zap-full.yaml --type "zap/full" --name full-test
kubectl testkube run test --watch full-test
```

The required ZAP arguments and options need to be specified via a dedicated YAML configuration file, e.g.
```yaml
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
  config: examples/zap-api.conf
  # -d show debug messages
  debug: true
  # -s short output
  short: false
  # -l minimum level to show: PASS, IGNORE, INFO, WARN or FAIL
  level: INFO
  # -c context file
  context: examples/context.config
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

# Issues and enchancements 

Please report any [issues](https://github.com/kubeshop/testkube/issues).
