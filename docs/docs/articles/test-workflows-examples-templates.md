# Test Workflows Examples - Templates

*It’s common to expose a snippet of configuration that could be reused across other TestWorkflows.*

## TestWorkflowTemplate

### Syntax

The templates have the same syntax as TestWorkflows, the only difference being that they can’t inherit other templates.

### Multiple Inheritance

Instead of the Executor approach, you may use multiple templates in a single TestWorkflow. Thanks to that, you can reuse each behavior separately. As an example, you may want to have template:

- Provide a template for common Cypress usage.
- Provide a template for closing Istio sidecar.
- Provide a template to attach labels per company department.
- Provide access to a data source.

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflowTemplate
metadata:
  name: close-istio
spec:
  after:
  - name: 'Close Istio sidecar'
    condition: always
    shell: 'touch /pod_control/job_finished'
```    

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: overview--example-15
spec:
  use:
  - name: 'close-istio'

  steps:
  - shell: 'tree /usr/bin'
```

## Configuration

The templates may be configured the same way as TestWorkflows, with OpenAPI-like specs.

### Usage

To pass the configuration, along with the template name,
simply pass the config too.

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflowTemplate
metadata:
  name: playwright
spec:
  config:
    version:
      type: string
      default: '1.32.3'
    workers:
      type: integer
      default: 2

  steps:
  - run:
      image: 'mcr.microsoft.com/playwright:v{{ config.version }}'
      shell: 'npm ci && npx playwright test --workers {{ config.workers }}'
```

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: overview--example-16
spec:
  content:
    git:
      uri: 'https://github.com/kubeshop/testkube'
      paths:
      - 'test/playwright/executor-tests/playwright-project'

  container:
    workingDir: '/data/repo/test/playwright/executor-tests/playwright-project'

  steps:
  - template:
      name: 'playwright'
      config:
        version: '1.33.3'
```

## Isolation (Expansion)

*There are 3 ways to include the TestWorkflowTemplate, that differ with the level of isolation (or rather - expansion).*

### Top Level - Use

A template can be included with the top-level use (array) clause - this way it will be included in the TestWorkflow, and all its defaults will be available in the whole TestWorkflow.

*This is the only place where constructs like Job and Pod setup can be specified.*

### Step Level - Use

When the template is included with use (array) on step level, all its defaults and steps will be included only for the step it’s included in.

### Step Level - Template

If you want to have a full isolation, so that no defaults will be expanded anywhere else (i.e., common envs for execution), you can use template (object) instruction. It’s not expanding any defaults outside the step.

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: overview--example-17
spec:
  use:
  - name: 'close-istio'
  - name: 'append-serviceaccount'

  steps:
  - use:
    - name: 'obtain-aws-credentials-envs'
    shell: 'echo $AWS_ACCESS_KEY_ID'

  - content:
      git:
        uri: 'https://github.com/kubeshop/testkube'
    workingDir: '/data/repo'
    template:
    - name: 'cypress'
```    


