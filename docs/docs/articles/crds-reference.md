# CRDs Reference

CRDs (Custom Resource Definitions) reference. Read more Testkube's CRDs in [Testkube Custom Resources](./crds.md) section.

## Packages

- [executor.testkube.io/v1](#executortestkubeiov1)
- [tests.testkube.io/v1](#teststestkubeiov1)
- [tests.testkube.io/v2](#teststestkubeiov2)
- [tests.testkube.io/v3](#teststestkubeiov3)

## executor.testkube.io/v1

Package v1 contains API Schema definitions for the executor v1 API group

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

Executor is the Schema for the executors API

_Appears in:_

- [ExecutorList](#executorlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                    | `Executor`                                                      |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ExecutorSpec](#executorspec)_                                                                             |                                                                 |

#### ExecutorList

ExecutorList contains a list of Executor

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                | `ExecutorList`                                                  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Executor](#executor) array_                                                                          |                                                                 |

#### ExecutorMeta

Executor meta data

_Appears in:_

- [ExecutorSpec](#executorspec)

| Field                                            | Description           |
| ------------------------------------------------ | --------------------- |
| `iconURI` _string_                               | URI for executor icon |
| `docsURI` _string_                               | URI for executor docs |
| `tooltips` _object (keys:string, values:string)_ | executor tooltips     |

#### ExecutorSpec

ExecutorSpec defines the desired state of Executor

_Appears in:_

- [Executor](#executor)

| Field                                                                                                                                                | Description                                                                                                                                            |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `types` _string array_                                                                                                                               | Types defines what types can be handled by executor e.g. "postman/collection", ":curl/command" etc                                                     |
| `executor_type` _[ExecutorType](#executortype)_                                                                                                      | ExecutorType one of "rest" for rest openapi based executors or "job" which will be default runners for testkube or "container" for container executors |
| `uri` _string_                                                                                                                                       | URI for rest based executors                                                                                                                           |
| `image` _string_                                                                                                                                     | Image for kube-job                                                                                                                                     |
| `args` _string array_                                                                                                                                | executor binary arguments                                                                                                                              |
| `command` _string array_                                                                                                                             | executor default binary command                                                                                                                        |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ | container executor default image pull secrets                                                                                                          |
| `features` _[Feature](#feature) array_                                                                                                               | Features list of possible features which executor handles                                                                                              |
| `content_types` _[ScriptContentType](#scriptcontenttype) array_                                                                                      | ContentTypes list of handled content types                                                                                                             |
| `job_template` _string_                                                                                                                              | Job template to launch executor                                                                                                                        |
| `meta` _[ExecutorMeta](#executormeta)_                                                                                                               | Meta data about executor                                                                                                                               |

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

Webhook is the Schema for the webhooks API

_Appears in:_

- [WebhookList](#webhooklist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                    | `Webhook`                                                       |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[WebhookSpec](#webhookspec)_                                                                               |                                                                 |

#### WebhookList

WebhookList contains a list of Webhook

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `executor.testkube.io/v1`                                       |
| `kind` _string_                                                                                                | `WebhookList`                                                   |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Webhook](#webhook) array_                                                                            |                                                                 |

#### WebhookSpec

WebhookSpec defines the desired state of Webhook

_Appears in:_

- [Webhook](#webhook)

| Field                                           | Description                                                        |
| ----------------------------------------------- | ------------------------------------------------------------------ |
| `uri` _string_                                  | Uri is address where webhook should be made                        |
| `events` _[EventType](#eventtype) array_        | Events declare list if events on which webhook should be called    |
| `selector` _string_                             | Labels to filter for tests and test suites                         |
| `payloadObjectField` _string_                   | will load the generated payload for notification inside the object |
| `payloadTemplate` _string_                      | golang based template for notification payload                     |
| `headers` _object (keys:string, values:string)_ | webhook headers                                                    |

## tests.testkube.io/v1

Package v1 contains API Schema definitions for the testkube v1 API group

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

GitAuthType defines git auth type

_Appears in:_

- [Repository](#repository)

#### Repository

Repository represents VCS repo, currently we're handling Git only

_Appears in:_

- [TestSourceSpec](#testsourcespec)

| Field                                      | Description                                                                              |
| ------------------------------------------ | ---------------------------------------------------------------------------------------- |
| `type` _string_                            | VCS repository type                                                                      |
| `uri` _string_                             | uri of content file or git directory                                                     |
| `branch` _string_                          | branch/tag name for checkout                                                             |
| `commit` _string_                          | commit id (sha) for checkout                                                             |
| `path` _string_                            | if needed we can checkout particular path (dir or file) in case of BIG/mono repositories |
| `usernameSecret` _[SecretRef](#secretref)_ |                                                                                          |
| `tokenSecret` _[SecretRef](#secretref)_    |                                                                                          |
| `certificateSecret` _string_               | git auth certificate secret for private repositories                                     |
| `workingDir` _string_                      | if provided we checkout the whole repository and run test from this directory            |
| `authType` _[GitAuthType](#gitauthtype)_   | auth type for git requests                                                               |

#### Script

Script is the Schema for the scripts API

_Appears in:_

- [ScriptList](#scriptlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `Script`                                                        |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ScriptSpec](#scriptspec)_                                                                                 |                                                                 |

#### ScriptList

ScriptList contains a list of Script

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `ScriptList`                                                    |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Script](#script) array_                                                                              |                                                                 |

#### ScriptSpec

ScriptSpec defines the desired state of Script

_Appears in:_

- [Script](#script)

| Field                                          | Description                                                                                                                                                           |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `type` _string_                                | script type                                                                                                                                                           |
| `name` _string_                                | script execution custom name                                                                                                                                          |
| `params` _object (keys:string, values:string)_ | execution params passed to executor                                                                                                                                   |
| `content` _string_                             | script content as string (content depends from executor)                                                                                                              |
| `input-type` _string_                          | script content type can be: - direct content - created from file, - git repo directory checkout in case when test is some kind of project or have more than one file, |
| `repository` _[Repository](#repository)_       | repository details if exists                                                                                                                                          |
| `tags` _string array_                          |                                                                                                                                                                       |

#### SecretRef

Testkube internal reference for secret storage in Kubernetes secrets

_Appears in:_

- [Repository](#repository)

| Field                | Description                 |
| -------------------- | --------------------------- |
| `namespace` _string_ | object kubernetes namespace |
| `name` _string_      | object name                 |
| `key` _string_       | object key                  |

#### Test

Test is the Schema for the tests API

_Appears in:_

- [TestList](#testlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `Test`                                                          |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSpec](#testspec)_                                                                                     |                                                                 |

#### TestList

TestList contains a list of Test

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestList`                                                      |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Test](#test) array_                                                                                  |                                                                 |

#### TestSource

TestSource is the Schema for the testsources API

_Appears in:_

- [TestSourceList](#testsourcelist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `TestSource`                                                    |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSourceSpec](#testsourcespec)_                                                                         |                                                                 |

#### TestSourceList

TestSourceList contains a list of TestSource

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestSourceList`                                                |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestSource](#testsource) array_                                                                      |                                                                 |

#### TestSourceSpec

TestSourceSpec defines the desired state of TestSource

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

TestSpec defines the desired state of Test

_Appears in:_

- [Test](#test)

| Field                                          | Description                                                             |
| ---------------------------------------------- | ----------------------------------------------------------------------- |
| `before` _[TestStepSpec](#teststepspec) array_ | Before steps is list of scripts which will be sequentially orchestrated |
| `steps` _[TestStepSpec](#teststepspec) array_  | Steps is list of scripts which will be sequentially orchestrated        |
| `after` _[TestStepSpec](#teststepspec) array_  | After steps is list of scripts which will be sequentially orchestrated  |
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

TestStepSpec will of particular type will have config for possible step types

_Appears in:_

- [TestSpec](#testspec)

| Field                                           | Description |
| ----------------------------------------------- | ----------- |
| `type` _string_                                 |             |
| `execute` _[TestStepExecute](#teststepexecute)_ |             |
| `delay` _[TestStepDelay](#teststepdelay)_       |             |

#### TestSuite

TestSuite is the Schema for the testsuites API

_Appears in:_

- [TestSuiteList](#testsuitelist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                    | `TestSuite`                                                     |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSuiteSpec](#testsuitespec)_                                                                           |                                                                 |

#### TestSuiteList

TestSuiteList contains a list of TestSuite

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestSuiteList`                                                 |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestSuite](#testsuite) array_                                                                        |                                                                 |

#### TestSuiteSpec

TestSuiteSpec defines the desired state of TestSuite

_Appears in:_

- [TestSuite](#testsuite)

| Field                                                            | Description                                                           |
| ---------------------------------------------------------------- | --------------------------------------------------------------------- |
| `before` _[TestSuiteStepSpec](#testsuitestepspec) array_         | Before steps is list of tests which will be sequentially orchestrated |
| `steps` _[TestSuiteStepSpec](#testsuitestepspec) array_          | Steps is list of tests which will be sequentially orchestrated        |
| `after` _[TestSuiteStepSpec](#testsuitestepspec) array_          | After steps is list of tests which will be sequentially orchestrated  |
| `repeats` _integer_                                              |                                                                       |
| `description` _string_                                           |                                                                       |
| `schedule` _string_                                              | schedule in cron job format for scheduled test execution              |
| `params` _object (keys:string, values:string)_                   | DEPRECATED execution params passed to executor                        |
| `variables` _object (keys:string, values:[Variable](#variable))_ | Variables are new params with secrets attached                        |

#### TestSuiteStepDelay

TestSuiteStepDelay contains step delay parameters

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                | Description    |
| -------------------- | -------------- |
| `duration` _integer_ | Duration in ms |

#### TestSuiteStepExecute

TestSuiteStepExecute defines step to be executed

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                     | Description |
| ------------------------- | ----------- |
| `namespace` _string_      |             |
| `name` _string_           |             |
| `stopOnFailure` _boolean_ |             |

#### TestSuiteStepSpec

TestSuiteStepSpec will of particular type will have config for possible step types

_Appears in:_

- [TestSuiteSpec](#testsuitespec)

| Field                                                     | Description |
| --------------------------------------------------------- | ----------- |
| `type` _string_                                           |             |
| `execute` _[TestSuiteStepExecute](#testsuitestepexecute)_ |             |
| `delay` _[TestSuiteStepDelay](#testsuitestepdelay)_       |             |

#### TestTrigger

TestTrigger is the Schema for the testtriggers API

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

TestTriggerAction defines action for test triggers

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerCondition

TestTriggerCondition is used for definition of the condition for test triggers

_Appears in:_

- [TestTriggerConditionSpec](#testtriggerconditionspec)

| Field             | Description                                                                         |
| ----------------- | ----------------------------------------------------------------------------------- |
| `type` _string_   | test trigger condition                                                              |
| `reason` _string_ | test trigger condition reason                                                       |
| `ttl` _integer_   | duration in seconds in the past from current time when the condition is still valid |

#### TestTriggerConditionSpec

TestTriggerConditionSpec defines the condition specification for TestTrigger

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

| Field                                                              | Description                                                                  |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| `conditions` _[TestTriggerCondition](#testtriggercondition) array_ | list of test trigger conditions                                              |
| `timeout` _integer_                                                | duration in seconds the test trigger waits for conditions, until its stopped |

#### TestTriggerEvent

_Underlying type:_ `string`

TestTriggerEvent defines event for test triggers

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerExecution

_Underlying type:_ `string`

TestTriggerExecution defines execution for test triggers

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerList

TestTriggerList contains a list of TestTrigger

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v1`                                          |
| `kind` _string_                                                                                                | `TestTriggerList`                                               |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestTrigger](#testtrigger) array_                                                                    |                                                                 |

#### TestTriggerResource

_Underlying type:_ `string`

TestTriggerResource defines resource for test triggers

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

#### TestTriggerSelector

TestTriggerSelector is used for selecting Kubernetes Objects

_Appears in:_

- [TestTriggerSpec](#testtriggerspec)

| Field                                                                                                                         | Description                                                                                    |
| ----------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `name` _string_                                                                                                               | Name selector is used to identify a Kubernetes Object based on the metadata name               |
| `namespace` _string_                                                                                                          | Namespace of the Kubernetes object                                                             |
| `labelSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#labelselector-v1-meta)_ | LabelSelector is used to identify a group of Kubernetes Objects based on their metadata labels |

#### TestTriggerSpec

TestTriggerSpec defines the desired state of TestTrigger

_Appears in:_

- [TestTrigger](#testtrigger)

| Field                                                                                                       | Description                                                                                               |
| ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `resource` _[TestTriggerResource](#testtriggerresource)_                                                    | For which Resource do we monitor Event which triggers an Action on certain conditions                     |
| `resourceSelector` _[TestTriggerSelector](#testtriggerselector)_                                            | ResourceSelector identifies which Kubernetes Objects should be watched                                    |
| `event` _[TestTriggerEvent](#testtriggerevent)_                                                             | On which Event for a Resource should an Action be triggered                                               |
| `conditionSpec` _[TestTriggerConditionSpec](#testtriggerconditionspec)_                                     | What resource conditions should be matched                                                                |
| `action` _[TestTriggerAction](#testtriggeraction)_                                                          | Action represents what needs to be executed for selected Execution                                        |
| `execution` _[TestTriggerExecution](#testtriggerexecution)_                                                 | Execution identifies for which test execution should an Action be executed                                |
| `testSelector` _[TestTriggerSelector](#testtriggerselector)_                                                | TestSelector identifies on which Testkube Kubernetes Objects an Action should be taken                    |
| `delay` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#duration-v1-meta)_ | Delay is a duration string which specifies how long should the test be delayed after a trigger is matched |

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

## tests.testkube.io/v2

Package v2 contains API Schema definitions for the testkube v2 API group

### Resource Types

- [Script](#script)
- [ScriptList](#scriptlist)
- [Test](#test)
- [TestList](#testlist)
- [TestSuite](#testsuite)
- [TestSuiteList](#testsuitelist)

#### Repository

Repository represents VCS repo, currently we're handling Git only

_Appears in:_

- [TestContent](#testcontent)

| Field               | Description                                                                              |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `type` _string_     | VCS repository type                                                                      |
| `uri` _string_      | uri of content file or git directory                                                     |
| `branch` _string_   | branch/tag name for checkout                                                             |
| `commit` _string_   | commit id (sha) for checkout                                                             |
| `path` _string_     | if needed we can checkout particular path (dir or file) in case of BIG/mono repositories |
| `username` _string_ | git auth username for private repositories                                               |
| `token` _string_    | git auth token for private repositories                                                  |

#### RunningContext

running context for test or test suite execution

_Appears in:_

- [TestSuiteExecutionRequest](#testsuiteexecutionrequest)

| Field                                              | Description                           |
| -------------------------------------------------- | ------------------------------------- |
| `type` _[RunningContextType](#runningcontexttype)_ | One of possible context types         |
| `context` _string_                                 | Context value depending from its type |

#### RunningContextType

_Underlying type:_ `string`

_Appears in:_

- [RunningContext](#runningcontext)

#### Script

Script is the Schema for the scripts API

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
| `uri` _string_                           | uri of script content        |

#### ScriptList

ScriptList contains a list of Script

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                | `ScriptList`                                                    |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Script](#script) array_                                                                              |                                                                 |

#### ScriptSpec

ScriptSpec defines the desired state of Script

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

Test is the Schema for the tests API

_Appears in:_

- [TestList](#testlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                    | `Test`                                                          |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSpec](#testspec)_                                                                                     |                                                                 |

#### TestContent

TestContent defines test content

_Appears in:_

- [TestSpec](#testspec)

| Field                                    | Description                |
| ---------------------------------------- | -------------------------- |
| `type` _string_                          | test type                  |
| `repository` _[Repository](#repository)_ | repository of test content |
| `data` _string_                          | test content body          |
| `uri` _string_                           | uri of test content        |

#### TestList

TestList contains a list of Test

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                | `TestList`                                                      |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Test](#test) array_                                                                                  |                                                                 |

#### TestSpec

TestSpec defines the desired state of Test

_Appears in:_

- [Test](#test)

| Field                                                            | Description                                              |
| ---------------------------------------------------------------- | -------------------------------------------------------- |
| `type` _string_                                                  | test type                                                |
| `name` _string_                                                  | test execution custom name                               |
| `params` _object (keys:string, values:string)_                   | DEPRECATED execution params passed to executor           |
| `variables` _object (keys:string, values:[Variable](#variable))_ | Variables are new params with secrets attached           |
| `content` _[TestContent](#testcontent)_                          | test content object                                      |
| `schedule` _string_                                              | schedule in cron job format for scheduled test execution |
| `executorArgs` _string array_                                    | additional executor binary arguments                     |

#### TestSuite

TestSuite is the Schema for the testsuites API

_Appears in:_

- [TestSuiteList](#testsuitelist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                    | `TestSuite`                                                     |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSuiteSpec](#testsuitespec)_                                                                           |                                                                 |

#### TestSuiteExecutionCore

test suite execution core

_Appears in:_

- [TestSuiteStatus](#testsuitestatus)

| Field                                                                                                   | Description                     |
| ------------------------------------------------------------------------------------------------------- | ------------------------------- |
| `id` _string_                                                                                           | execution id                    |
| `startTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | test suite execution start time |
| `endTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_   | test suite execution end time   |

#### TestSuiteExecutionRequest

test suite execution request body

_Appears in:_

- [TestSuiteSpec](#testsuitespec)

| Field                                                            | Description                                           |
| ---------------------------------------------------------------- | ----------------------------------------------------- |
| `name` _string_                                                  | test execution custom name                            |
| `namespace` _string_                                             | test kubernetes namespace (\"testkube\" when not set) |
| `variables` _object (keys:string, values:[Variable](#variable))_ |                                                       |
| `secretUUID` _string_                                            | secret uuid                                           |
| `labels` _object (keys:string, values:string)_                   | test suite labels                                     |
| `executionLabels` _object (keys:string, values:string)_          | execution labels                                      |
| `sync` _boolean_                                                 | whether to start execution sync or async              |
| `httpProxy` _string_                                             | http proxy for executor containers                    |
| `httpsProxy` _string_                                            | https proxy for executor containers                   |
| `timeout` _integer_                                              | timeout for test suite execution                      |
| `runningContext` _[RunningContext](#runningcontext)_             |                                                       |
| `cronJobTemplate` _string_                                       | cron job template extensions                          |

#### TestSuiteList

TestSuiteList contains a list of TestSuite

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v2`                                          |
| `kind` _string_                                                                                                | `TestSuiteList`                                                 |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[TestSuite](#testsuite) array_                                                                        |                                                                 |

#### TestSuiteSpec

TestSuiteSpec defines the desired state of TestSuite

_Appears in:_

- [TestSuite](#testsuite)

| Field                                                                        | Description                                                           |
| ---------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| `before` _[TestSuiteStepSpec](#testsuitestepspec) array_                     | Before steps is list of tests which will be sequentially orchestrated |
| `steps` _[TestSuiteStepSpec](#testsuitestepspec) array_                      | Steps is list of tests which will be sequentially orchestrated        |
| `after` _[TestSuiteStepSpec](#testsuitestepspec) array_                      | After steps is list of tests which will be sequentially orchestrated  |
| `repeats` _integer_                                                          |                                                                       |
| `description` _string_                                                       |                                                                       |
| `schedule` _string_                                                          | schedule in cron job format for scheduled test execution              |
| `executionRequest` _[TestSuiteExecutionRequest](#testsuiteexecutionrequest)_ |                                                                       |

#### TestSuiteStepDelay

TestSuiteStepDelay contains step delay parameters

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                | Description    |
| -------------------- | -------------- |
| `duration` _integer_ | Duration in ms |

#### TestSuiteStepExecute

TestSuiteStepExecute defines step to be executed

_Appears in:_

- [TestSuiteStepSpec](#testsuitestepspec)

| Field                     | Description |
| ------------------------- | ----------- |
| `namespace` _string_      |             |
| `name` _string_           |             |
| `stopOnFailure` _boolean_ |             |

#### TestSuiteStepSpec

TestSuiteStepSpec for particular type will have config for possible step types

_Appears in:_

- [TestSuiteSpec](#testsuitespec)

| Field                                                     | Description |
| --------------------------------------------------------- | ----------- |
| `type` _[TestSuiteStepType](#testsuitesteptype)_          |             |
| `execute` _[TestSuiteStepExecute](#testsuitestepexecute)_ |             |
| `delay` _[TestSuiteStepDelay](#testsuitestepdelay)_       |             |

#### TestSuiteStepType

_Underlying type:_ `string`

TestSuiteStepType defines different type of test suite steps

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

Package v3 contains API Schema definitions for the tests v3 API group

### Resource Types

- [Test](#test)
- [TestList](#testlist)

#### ArgsModeType

_Underlying type:_ `string`

ArgsModeType defines args mode type

_Appears in:_

- [ExecutionRequest](#executionrequest)

#### ArtifactRequest

artifact request body with test artifacts

_Appears in:_

- [ExecutionRequest](#executionrequest)

| Field                       | Description                                        |
| --------------------------- | -------------------------------------------------- |
| `storageClassName` _string_ | artifact storage class name for container executor |
| `volumeMountPath` _string_  | artifact volume mount path for container executor  |
| `dirs` _string array_       | artifact directories for scraping                  |

#### EnvReference

Reference to env resource

_Appears in:_

- [ExecutionRequest](#executionrequest)

| Field                                                                                                                                   | Description                                     |
| --------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| `reference` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core)_ |                                                 |
| `mount` _boolean_                                                                                                                       | whether we shoud mount resource                 |
| `mountPath` _string_                                                                                                                    | where we shoud mount resource                   |
| `mapToVariables` _boolean_                                                                                                              | whether we shoud map to variables from resource |

#### ExecutionCore

test execution core

_Appears in:_

- [TestStatus](#teststatus)

| Field                                                                                                   | Description      |
| ------------------------------------------------------------------------------------------------------- | ---------------- |
| `id` _string_                                                                                           | execution id     |
| `number` _integer_                                                                                      | execution number |
| `startTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | test start time  |
| `endTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_   | test end time    |

#### ExecutionRequest

test execution request body

_Appears in:_

- [TestSpec](#testspec)

| Field                                                                                                                                                | Description                                                                                                                                                                                                  |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `name` _string_                                                                                                                                      | test execution custom name                                                                                                                                                                                   |
| `testSuiteName` _string_                                                                                                                             | unique test suite name (CRD Test suite name), if it's run as a part of test suite                                                                                                                            |
| `number` _integer_                                                                                                                                   | test execution number                                                                                                                                                                                        |
| `executionLabels` _object (keys:string, values:string)_                                                                                              | test execution labels                                                                                                                                                                                        |
| `namespace` _string_                                                                                                                                 | test kubernetes namespace (\"testkube\" when not set)                                                                                                                                                        |
| `variablesFile` _string_                                                                                                                             | variables file content - need to be in format for particular executor (e.g. postman envs file)                                                                                                               |
| `isVariablesFileUploaded` _boolean_                                                                                                                  |                                                                                                                                                                                                              |
| `variables` _object (keys:string, values:[Variable](#variable))_                                                                                     |                                                                                                                                                                                                              |
| `testSecretUUID` _string_                                                                                                                            | test secret uuid                                                                                                                                                                                             |
| `testSuiteSecretUUID` _string_                                                                                                                       | test suite secret uuid, if it's run as a part of test suite                                                                                                                                                  |
| `args` _string array_                                                                                                                                | additional executor binary arguments                                                                                                                                                                         |
| `argsMode` _[ArgsModeType](#argsmodetype)_                                                                                                           | usage mode for arguments                                                                                                                                                                                     |
| `command` _string array_                                                                                                                             | executor binary command                                                                                                                                                                                      |
| `image` _string_                                                                                                                                     | container executor image                                                                                                                                                                                     |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ | container executor image pull secrets                                                                                                                                                                        |
| `envs` _object (keys:string, values:string)_                                                                                                         | Environment variables passed to executor. Deprecated: use Basic Variables instead                                                                                                                            |
| `secretEnvs` _object (keys:string, values:string)_                                                                                                   | Execution variables passed to executor from secrets. Deprecated: use Secret Variables instead                                                                                                                |
| `sync` _boolean_                                                                                                                                     | whether to start execution sync or async                                                                                                                                                                     |
| `httpProxy` _string_                                                                                                                                 | http proxy for executor containers                                                                                                                                                                           |
| `httpsProxy` _string_                                                                                                                                | https proxy for executor containers                                                                                                                                                                          |
| `negativeTest` _boolean_                                                                                                                             | negative test will fail the execution if it is a success and it will succeed if it is a failure                                                                                                              |
| `activeDeadlineSeconds` _integer_                                                                                                                    | Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer. |
| `artifactRequest` _[ArtifactRequest](#artifactrequest)_                                                                                              |                                                                                                                                                                                                              |
| `jobTemplate` _string_                                                                                                                               | job template extensions                                                                                                                                                                                      |
| `cronJobTemplate` _string_                                                                                                                           | cron job template extensions                                                                                                                                                                                 |
| `preRunScript` _string_                                                                                                                              | script to run before test execution                                                                                                                                                                          |
| `scraperTemplate` _string_                                                                                                                           | scraper template extensions                                                                                                                                                                                  |
| `envConfigMaps` _[EnvReference](#envreference) array_                                                                                                | config map references                                                                                                                                                                                        |
| `envSecrets` _[EnvReference](#envreference) array_                                                                                                   | secret references                                                                                                                                                                                            |
| `runningContext` _[RunningContext](#runningcontext)_                                                                                                 |                                                                                                                                                                                                              |

#### GitAuthType

_Underlying type:_ `string`

GitAuthType defines git auth type

_Appears in:_

- [Repository](#repository)

#### Repository

Repository represents VCS repo, currently we're handling Git only

_Appears in:_

- [TestContent](#testcontent)

| Field                                      | Description                                                                              |
| ------------------------------------------ | ---------------------------------------------------------------------------------------- |
| `type` _string_                            | VCS repository type                                                                      |
| `uri` _string_                             | uri of content file or git directory                                                     |
| `branch` _string_                          | branch/tag name for checkout                                                             |
| `commit` _string_                          | commit id (sha) for checkout                                                             |
| `path` _string_                            | if needed we can checkout particular path (dir or file) in case of BIG/mono repositories |
| `usernameSecret` _[SecretRef](#secretref)_ |                                                                                          |
| `tokenSecret` _[SecretRef](#secretref)_    |                                                                                          |
| `certificateSecret` _string_               | git auth certificate secret for private repositories                                     |
| `workingDir` _string_                      | if provided we checkout the whole repository and run test from this directory            |
| `authType` _[GitAuthType](#gitauthtype)_   | auth type for git requests                                                               |

#### RunningContext

running context for test or test suite execution

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

Testkube internal reference for secret storage in Kubernetes secrets

_Appears in:_

- [Repository](#repository)

| Field                | Description                 |
| -------------------- | --------------------------- |
| `namespace` _string_ | object kubernetes namespace |
| `name` _string_      | object name                 |
| `key` _string_       | object key                  |

#### Test

Test is the Schema for the tests API

_Appears in:_

- [TestList](#testlist)

| Field                                                                                                              | Description                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                              | `tests.testkube.io/v3`                                          |
| `kind` _string_                                                                                                    | `Test`                                                          |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[TestSpec](#testspec)_                                                                                     |                                                                 |

#### TestContent

TestContent defines test content

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

TestList contains a list of Test

| Field                                                                                                          | Description                                                     |
| -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `apiVersion` _string_                                                                                          | `tests.testkube.io/v3`                                          |
| `kind` _string_                                                                                                | `TestList`                                                      |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Test](#test) array_                                                                                  |                                                                 |

#### TestSpec

TestSpec defines the desired state of Test

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
