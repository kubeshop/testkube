# CRDs Reference

CRDs (Custom Resource Definitions) reference. Read more Testkube's CRDs in [Testkube Custom Resources](./crds.md) section.

## Packages

- [executor.testkube.io/v1](#executortestkubeiov1)
- [tests.testkube.io/v1](#teststestkubeiov1)
- [tests.testkube.io/v2](#teststestkubeiov2)
- [tests.testkube.io/v3](#teststestkubeiov3)

## executor.testkube.io/v1

Package v1 contains API Schema definitions for the executor v1 API group.

### Resource Types

- [Executor](#executor)
- [ExecutorList](#executorlist)
- [Webhook](#webhook)
- [WebhookList](#webhooklist)

#### EventType

_Underlying type:_ `string`

_Appears in:_

- [WebhookSpec](#webhookspec)

#### Executor

Executor is the Schema for the executors API.

_Appears in:_

- [ExecutorList](#executorlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                    | `Executor`                                                      |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ExecutorSpec](#executorspec)_                                                                             |                                                                 |

#### ExecutorList

ExecutorList contains a list of Executors.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                | `ExecutorList`                                                  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Executor](#executor) array_                                                                          |                                                                 |

#### ExecutorMeta

Executor meta data.

_Appears in:_

- [ExecutorSpec](#executorspec)

| Field                                            | Description           |
| ------------------------------------------------ | --------------------- |
| `iconURI` _string_                               | URI for executor icon |
| `docsURI` _string_                               | URI for executor docs |
| `tooltips` _object (keys:string, values:string)_ | executor tooltips     |

#### ExecutorSpec

ExecutorSpec defines the desired state of the Executor.

_Appears in:_

- [Executor](#executor)

| Field                                                                                                                                                | Description                                                                                                                                            |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `types` _string array_                                                                                                                               | Types defines what types can be handled by executor e.g. "postman/collection", ":curl/command", etc.                                                     |
| `executor_type` _[ExecutorType](#executortype)_                                                                                                      | ExecutorType one of "rest" for rest openapi based executors or "job" which will be default runners for testkube or "container" for container executors. |
| `uri` _string_                                                                                                                                       | URI for rest-based executors.                                                                                                                           |
| `image` _string_                                                                                                                                     | Image for kube-job.                                                                                                                                     |
| `args` _string array_                                                                                                                                | Executor binary arguments.                                                                                                                              |
| `command` _string array_                                                                                                                             | Executor default binary command.                                                                                                                        |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ | Container executor default image pull secrets.                                                                                                          |
| `features` _[Feature](#feature) array_                                                                                                               | Features list of Possible features which the executor handles.                                                                                              |
| `content_types` _[ScriptContentType](#scriptcontenttype) array_                                                                                      | ContentTypes lists the handled content types.                                                                                                             |
| `job_template` _string_                                                                                                                              | Job template to launch executor.                                                                                                                        |
| `meta` _[ExecutorMeta](#executormeta)_                                                                                                               | Meta data about the executor.                                                                                                                               |

#### ExecutorType

_Underlying type:_ `string`

_Appears in:_

- [ExecutorSpec](#executorspec)

#### Feature

_Underlying type:_ `string`

_Appears in:_

- [ExecutorSpec](#executorspec)

#### ScriptContentType

_Underlying type:_ `string`

_Appears in:_

- [ExecutorSpec](#executorspec)

#### Webhook

Webhook is the Schema for the webhooks API.

_Appears in:_

- [WebhookList](#webhooklist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                    | `Webhook`                                                       |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[WebhookSpec](#webhookspec)_                                                                               |                                                                 |

#### WebhookList

WebhookList contains a list of Webhooks.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                | `WebhookList`                                                   |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Webhook](#webhook) array_                                                                            |                                                                 |

#### WebhookSpec

WebhookSpec defines the desired state of a Webhook.

_Appears in:_

- [Webhook](#webhook)

| Field                                           | Description                                                        |
| ----------------------------------------------- | ------------------------------------------------------------------ |
| `uri` _string_                                  | The URI is the address where the webhook should be made.                        |
| `events` _[EventType](#eventtype) array_        | Events declares a list of events on which webhook should be called.    |
| `selector` _string_                             | Labels to filter for tests and test suites.                         |
| `payloadObjectField` _string_                   | Will load the generated payload for notification inside the object. |
| `payloadTemplate` _string_                      | Golang based template for notification payload.                     |
| `headers` _object (keys:string, values:string)_ | Webhook headers.                                                    |

## tests.testkube.io/v1

Package v1 contains API Schema definitions for the testkube v1 API group.

### Resource Types

- [Script](#script)
- [ScriptList](#scriptlist)
- [Test](#test)
- [TestList](#testlist)
- [TestSource](#testsource)
- [TestSourceList](#testsourcelist)
- [TestSuite](#testsuite)
- [TestSuiteList](#testsuitelist)
- [TestTrigger](#testtrigger)
- [TestTriggerList](#testtriggerlist)

#### GitAuthType

_Underlying type:_ `string`

GitAuthType defines git auth type.

_Appears in:_

- [Repository](#repository)

#### Repository

Repository represents VCS repo, currently we're handling Git only.

_Appears in:_

- [TestSourceSpec](#testsourcespec)

| Field                                      | Description                                                                              |
| ------------------------------------------ | ---------------------------------------------------------------------------------------- |
| `type` _string_                            | VCS repository type.                                                                      |
| `uri` _string_                             | URI of content file or Git directory.                                                     |
| `branch` _string_                          | Branch/tag name for checkout.                                                             |
| `commit` _string_                          | Commit id (sha) for checkout.                                                             |
| `path` _string_                            | If needed, we can checkout a particular path (dir or file) in case of BIG/mono repositories. |
| `usernameSecret` _[SecretRef](#secretref)_ |                                                                                          |
| `tokenSecret` _[SecretRef](#secretref)_    |                                                                                          |
| `certificateSecret` _string_               | Git auth certificate, secret for private repositories.                                     |
| `workingDir` _string_                      | If provided, we checkout the whole repository and run the test from this directory.            |
| `authType` _[GitAuthType](#gitauthtype)_   | Auth type for Git requests.                                                               |

#### Script

Script is the Schema for the scripts API.

_Appears in:_

- [ScriptList](#scriptlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `Script`                                                        |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ScriptSpec](#scriptspec)_                                                                                 |                                                                 |

#### ScriptList

ScriptList contains a list of scripts.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `ScriptList`                                                    |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Script](#script) array_                                                                              |                                                                 |

#### ScriptSpec

ScriptSpec defines the desired state of a script.

_Appears in:_

- [Script](#script)

| Field                                          | Description                                                                                                                                                           |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `type` _string_                                | Script type.                                                                                                                                                           |
| `name` _string_                                | Script execution custom name.                                                                                                                                          |
| `params` _object (keys:string, values:string)_ | Execution params passed to executor.                                                                                                                                   |
| `content` _string_                             | Script content as string (content depends on the executor).                                                                                                              |
| `input-type` _string_                          | Script content type can be: (1) direct content - created from file, (2) Git repo directory checkout, in case the test is some kind of project or has more than one file. |
| `repository` _[Repository](#repository)_       | Repository details, if they exist.                                                                                                                                          |
| `tags` _string array_                          |                                                                                                                                                                       |

#### SecretRef

Testkube internal reference for secret storage in Kubernetes secrets.

_Appears in:_

- [Repository](#repository)

| Field                | Description                 |
| -------------------- | --------------------------- |
| `namespace` _string_ | object kubernetes namespace |
| `name` _string_      | object name                 |
| `key` _string_       | object key                  |

#### Test

Test is the Schema for the tests API.

_Appears in:_

- [TestList](#testlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `Test`                                                          |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSpec](#testspec)_                                                                                     |                                                                 |

#### TestList

TestList contains a list of Tests.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestList`                                                      |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Test](#test) array_                                                                                  |                                                                 |

#### TestSource

TestSource is the the Schema for the testsources API.

_Appears in:_

- [TestSourceList](#testsourcelist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `TestSource`                                                    |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSourceSpec](#testsourcespec)_                                                                         |                                                                 |

#### TestSourceList

TestSourceList contains a list of TestSources.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestSourceList`                                                |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestSource](#testsource) array_                                                                      |                                                                 |

#### TestSourceSpec

TestSourceSpec defines the desired state of TestSource.

_Appears in:_

- [TestSource](#testsource)

| Field                                      | Description                |
| ------------------------------------------ | -------------------------- |
| `type` _[TestSourceType](#testsourcetype)_ |                            |
| `repository` _[Repository](#repository)_   | repository of test content |
| `data` _string_                            | test content body          |
| `uri` _string_                             | uri of test content        |

#### TestSourceType

_Underlying type:_ `string`

_Appears in:_

- [TestSourceSpec](#testsourcespec)

#### TestSpec

TestSpec defines the desired state of the Test.

_Appears in:_

- [Test](#test)

| Field                                          | Description                                                             |
| ---------------------------------------------- | ----------------------------------------------------------------------- |
| `before` _[TestStepSpec](#teststepspec) array_ | Before steps is a list of scripts which will be sequentially orchestrated. |
| `steps` _[TestStepSpec](#teststepspec) array_  | Steps is a list of scripts which will be sequentially orchestrated.        |
| `after` _[TestStepSpec](#teststepspec) array_  | After steps is a list of scripts which will be sequentially orchestrated.  |
| `repeats` _integer_                            |                                                                         |
| `description` _string_                         |                                                                         |
| `tags` _string array_                          |                                                                         |

#### TestStepDelay

_Appears in:_

- [TestStepSpec](#teststepspec)

| Field                | Description    |
| -------------------- | -------------- |
| `duration` _integer_ | Duration in ms |

#### TestStepExecute

_Appears in:_

- [TestStepSpec](#teststepspec)

| Field                     | Description |
| ------------------------- | ----------- |
| `namespace` _string_      |             |
| `name` _string_           |             |
| `stopOnFailure` _boolean_ |             |

#### TestStepSpec

TestStepSpec of particular type will have config for possible step types.

_Appears in:_

- [TestSpec](#testspec)

| Field                                           | Description |
| ----------------------------------------------- | ----------- |
| `type` _string_                                 |             |
| `execute` _[TestStepExecute](#teststepexecute)_ |             |
| `delay` _[TestStepDelay](#teststepdelay)_       |             |

#### TestSuite

TestSuite is the Schema for the testsuites API.

_Appears in:_

- [TestSuiteList](#testsuitelist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `TestSuite`                                                     |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSuiteSpec](#testsuitespec)_                                                                           |                                                                 |

#### TestSuiteList

TestSuiteList contains a list of TestSuites.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestSuiteList`                                                 |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestSuite](#testsuite) array_                                                                        |                                                                 |

#### TestSuiteSpec

TestSuiteSpec defines the desired state of a TestSuite.

_Appears in:_

- [TestSuite](#testsuite)

| Field                                                            | Description                                                           |
| ---------------------------------------------------------------- | --------------------------------------------------------------------- |
| `before` _[TestSuiteStepSpec](#testsuitestepspec) array_         | Before steps is a list of tests which will be sequentially orchestrated. |
| `steps` _[TestSuiteStepSpec](#testsuitestepspec) array_          | Steps is a list of tests which will be sequentially orchestrated.        |
| `after` _[TestSuiteStepSpec](#testsuitestepspec) array_          | After steps is a list of tests which will be sequentially orchestrated.  |
| `repeats` _integer_                                              |                                                                       |
| `description` _string_                                           |                                                                       |
| `schedule` _string_                                              | Schedule in cron job format for scheduled test execution.              |
| `params` _object (keys:string, values:string)_                   | DEPRECATED execution params passed to the executor.                        |
| `variables` _object (keys:string, values:[Variable](#variable))_ | Variables are new params with secrets attached.                        |

#### TestSuiteStepDelay

TestSuiteStepDelay contains step delay parameters.

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                | Description    |
| -------------------- | -------------- |
| `duration` _integer_ | Duration in ms |

#### TestSuiteStepExecute

TestSuiteStepExecute defines the step to be executed.

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                     | Description |
| ------------------------- | ----------- |
| `namespace` _string_      |             |
| `name` _string_           |             |
| `stopOnFailure` _boolean_ |             |

#### TestSuiteStepSpec

TestSuiteStepSpec of a particular type will have config for possible step types.

_Appears in:_

- [TestSuiteSpec](#testsuitespec)

| Field                                                     | Description |
| --------------------------------------------------------- | ----------- |
| `type` _string_                                           |             |
| `execute` _[TestSuiteStepExecute](#testsuitestepexecute)_ |             |
| `delay` _[TestSuiteStepDelay](#testsuitestepdelay)_       |             |

#### TestTrigger

TestTrigger is the Schema for the testtriggers API.

_Appears in:_

- [TestTriggerList](#testtriggerlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `TestTrigger`                                                   |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestTriggerSpec](#testtriggerspec)_                                                                       |                                                                 |

#### TestTriggerAction

_Underlying type:_ `string`

TestTriggerAction defines action for test triggers.

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerCondition

TestTriggerCondition is used for definition of the condition for test triggers.

_Appears in:_

- [TestTriggerConditionSpec](#testtriggerconditionspec)

| Field             | Description                                                                         |
| ----------------- | ----------------------------------------------------------------------------------- |
| `type` _string_   | Test trigger condition.                                                              |
| `reason` _string_ | Test trigger condition reason.                                                       |
| `ttl` _integer_   | Duration in seconds in the past from current time when the condition is still valid. |

#### TestTriggerConditionSpec

TestTriggerConditionSpec defines the condition specification for the TestTrigger.

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

| Field                                                              | Description                                                                  |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| `conditions` _[TestTriggerCondition](#testtriggercondition) array_ | List of test trigger conditions.                                              |
| `timeout` _integer_                                                | Duration in seconds the test trigger waits for conditions, until it is stopped. |

#### TestTriggerEvent

_Underlying type:_ `string`

TestTriggerEvent defines an event for test triggers.

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerExecution

_Underlying type:_ `string`

TestTriggerExecution defines execution for test triggers.

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerList

TestTriggerList contains a list of TestTriggers.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestTriggerList`                                               |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestTrigger](#testtrigger) array_                                                                    |                                                                 |

#### TestTriggerResource

_Underlying type:_ `string`

TestTriggerResource defines the resource for test triggers.

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerSelector

TestTriggerSelector is used for selecting Kubernetes Objects.

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

| Field                                                                                                                         | Description                                                                                    |
| ----------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `name` _string_                                                                                                               | Name selector is used to identify a Kubernetes Object based on the metadata name.               |
| `namespace` _string_                                                                                                          | Namespace of the Kubernetes object.                                                             |
| `labelSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#labelselector-v1-meta)_ | LabelSelector is used to identify a group of Kubernetes Objects based on their metadata labels. |

#### TestTriggerSpec

TestTriggerSpec defines the desired state of a TestTrigger.

_Appears in:_

- [TestTrigger](#testtrigger)

| Field                                                                                                       | Description                                                                                               |
| ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `resource` _[TestTriggerResource](#testtriggerresource)_                                                    | Defines the Resource monitor Event which triggers an Action on certain conditions.                     |
| `resourceSelector` _[TestTriggerSelector](#testtriggerselector)_                                            | ResourceSelector identifies which Kubernetes Objects should be watched.                                    |
| `event` _[TestTriggerEvent](#testtriggerevent)_                                                             | Defines the Event on which a Resource an Action should be triggered.                                               |
| `conditionSpec` _[TestTriggerConditionSpec](#testtriggerconditionspec)_                                     | Which resource conditions should be matched.                                                                |
| `action` _[TestTriggerAction](#testtriggeraction)_                                                          | Action represents what needs to be executed for a selected Execution.                                        |
| `execution` _[TestTriggerExecution](#testtriggerexecution)_                                                 | Execution identifies which test execution an Action should be executed for.                                |
| `testSelector` _[TestTriggerSelector](#testtriggerselector)_                                                | TestSelector identifies on which Testkube Kubernetes Objects an Action should be taken.                    |
| `delay` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#duration-v1-meta)_ | Delay is a duration string which specifies how long the test should be delayed after a trigger is matched. |

#### Variable

_Appears in:_

- [ExecutionRequest](#executionrequest)
- [TestSpec](#testspec)
- [TestSuiteExecutionRequest](#testsuiteexecutionrequest)
- [TestSuiteSpec](#testsuitespec)

| Field                                                                                                                   | Description                |
| ----------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| `type` _string_                                                                                                         | variable type              |
| `name` _string_                                                                                                         | variable name              |
| `value` _string_                                                                                                        | variable string value      |
| `valueFrom` _[EnvVarSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvarsource-v1-core)_ | Load variable from var source. |

## tests.testkube.io/v2

Package v2 contains API Schema definitions for the Testkube v2 API group.

### Resource Types

- [Script](#script)
- [ScriptList](#scriptlist)
- [Test](#test)
- [TestList](#testlist)
- [TestSuite](#testsuite)
- [TestSuiteList](#testsuitelist)

#### Repository

Repository represents the VCS repo, currently we're handling Git only.

_Appears in:_

- [TestContent](#testcontent)

| Field               | Description                                                                              |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `type` _string_     | VCS repository type.                                                                      |
| `uri` _string_      | URI of content file or Git directory.                                                     |
| `branch` _string_   | Branch/tag name for checkout.                                                             |
| `commit` _string_   | Commit ID (sha) for checkout.                                                             |
| `path` _string_     | If needed, we can checkout a particular path (dir or file) in case of BIG/mono repositories. |
| `username` _string_ | Git auth username for private repositories.                                               |
| `token` _string_    | Git auth token for private repositories.                                                  |

#### RunningContext

Running context for test or test suite execution.

_Appears in:_

- [TestSuiteExecutionRequest](#testsuiteexecutionrequest)

| Field                                              | Description                           |
| -------------------------------------------------- | ------------------------------------- |
| `type` _[RunningContextType](#runningcontexttype)_ | One of possible context types.         |
| `context` _string_                                 | Context value depending from its type. |

#### RunningContextType

_Underlying type:_ `string`

_Appears in:_

- [RunningContext](#runningcontext)

#### Script

Script is the Schema for the scripts API.

_Appears in:_

- [ScriptList](#scriptlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                    | `Script`                                                        |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ScriptSpec](#scriptspec)_                                                                                 |                                                                 |

#### ScriptContent

_Appears in:_

- [ScriptSpec](#scriptspec)

| Field                                    | Description                  |
| ---------------------------------------- | ---------------------------- |
| `type` _string_                          | script type                  |
| `repository` _[Repository](#repository)_ | repository of script content |
| `data` _string_                          | script content body          |
| `uri` _string_                           | URI of script content        |

#### ScriptList

ScriptList contains a list of Scripts.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                | `ScriptList`                                                    |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Script](#script) array_                                                                              |                                                                 |

#### ScriptSpec

ScriptSpec defines the desired state of a Script.

_Appears in:_

- [Script](#script)

| Field                                          | Description                         |
| ---------------------------------------------- | ----------------------------------- |
| `type` _string_                                | script type                         |
| `name` _string_                                | script execution custom name        |
| `params` _object (keys:string, values:string)_ | execution params passed to executor |
| `content` _[ScriptContent](#scriptcontent)_    | script content object               |
| `tags` _string array_                          | script tags                         |

#### Test

Test is the Schema for the tests API.

_Appears in:_

- [TestList](#testlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                    | `Test`                                                          |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSpec](#testspec)_                                                                                     |                                                                 |

#### TestContent

TestContent defines the test content.

_Appears in:_

- [TestSpec](#testspec)

| Field                                    | Description                |
| ---------------------------------------- | -------------------------- |
| `type` _string_                          | test type                  |
| `repository` _[Repository](#repository)_ | repository of test content |
| `data` _string_                          | test content body          |
| `uri` _string_                           | uri of test content        |

#### TestList

TestList contains a list of Tests.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                | `TestList`                                                      |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Test](#test) array_                                                                                  |                                                                 |

#### TestSpec

TestSpec defines the desired state of a Test.

_Appears in:_

- [Test](#test)

| Field                                                            | Description                                              |
| ---------------------------------------------------------------- | -------------------------------------------------------- |
| `type` _string_                                                  | test type                                                |
| `name` _string_                                                  | test execution custom name                               |
| `params` _object (keys:string, values:string)_                   | DEPRECATED execution params passed to executor.           |
| `variables` _object (keys:string, values:[Variable](#variable))_ | Variables are new params with secrets attached.           |
| `content` _[TestContent](#testcontent)_                          | test content object                                      |
| `schedule` _string_                                              | Schedule in cron job format for scheduled test execution. |
| `executorArgs` _string array_                                    | Additional executor binary arguments.                     |

#### TestSuite

TestSuite is the Schema for the testsuites API.

_Appears in:_

- [TestSuiteList](#testsuitelist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                    | `TestSuite`                                                     |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSuiteSpec](#testsuitespec)_                                                                           |                                                                 |

#### TestSuiteExecutionCore

The test suite execution core.

_Appears in:_

- [TestSuiteStatus](#testsuitestatus)

| Field                                                                                                   | Description                     |
| ------------------------------------------------------------------------------------------------------- | ------------------------------- |
| `id` _string_                                                                                           | execution ID                    |
| `startTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | test suite execution start time |
| `endTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_   | test suite execution end time   |

#### TestSuiteExecutionRequest

The test suite execution request body.

_Appears in:_

- [TestSuiteSpec](#testsuitespec)

| Field                                                            | Description                                           |
| ---------------------------------------------------------------- | ----------------------------------------------------- |
| `name` _string_                                                  | The test execution custom name.                            |
| `namespace` _string_                                             | The test Kubernetes namespace (\"testkube\" when not set). |
| `variables` _object (keys:string, values:[Variable](#variable))_ |                                                       |
| `secretUUID` _string_                                            | secret UUID                                           |
| `labels` _object (keys:string, values:string)_                   | test suite labels                                     |
| `executionLabels` _object (keys:string, values:string)_          | execution labels                                      |
| `sync` _boolean_                                                 | Whether to start execution sync or async.              |
| `httpProxy` _string_                                             | HTTP proxy for executor containers                    |
| `httpsProxy` _string_                                            | HTTPS proxy for executor containers                   |
| `timeout` _integer_                                              | Timeout for test suite execution.                      |
| `runningContext` _[RunningContext](#runningcontext)_             |                                                       |
| `cronJobTemplate` _string_                                       | Cron job template extensions.                          |

#### TestSuiteList

TestSuiteList contains a list of TestSuites.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                | `TestSuiteList`                                                 |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestSuite](#testsuite) array_                                                                        |                                                                 |

#### TestSuiteSpec

TestSuiteSpec defines the desired state of a TestSuite.

_Appears in:_

- [TestSuite](#testsuite)

| Field                                                                        | Description                                                           |
| ---------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| `before` _[TestSuiteStepSpec](#testsuitestepspec) array_                     | Before steps is a list of tests which will be sequentially orchestrated. |
| `steps` _[TestSuiteStepSpec](#testsuitestepspec) array_                      | Steps is a list of tests which will be sequentially orchestrated.        |
| `after` _[TestSuiteStepSpec](#testsuitestepspec) array_                      | After steps is a list of tests which will be sequentially orchestrated.  |
| `repeats` _integer_                                                          |                                                                       |
| `description` _string_                                                       |                                                                       |
| `schedule` _string_                                                          | Schedule in cron job format for scheduled test execution.              |
| `executionRequest` _[TestSuiteExecutionRequest](#testsuiteexecutionrequest)_ |                                                                       |

#### TestSuiteStepDelay

TestSuiteStepDelay contains step delay parameters.

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                | Description    |
| -------------------- | -------------- |
| `duration` _integer_ | Duration in ms |

#### TestSuiteStepExecute

TestSuiteStepExecute defines the step to be executed.

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                     | Description |
| ------------------------- | ----------- |
| `namespace` _string_      |             |
| `name` _string_           |             |
| `stopOnFailure` _boolean_ |             |

#### TestSuiteStepSpec

TestSuiteStepSpec for a particular type will have the config for possible step types.

_Appears in:_

- [TestSuiteSpec](#testsuitespec)

| Field                                                     | Description |
| --------------------------------------------------------- | ----------- |
| `type` _[TestSuiteStepType](#testsuitesteptype)_          |             |
| `execute` _[TestSuiteStepExecute](#testsuitestepexecute)_ |             |
| `delay` _[TestSuiteStepDelay](#testsuitestepdelay)_       |             |

#### TestSuiteStepType

_Underlying type:_ `string`

TestSuiteStepType defines different types of test suite steps.

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

#### Variable

_Appears in:_

- [ExecutionRequest](#executionrequest)
- [TestSpec](#testspec)
- [TestSuiteExecutionRequest](#testsuiteexecutionrequest)
- [TestSuiteSpec](#testsuitespec)

| Field                                                                                                                   | Description                |
| ----------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| `type` _string_                                                                                                         | variable type              |
| `name` _string_                                                                                                         | variable name              |
| `value` _string_                                                                                                        | variable string value      |
| `valueFrom` _[EnvVarSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvarsource-v1-core)_ | or load it from var source |

## tests.testkube.io/v3

Package v3 contains API Schema definitions for the tests v3 API group.

### Resource Types

- [Test](#test)
- [TestList](#testlist)

#### ArgsModeType

_Underlying type:_ `string`

ArgsModeType defines the args mode type.

_Appears in:_

- [ExecutionRequest](#executionrequest)

#### ArtifactRequest

Artifact request body with test artifacts.

_Appears in:_

- [ExecutionRequest](#executionrequest)

| Field                       | Description                                        |
| --------------------------- | -------------------------------------------------- |
| `storageClassName` _string_ | The artifact storage class name for the container executor. |
| `volumeMountPath` _string_  | The artifact volume mount path for the container executor.  |
| `dirs` _string array_       | The artifact directories for scraping.                  |

#### EnvReference

Reference to env resource.

_Appears in:_

- [ExecutionRequest](#executionrequest)

| Field                                                                                                                                   | Description                                     |
| --------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| `reference` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core)_ |                                                 |
| `mount` _boolean_                                                                                                                       | Whether we should mount a resource.                 |
| `mountPath` _string_                                                                                                                    | Where we should mount resource.                   |
| `mapToVariables` _boolean_                                                                                                              | Whether we should map to variables from a resource. |

#### ExecutionCore

The test execution core.

_Appears in:_

- [TestStatus](#teststatus)

| Field                                                                                                   | Description      |
| ------------------------------------------------------------------------------------------------------- | ---------------- |
| `id` _string_                                                                                           | execution id     |
| `number` _integer_                                                                                      | execution number |
| `startTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | test start time  |
| `endTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_   | test end time    |

#### ExecutionRequest

The test execution request body.

_Appears in:_

- [TestSpec](#testspec)

| Field                                                                                                                                                | Description                                                                                                                                                                                                  |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `name` _string_                                                                                                                                      | The test execution custom name.                                                                                                                                                                                   |
| `testSuiteName` _string_                                                                                                                             | The unique test suite name (CRD Test suite name), if it's run as a part of a test suite.                                                                                                                            |
| `number` _integer_                                                                                                                                   | The test execution number.                                                                                                                                                                                        |
| `executionLabels` _object (keys:string, values:string)_                                                                                              | The test execution labels.                                                                                                                                                                                        |
| `namespace` _string_                                                                                                                                 | The test Kubernetes namespace (\"testkube\" when not set).                                                                                                                                                        |
| `variablesFile` _string_                                                                                                                             | Variables file content - needs to be in the format for a particular executor (e.g. postman envs file).                                                                                                               |
| `isVariablesFileUploaded` _boolean_                                                                                                                  |                                                                                                                                                                                                              |
| `variables` _object (keys:string, values:[Variable](#variable))_                                                                                     |                                                                                                                                                                                                              |
| `testSecretUUID` _string_                                                                                                                            | test secret UUID                                                                                                                                                                                             |
| `testSuiteSecretUUID` _string_                                                                                                                       | The test suite secret uuid, if it's run as a part of a test suite.                                                                                                                                                  |
| `args` _string array_                                                                                                                                | Additional executor binary arguments.                                                                                                                                                                         |
| `argsMode` _[ArgsModeType](#argsmodetype)_                                                                                                           | Usage mode for arguments.                                                                                                                                                                                     |
| `command` _string array_                                                                                                                             | Executor binary command.                                                                                                                                                                                      |
| `image` _string_                                                                                                                                     | Container executor image.                                                                                                                                                                                     |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ | Container executor image pull secrets.                                                                                                                                                                        |
| `envs` _object (keys:string, values:string)_                                                                                                         | Environment variables passed to executor. Deprecated: use Basic Variables instead.                                                                                                                            |
| `secretEnvs` _object (keys:string, values:string)_                                                                                                   | Execution variables passed to executor from secrets. Deprecated: use Secret Variables instead.                                                                                                                |
| `sync` _boolean_                                                                                                                                     | Whether to start execution sync or async.                                                                                                                                                                     |
| `httpProxy` _string_                                                                                                                                 | HTTP proxy for executor containers.                                                                                                                                                                           |
| `httpsProxy` _string_                                                                                                                                | HTTPS proxy for executor containers.                                                                                                                                                                          |
| `negativeTest` _boolean_                                                                                                                             | A negative test will fail the execution if it is a success and it will succeed if it is a failure.                                                                                                              |
| `activeDeadlineSeconds` _integer_                                                                                                                    | Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer. |
| `artifactRequest` _[ArtifactRequest](#artifactrequest)_                                                                                              |                                                                                                                                                                                                              |
| `jobTemplate` _string_                                                                                                                               | job template extensions                                                                                                                                                                                      |
| `cronJobTemplate` _string_                                                                                                                           | cron job template extensions                                                                                                                                                                                 |
| `preRunScript` _string_                                                                                                                              | The script to run before test execution.                                                                                                                                                                          |
| `scraperTemplate` _string_                                                                                                                           | scraper template extensions                                                                                                                                                                                  |
| `envConfigMaps` _[EnvReference](#envreference) array_                                                                                                | config map references                                                                                                                                                                                        |
| `envSecrets` _[EnvReference](#envreference) array_                                                                                                   | secret references                                                                                                                                                                                            |
| `runningContext` _[RunningContext](#runningcontext)_                                                                                                 |                                                                                                                                                                                                              |

#### GitAuthType

_Underlying type:_ `string`

GitAuthType defines the Git auth type.

_Appears in:_

- [Repository](#repository)

#### Repository

Repository represents VCS repo, currently we're handling Git only.

_Appears in:_

- [TestContent](#testcontent)

| Field                                      | Description                                                                              |
| ------------------------------------------ | ---------------------------------------------------------------------------------------- |
| `type` _string_                            | VCS repository type                                                                      |
| `uri` _string_                             | URI of content file or Git directory                                                     |
| `branch` _string_                          | branch/tag name for checkout                                                             |
| `commit` _string_                          | commit id (sha) for checkout                                                             |
| `path` _string_                            | If needed, we can checkout a particular path (dir or file) in the case of BIG/mono repositories. |
| `usernameSecret` _[SecretRef](#secretref)_ |                                                                                          |
| `tokenSecret` _[SecretRef](#secretref)_    |                                                                                          |
| `certificateSecret` _string_               | Git auth certificate secret for private repositories                                     |
| `workingDir` _string_                      | If provided, we check out the whole repository and run the test from this directory.            |
| `authType` _[GitAuthType](#gitauthtype)_   | auth type for git requests                                                               |

#### RunningContext

The Running context for test or test suite execution.

_Appears in:_

- [ExecutionRequest](#executionrequest)

| Field                                              | Description                           |
| -------------------------------------------------- | ------------------------------------- |
| `type` _[RunningContextType](#runningcontexttype)_ | One of possible context types         |
| `context` _string_                                 | Context value depending from its type |

#### RunningContextType

_Underlying type:_ `string`

_Appears in:_

- [RunningContext](#runningcontext)

#### SecretRef

Testkube internal reference for secret storage in Kubernetes secrets.

_Appears in:_

- [Repository](#repository)

| Field                | Description                 |
| -------------------- | --------------------------- |
| `namespace` _string_ | object kubernetes namespace |
| `name` _string_      | object name                 |
| `key` _string_       | object key                  |

#### Test

Test is the Schema for the tests API.

_Appears in:_

- [TestList](#testlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v3`                                          |
| `kind` _string_                                                                                                    | `Test`                                                          |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSpec](#testspec)_                                                                                     |                                                                 |

#### TestContent

TestContent defines test content.

_Appears in:_

- [TestSpec](#testspec)

| Field                                        | Description                |
| -------------------------------------------- | -------------------------- |
| `type` _[TestContentType](#testcontenttype)_ | test type                  |
| `repository` _[Repository](#repository)_     | repository of test content |
| `data` _string_                              | test content body          |
| `uri` _string_                               | uri of test content        |

#### TestContentType

_Underlying type:_ `string`

_Appears in:_

- [TestContent](#testcontent)

#### TestList

TestList contains a list of the Test.

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v3`                                          |
| `kind` _string_                                                                                                | `TestList`                                                      |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Test](#test) array_                                                                                  |                                                                 |

#### TestSpec

TestSpec defines the the desired state of a Test.

_Appears in:_

- [Test](#test)

| Field                                                      | Description                                              |
| ---------------------------------------------------------- | -------------------------------------------------------- |
| `type` _string_                                            | test type                                                |
| `name` _string_                                            | test name                                                |
| `content` _[TestContent](#testcontent)_                    | test content object                                      |
| `source` _string_                                          | reference to test source resource                        |
| `schedule` _string_                                        | schedule in cron job format for scheduled test execution |
| `executionRequest` _[ExecutionRequest](#executionrequest)_ |                                                          |
| `uploads` _string array_                                   | files to be used from minio uploads                      |

#### Variable

_Appears in:_

- [ExecutionRequest](#executionrequest)
- [TestSpec](#testspec)
- [TestSuiteExecutionRequest](#testsuiteexecutionrequest)
- [TestSuiteSpec](#testsuitespec)

| Field                                                                                                                   | Description                |
| ----------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| `type` _string_                                                                                                         | variable type              |
| `name` _string_                                                                                                         | variable name              |
| `value` _string_                                                                                                        | variable string value      |
| `valueFrom` _[EnvVarSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvarsource-v1-core)_ | or load it from var source |
