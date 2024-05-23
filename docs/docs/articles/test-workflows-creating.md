# Creating Test Workflows

## CLI
Testkube CLI allows managing Test Workflows in the similar way as Test and TestSuites.

### Create
`testkube create testworkflow -f EXAMPLE_FILE.yaml`

#### kubectl apply
Alternatively, the `kubectl apply` can be used:
`kubectl apply -f EXAMPLE_FILE.yaml`

### Get
The Test Workflow details can be displayed using `testkube get testworkflow` command using the Test Workflow name:
`testkube get testworkflow TEST_WORKFLOW_NAME`

#### Filtering by Labels
Test Workflows can also be filtered using labels with `--label`:
`testkube get testworkflow --label example=label`

### Run
The Test Workflow can be run using the `testkube run testworkflow` command using Test Workflow name:
`testkube run testworkflow TEST_WORKFLOW_NAME`

Optionally, the follow option can be used to watch execution and get the log summary directly:
`testkube run testworkflow TEST_WORKFLOW_NAME -f`

### Delete
The Test Workflow can be deleted using the `testkube delete testworkflow` command using the Test Workflow name:
`testkube delete testworkflow TEST_WORKFLOW_NAME`

### Alias
`tw` alias can be used instead of `testworkflow` - for example:
`testkube get tw`

## Testkube Pro UI (Dashboard)
If you prefer to use the Dashboard, go to Test Workflows:

![menu test workflow icon](../img/dashboard-menu-workflows.png)

and click the `Add a new test workflow` button.

### Creation Options
Currently, the Test Workflow can be created from scratch, with the help of the wizard, by using an example or by importing YML.

![create test workflow selection](../img/dashboard-create-workflow-selection.png)

#### Wizard
The wizard consists of three steps:

##### Name & Type
To define a test, specify its name and choose from the available templates. Each template may come with predefined configuration values, which can be modified as needed.

![create test workflow from wizard - name & type step](../img/dashboard-create-workflow-from-wizard-name-type-step.png)

##### Source
Specify the source from which to fetch the data. Choose between Git, File, or String sources.

![create test workflow from wizard - source step](../img/dashboard-create-workflow-from-wizard-source-step.png)

##### Summary

Preview the YAML content of the test workflow, make changes if necessary, and create it.

![create test workflow from wizard - summary step](../img/dashboard-create-workflow-from-wizard-summary-step.png)

#### Example
You can choose one of the predefined examples and adjust it.

![create test workflow from example](../img/dashboard-create-workflow-from-example.png)

#### YML
You can also paste the complete Test Workflow definition

![create test workflow from yaml](../img/dashboard-create-workflow-from-yaml.png)

# Additional Test Workflow Examples
Additional Test Workflow examples can be found in the Testkube repository.

- [Cypress](https://github.com/kubeshop/testkube/blob/develop/test/cypress/executor-tests/crd-workflow/smoke.yaml)

- [Gradle](https://github.com/kubeshop/testkube/blob/develop/test/gradle/executor-smoke/crd-workflow/smoke.yaml)

- [JMeter](https://github.com/kubeshop/testkube/blob/develop/test/jmeter/executor-tests/crd-workflow/smoke.yaml)

- [k6](https://github.com/kubeshop/testkube/blob/develop/test/k6/executor-tests/crd-workflow/smoke.yaml)

- [Maven](https://github.com/kubeshop/testkube/blob/develop/test/maven/executor-smoke/crd-workflow/smoke.yaml)

- [Playwright](https://github.com/kubeshop/testkube/blob/develop/test/playwright/executor-tests/crd-workflow/smoke.yaml)

- [Postman](https://github.com/kubeshop/testkube/blob/develop/test/postman/executor-tests/crd-workflow/smoke.yaml)

- [SoapUI](https://github.com/kubeshop/testkube/blob/develop/test/soapui/executor-smoke/crd-workflow/smoke.yaml)
