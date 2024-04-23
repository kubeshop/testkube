# Test Workflow Template

Test Workflow Templates allow you to define common properties/traits that you can reuse 
consistently across your TestWorkflows. 

For example, with Test Workflow Templates you can easily 
- define specific ENV variables to be passed to all containers.
- define a common Git Source for your tests.
- define common setup or tearDown steps for your tests.
- define specific commands or images to use for your tests, while still allowing a Test Workflow to set
  the version of the testing tool.
- etc.

## How does it work?

A TestWorkflowTemplate definition follows the same structure as a Test Workflow, with the additional
ability to define configuration parameters that can be specified when the template is used.

TestWorkflowTemplates are "merged" into a referring Test Workflow at run-time and the 
resulting Test Workflow is executed by Testkube.

For example, the template below defines the basics for running a k6 test, with parameters for which
k6 version to use and which parameters to pass to k6:

```yaml
kind: TestWorkflowTemplate
apiVersion: testworkflows.testkube.io/v1
metadata:
  name: example-template--k6 # Template name (mandatory) - example-template/k6
spec:
  config: # default config values
    version:
      description: k6 version to use
      type: string
      default: 0.49.0
    params:
      description: Additional params for the k6 run command
      type: string
      default: ""
  steps: # steps to be executed
  - name: Run k6 tests
    container:
      image: grafana/k6:{{ config.version }} # default values are assigned
    shell: k6 run {{ config.params }}
```

The template can then be "used" in a corresponding Test Workflow step:

```yaml
steps:
  - name: Run from template
    workingDir: /data/repo/test/k6/executor-tests
    template: # template will be executed here
      name: example-template/k6 # template name
      config: # template config - values passed to Test Workflow Template
        version: 0.48.0 # version override
        params: "k6-smoke-test-without-envs.js"
```

Full example:
```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: k6-example-from-template
spec:
  container:
    resources:
      requests:
        cpu: 128m
        memory: 128Mi
    workingDir: /data/repo/test/k6/executor-tests
  steps:
  - name: Checkout
    content:
      git:
        uri: https://github.com/kubeshop/testkube
        revision: main
        paths:
        - test/k6/executor-tests/k6-smoke-test-without-envs.js
  - name: Run from template
    workingDir: /data/repo/test/k6/executor-tests
    template: # template will be executed here
      name: example-template/k6 # template name
      config: # template config - values passed to Test Workflow Template
        version: 0.48.0 # version override
        params: "k6-smoke-test-without-envs.js"
```

# Example Test Workflow Templates
Example Test Workflow Templates can be found in the Testkube repository:

- [Cypress](https://github.com/kubeshop/testkube/blob/develop/test/test-workflow-templates/cypress.yaml)
- [k6](https://github.com/kubeshop/testkube/blob/develop/test/test-workflow-templates/k6.yaml)
- [postman](https://github.com/kubeshop/testkube/blob/develop/test/test-workflow-templates/cypress.yaml)
