/**
 * Testkube API
 * 1.0.0
 * DO NOT MODIFY - This file has been generated using oazapfts.
 * See https://www.npmjs.com/package/oazapfts
 */
import * as Oazapfts from "oazapfts/lib/runtime";
import * as QS from "oazapfts/lib/runtime/query";
export const defaults: Oazapfts.RequestOpts = {
    baseUrl: "/",
};
const oazapfts = Oazapfts.runtime(defaults);
export const servers = {};
export type TestTriggerKeyMap = {
    resources: string[];
    actions: string[];
    executions: string[];
    events: {
        [key: string]: string[];
    };
};
export type TestTriggerResources = "pod" | "deployment" | "statefulset" | "daemonset" | "service" | "ingress" | "event" | "configmap";
export type TestTriggerSelector = {
    name?: string;
    "namespace"?: string;
    labelSelector?: {
        matchExpressions?: {
            key: string;
            operator: string;
            values?: string[];
        }[];
        matchLabels?: {
            [key: string]: string;
        };
    };
};
export type TestTriggerConditionStatuses = "True" | "False" | "Unknown";
export type TestTriggerCondition = {
    status: TestTriggerConditionStatuses;
    "type": string;
    reason?: string;
};
export type TestTriggerConditionSpec = {
    conditions?: TestTriggerCondition[];
    timeout?: number;
};
export type TestTriggerActions = "run";
export type TestTriggerExecutions = "test" | "testsuite";
export type TestTrigger = {
    name?: string;
    "namespace"?: string;
    labels?: {
        [key: string]: string;
    };
    resource: TestTriggerResources;
    resourceSelector: TestTriggerSelector;
    event: string;
    conditionSpec?: TestTriggerConditionSpec;
    action: TestTriggerActions;
    execution: TestTriggerExecutions;
    testSelector: TestTriggerSelector;
};
export type Problem = {
    "type"?: string;
    title?: string;
    status?: number;
    detail?: string;
    instance?: string;
};
export type ObjectRef = {
    "namespace"?: string;
    name: string;
};
export type TestTriggerUpsertRequest = TestTrigger & ObjectRef;
export type TestSuiteStepExecuteTest = ObjectRef;
export type TestSuiteStepDelay = {
    duration: number;
};
export type TestSuiteStep = {
    stopTestOnFailure: boolean;
    execute?: TestSuiteStepExecuteTest;
    delay?: TestSuiteStepDelay;
};
export type VariableType = "basic" | "secret";
export type SecretRef = {
    "namespace"?: string;
    name: string;
    key: string;
};
export type ConfigMapRef = {
    "namespace"?: string;
    name: string;
    key: string;
};
export type Variable = {
    name?: string;
    value?: string;
    "type"?: VariableType;
    secretRef?: SecretRef;
    configMapRef?: ConfigMapRef;
};
export type Variables = {
    [key: string]: Variable;
};
export type RepositoryParameters = {
    branch?: string;
    commit?: string;
    path?: string;
    workingDir?: string;
};
export type TestContentRequest = {
    repository?: RepositoryParameters;
};
export type RunningContext = {
    "type": "userCLI" | "userUI" | "testsuite" | "testtrigger" | "scheduler";
    context?: string;
};
export interface TestSuiteExecutionRequest {
    name?: string;
    "number"?: number;
    "namespace"?: string;
    variables?: Variables;
    labels?: {
        [key: string]: string;
    };
    executionLabels?: {
        [key: string]: string;
    };
    sync?: boolean;
    httpProxy?: string;
    httpsProxy?: string;
    timeout?: number;
    contentRequest?: TestContentRequest;
    runningContext?: RunningContext;
    cronJobTemplate?: string;
}
export type TestSuiteExecutionStatus = "queued" | "running" | "passed" | "failed" | "aborting" | "aborted" | "timeout";
export type TestSuiteExecutionCore = {
    id?: string;
    startTime?: string;
    endTime?: string;
    status?: TestSuiteExecutionStatus;
};
export type TestSuiteStatus = {
    latestExecution?: TestSuiteExecutionCore;
};
export type TestSuite = {
    name: string;
    "namespace"?: string;
    description?: string;
    before?: TestSuiteStep[];
    steps: TestSuiteStep[];
    after?: TestSuiteStep[];
    labels?: {
        [key: string]: string;
    };
    schedule?: string;
    repeats?: number;
    created?: string;
    executionRequest?: TestSuiteExecutionRequest;
    status: TestSuiteStatus;
};
export type TestSuiteUpsertRequest = TestSuite & ObjectRef;
export type TestSuiteUpdateRequest = (TestSuite & ObjectRef) | null;
export type ExecutionsMetricsExecutions = {
    executionId?: string;
    duration?: string;
    durationMs?: number;
    status?: string;
    name?: string;
    startTime?: string;
};
export type ExecutionsMetrics = {
    passFailRatio?: number;
    executionDurationP50?: string;
    executionDurationP50ms?: number;
    executionDurationP90?: string;
    executionDurationP90ms?: number;
    executionDurationP95?: string;
    executionDurationP95ms?: number;
    executionDurationP99?: string;
    executionDurationP99ms?: number;
    totalExecutions?: number;
    failedExecutions?: number;
    executions?: ExecutionsMetricsExecutions[];
};
export type Repository = {
    "type": "git";
    uri: string;
    branch?: string;
    commit?: string;
    path?: string;
    username?: string;
    token?: string;
    usernameSecret?: SecretRef;
    tokenSecret?: SecretRef;
    certificateSecret?: string;
    workingDir?: string;
    authType?: "basic" | "header";
};
export type TestContent = {
    "type"?: "string" | "file-uri" | "git-file" | "git-dir" | "git";
    repository?: Repository;
    data?: string;
    uri?: string;
};
export type LocalObjectReference = {
    name?: string;
};
export type ArtifactRequest = {
    storageClassName?: string;
    volumeMountPath?: string;
    dirs?: string[];
};
export type EnvReference = {
    reference: LocalObjectReference;
    mount?: boolean;
    mountPath?: string;
    mapToVariables?: boolean;
};
export interface ExecutionRequest {
    name?: string;
    testSuiteName?: string;
    "number"?: number;
    executionLabels?: {
        [key: string]: string;
    };
    "namespace"?: string;
    isVariablesFileUploaded?: boolean;
    variablesFile?: string;
    variables?: Variables;
    command?: string[];
    args?: string[];
    args_mode?: "append" | "override";
    image?: string;
    imagePullSecrets?: LocalObjectReference[];
    envs?: {
        [key: string]: string;
    };
    secretEnvs?: {
        [key: string]: string;
    };
    sync?: boolean;
    httpProxy?: string;
    httpsProxy?: string;
    negativeTest?: boolean;
    isNegativeTestChangedOnRun?: boolean;
    activeDeadlineSeconds?: number;
    uploads?: string[];
    bucketName?: string;
    artifactRequest?: ArtifactRequest;
    jobTemplate?: string;
    cronJobTemplate?: string;
    contentRequest?: TestContentRequest;
    preRunScript?: string;
    scraperTemplate?: string;
    envConfigMaps?: EnvReference[];
    envSecrets?: EnvReference[];
    runningContext?: RunningContext;
}
export type ExecutionStatus = "queued" | "running" | "passed" | "failed" | "aborted" | "timeout";
export type ExecutionCore = {
    id?: string;
    "number"?: number;
    startTime?: string;
    endTime?: string;
    status?: ExecutionStatus;
};
export type TestStatus = {
    latestExecution?: ExecutionCore;
};
export type Test = {
    name?: string;
    "namespace"?: string;
    "type"?: string;
    content?: TestContent;
    source?: string;
    created?: string;
    labels?: {
        [key: string]: string;
    };
    schedule?: string;
    uploads?: string[];
    executionRequest?: ExecutionRequest;
    status?: TestStatus;
};
export type TestSuiteStepType = "executeTest" | "delay";
export type TestSuiteStepExecutionSummary = {
    id: string;
    name: string;
    testName?: string;
    status: ExecutionStatus;
    "type"?: TestSuiteStepType;
};
export type TestSuiteExecutionSummary = {
    id: string;
    name: string;
    testSuiteName: string;
    status: TestSuiteExecutionStatus;
    startTime?: string;
    endTime?: string;
    duration?: string;
    durationMs?: number;
    execution?: TestSuiteStepExecutionSummary[];
    labels?: {
        [key: string]: string;
    };
};
export type TestSuiteWithExecutionSummary = {
    testSuite: TestSuite;
    latestExecution?: TestSuiteExecutionSummary;
};
export type AssertionResult = {
    name?: string;
    status?: "passed" | "failed";
    errorMessage?: string | null;
};
export type ExecutionStepResult = {
    name: string;
    duration?: string;
    status: "passed" | "failed";
    assertionResults?: AssertionResult[];
};
export type ExecutionResult = {
    status: ExecutionStatus;
    output?: string;
    outputType?: "text/plain" | "application/junit+xml" | "application/json";
    errorMessage?: string;
    steps?: ExecutionStepResult[];
    reports?: {
        junit?: string;
    };
};
export interface Execution {
    id?: string;
    testName?: string;
    testSuiteName?: string;
    testNamespace?: string;
    testType?: string;
    name?: string;
    "number"?: number;
    envs?: {
        [key: string]: string;
    };
    command?: string[];
    args?: string[];
    args_mode?: "append" | "override";
    variables?: Variables;
    isVariablesFileUploaded?: boolean;
    variablesFile?: string;
    content?: TestContent;
    startTime?: string;
    endTime?: string;
    duration?: string;
    durationMs?: number;
    executionResult?: ExecutionResult;
    labels?: {
        [key: string]: string;
    };
    uploads?: string[];
    bucketName?: string;
    artifactRequest?: ArtifactRequest;
    preRunScript?: string;
    runningContext?: RunningContext;
}
export type TestSuiteStepExecutionResult = {
    step?: TestSuiteStep;
    test?: ObjectRef;
    execution?: Execution;
};
export interface TestSuiteExecution {
    id: string;
    name: string;
    testSuite?: ObjectRef;
    status?: TestSuiteExecutionStatus;
    envs?: {
        [key: string]: string;
    };
    variables?: Variables;
    startTime?: string;
    endTime?: string;
    duration?: string;
    durationMs?: number;
    stepResults?: TestSuiteStepExecutionResult[];
    labels?: {
        [key: string]: string;
    };
    runningContext?: RunningContext;
}
export type TestSuiteWithExecution = {
    testSuite: TestSuite;
    latestExecution?: TestSuiteExecution;
};
export type ExecutionsTotals = {
    results: number;
    passed: number;
    failed: number;
    queued: number;
    running: number;
};
export type TestSuiteExecutionsResult = {
    totals: ExecutionsTotals;
    filtered?: ExecutionsTotals;
    results: TestSuiteExecutionSummary[];
};
export interface TestSuiteExecutionRead extends TestSuiteExecution {
    secretUUID?: string;
}
export type Artifact = {
    name?: string;
    size?: number;
    executionName?: string;
};
export type ExecutionSummary = {
    id: string;
    name: string;
    "number"?: number;
    testName: string;
    testNamespace?: string;
    testType: string;
    status: ExecutionStatus;
    startTime?: string;
    endTime?: string;
    duration?: string;
    durationMs?: number;
    labels?: {
        [key: string]: string;
    };
    runningContext?: RunningContext;
};
export type ExecutionsResult = {
    totals: ExecutionsTotals;
    filtered?: ExecutionsTotals;
    results: ExecutionSummary[];
};
export interface ExecutionRead extends Execution {
    testSecretUUID?: string;
    testSuiteSecretUUID?: string;
}
export type ExecutorOutput = {
    "type": "error" | "log" | "event" | "result";
    content?: string;
    result?: ExecutionResult;
    time?: string;
};
export type TestUpsertRequest = Test;
export type TestUpdateRequest = (Test) | null;
export type TestWithExecutionSummary = {
    test: Test;
    latestExecution?: ExecutionSummary;
};
export type TestWithExecution = {
    test: Test;
    latestExecution?: Execution;
};
export type ExecutorMeta = {
    iconURI?: string;
    docsURI?: string;
    tooltips?: {
        [key: string]: string;
    };
};
export type Executor = {
    executorType?: string;
    image?: string;
    imagePullSecrets?: LocalObjectReference[];
    command?: string[];
    args?: string[];
    types?: string[];
    uri?: string;
    contentTypes?: string[];
    jobTemplate?: string;
    labels?: {
        [key: string]: string;
    };
    features?: ("artifacts" | "junit-report")[];
    meta?: ExecutorMeta;
};
export type ExecutorUpsertRequest = Executor & ObjectRef;
export type ExecutorDetails = {
    name?: string;
    executor?: Executor;
    executions?: ExecutionsResult;
};
export type ExecutorUpdateRequest = (Executor & ObjectRef) | null;
export type EventType = "start-test" | "end-test-success" | "end-test-failed" | "end-test-aborted" | "end-test-timeout" | "start-testsuite" | "end-testsuite-success" | "end-testsuite-failed" | "end-testsuite-aborted" | "end-testsuite-timeout" | "created" | "updated" | "deleted";
export type Webhook = {
    name?: string;
    "namespace"?: string;
    uri: string;
    events: EventType[];
    selector?: string;
    payloadObjectField?: string;
    labels?: {
        [key: string]: string;
    };
};
export type WebhookCreateRequest = Webhook;
export type Config = {
    id: string;
    clusterId: string;
    enableTelemetry: boolean;
};
export type DebugInfo = {
    clientVersion?: string;
    serverVersion?: string;
    clusterVersion?: string;
    apiLogs?: string[];
    operatorLogs?: string[];
    executionLogs?: {
        [key: string]: string[];
    };
};
export type TestSource = TestContent & {
    name?: string;
    "namespace"?: string;
    labels?: {
        [key: string]: string;
    };
};
export type TestSourceUpsertRequest = TestContent & {
    name?: string;
    "namespace"?: string;
    labels?: {
        [key: string]: string;
    };
};
export type TestSourceBatchRequest = {
    batch: TestSourceUpsertRequest[];
};
export type TestSourceBatchResult = {
    created?: string[];
    updated?: string[];
    deleted?: string[];
};
export type TestSourceUpdateRequest = (TestContent & {
    name?: string;
    "namespace"?: string;
    labels?: {
        [key: string]: string;
    };
}) | null;
/**
 * Test triggers keymap
 */
export function getKeyMap(opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestTriggerKeyMap;
    }>("/keymap/triggers", {
        ...opts
    });
}
/**
 * List test triggers
 */
export function listTestTriggers({ $namespace, selector }: {
    $namespace?: string;
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestTrigger[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/triggers${QS.query(QS.explode({
        "namespace": $namespace,
        selector
    }))}`, {
        ...opts
    });
}
/**
 * Create new test trigger
 */
export function createTestTrigger(testTriggerUpsertRequest: TestTriggerUpsertRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestTrigger;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/triggers", oazapfts.json({
        ...opts,
        method: "POST",
        body: testTriggerUpsertRequest
    }));
}
/**
 * Bulk update test triggers
 */
export function bulkUpdateTestTriggers(body: TestTriggerUpsertRequest[], opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestTrigger[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/triggers", oazapfts.json({
        ...opts,
        method: "PATCH",
        body
    }));
}
/**
 * Delete test triggers
 */
export function deleteTestTriggers({ $namespace, selector }: {
    $namespace?: string;
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/triggers${QS.query(QS.explode({
        "namespace": $namespace,
        selector
    }))}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Get test trigger by ID
 */
export function getTestTriggerById(id: string, { $namespace }: {
    $namespace?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestTrigger;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/triggers/${encodeURIComponent(id)}${QS.query(QS.explode({
        "namespace": $namespace
    }))}`, {
        ...opts
    });
}
/**
 * Update test trigger
 */
export function updateTestTrigger(id: string, testTriggerUpsertRequest: TestTriggerUpsertRequest, { $namespace }: {
    $namespace?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestTrigger;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/triggers/${encodeURIComponent(id)}${QS.query(QS.explode({
        "namespace": $namespace
    }))}`, oazapfts.json({
        ...opts,
        method: "PATCH",
        body: testTriggerUpsertRequest
    }));
}
/**
 * Delete test trigger
 */
export function deleteTestTrigger(id: string, { $namespace }: {
    $namespace?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/triggers/${encodeURIComponent(id)}${QS.query(QS.explode({
        "namespace": $namespace
    }))}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Create new test suite
 */
export function createTestSuite(testSuiteUpsertRequest: TestSuiteUpsertRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: string;
    } | {
        status: 201;
        data: TestSuite;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/test-suites", oazapfts.json({
        ...opts,
        method: "POST",
        body: testSuiteUpsertRequest
    }));
}
/**
 * Get all test suites
 */
export function listTestSuites({ selector, textSearch }: {
    selector?: string;
    textSearch?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuite[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites${QS.query(QS.explode({
        selector,
        textSearch
    }))}`, {
        ...opts
    });
}
/**
 * Delete test suites
 */
export function deleteTestSuites({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Get test suite by ID
 */
export function getTestSuiteById(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuite;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}`, {
        ...opts
    });
}
/**
 * Update test suite
 */
export function updateTestSuite(id: string, testSuiteUpdateRequest: TestSuiteUpdateRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuite;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}`, oazapfts.json({
        ...opts,
        method: "PATCH",
        body: testSuiteUpdateRequest
    }));
}
/**
 * Delete test suite
 */
export function deleteTestSuite(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Get test suite metrics
 */
export function getTestSuiteMetrics(id: string, { last, limit }: {
    last?: number;
    limit?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutionsMetrics;
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/metrics${QS.query(QS.explode({
        last,
        limit
    }))}`, {
        ...opts
    });
}
/**
 * List tests for test suite
 */
export function listTestSuiteTests(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Test[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/tests`, {
        ...opts
    });
}
/**
 * Abort all executions of a test suite
 */
export function abortTestSuiteExecutions({ name }: {
    name?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/abort${QS.query(QS.explode({
        name
    }))}`, {
        ...opts,
        method: "POST"
    });
}
/**
 * Get all test suite with executions
 */
export function listTestSuiteWithExecutions({ selector, textSearch, status, pageSize, page }: {
    selector?: string;
    textSearch?: string;
    status?: TestSuiteExecutionStatus;
    pageSize?: number;
    page?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuiteWithExecutionSummary[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suite-with-executions${QS.query(QS.explode({
        selector,
        textSearch,
        status,
        pageSize,
        page
    }))}`, {
        ...opts
    });
}
/**
 * Get test suite by ID with execution
 */
export function getTestSuiteByIdWithExecution(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuiteWithExecution;
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suite-with-executions/${encodeURIComponent(id)}`, {
        ...opts
    });
}
/**
 * Starts new test suite execution
 */
export function executeTestSuite(id: string, testSuiteExecutionRequest: TestSuiteExecutionRequest, { $namespace, last }: {
    $namespace?: string;
    last?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 201;
        data: TestSuiteExecutionsResult;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/executions${QS.query(QS.explode({
        "namespace": $namespace,
        last
    }))}`, oazapfts.json({
        ...opts,
        method: "POST",
        body: testSuiteExecutionRequest
    }));
}
/**
 * Get all test suite executions
 */
export function listTestSuiteExecutions(id: string, { pageSize, page, status, startDate, endDate }: {
    pageSize?: number;
    page?: number;
    status?: TestSuiteExecutionStatus;
    startDate?: string;
    endDate?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuiteExecutionsResult;
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/executions${QS.query(QS.explode({
        pageSize,
        page,
        status,
        startDate,
        endDate
    }))}`, {
        ...opts
    });
}
/**
 * Get test suite execution
 */
export function getTestSuiteExecution(id: string, executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuiteExecutionRead;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/executions/${encodeURIComponent(executionId)}`, {
        ...opts
    });
}
/**
 * Aborts testsuite execution
 */
export function abortTestSuiteExecution(id: string, executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/executions/${encodeURIComponent(executionId)}`, {
        ...opts,
        method: "PATCH"
    });
}
/**
 * Get test suite execution artifacts
 */
export function getTestSuiteExecutionArtifactsByTestsuite(id: string, executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Artifact;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suites/${encodeURIComponent(id)}/executions/${encodeURIComponent(executionId)}/artifacts`, {
        ...opts
    });
}
/**
 * Starts new test suite executions
 */
export function executeTestSuites(testSuiteExecutionRequest: TestSuiteExecutionRequest, { $namespace, selector, concurrency }: {
    $namespace?: string;
    selector?: string;
    concurrency?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 201;
        data: TestSuiteExecutionsResult[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suite-executions${QS.query(QS.explode({
        "namespace": $namespace,
        selector,
        concurrency
    }))}`, oazapfts.json({
        ...opts,
        method: "POST",
        body: testSuiteExecutionRequest
    }));
}
/**
 * Get all test suite executions
 */
export function listAllTestSuiteExecutions({ last, test, textSearch, pageSize, page, status, startDate, endDate, selector }: {
    last?: number;
    test?: string;
    textSearch?: string;
    pageSize?: number;
    page?: number;
    status?: TestSuiteExecutionStatus;
    startDate?: string;
    endDate?: string;
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuiteExecutionsResult;
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suite-executions${QS.query(QS.explode({
        last,
        test,
        textSearch,
        pageSize,
        page,
        status,
        startDate,
        endDate,
        selector
    }))}`, {
        ...opts
    });
}
/**
 * Get test suite execution by ID
 */
export function getTestSuiteExecutionById(executionId: string, { last }: {
    last?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSuiteExecutionRead;
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suite-executions/${encodeURIComponent(executionId)}${QS.query(QS.explode({
        last
    }))}`, {
        ...opts
    });
}
/**
 * Aborts testsuite execution
 */
export function abortTestSuiteExecutionById(executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-suite-executions/${encodeURIComponent(executionId)}`, {
        ...opts,
        method: "PATCH"
    });
}
/**
 * Get test suite execution artifacts
 */
export function getTestSuiteExecutionArtifacts(executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Artifact;
    } | {
        status: 500;
        data: Problem[];
    }>(`/test-suite-executions/${encodeURIComponent(executionId)}/artifacts`, {
        ...opts
    });
}
/**
 * Starts new test executions
 */
export function executeTests(executionRequest: ExecutionRequest, { $namespace, selector, executionSelector, concurrency }: {
    $namespace?: string;
    selector?: string;
    executionSelector?: string;
    concurrency?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 201;
        data: ExecutionResult[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/executions${QS.query(QS.explode({
        "namespace": $namespace,
        selector,
        executionSelector,
        concurrency
    }))}`, oazapfts.json({
        ...opts,
        method: "POST",
        body: executionRequest
    }));
}
/**
 * Get all test executions
 */
export function listExecutions({ test, $type, textSearch, pageSize, page, status, startDate, endDate, selector }: {
    test?: string;
    $type?: string;
    textSearch?: string;
    pageSize?: number;
    page?: number;
    status?: ExecutionStatus;
    startDate?: string;
    endDate?: string;
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutionsResult;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/executions${QS.query(QS.explode({
        test,
        "type": $type,
        textSearch,
        pageSize,
        page,
        status,
        startDate,
        endDate,
        selector
    }))}`, {
        ...opts
    });
}
/**
 * Get test execution by ID
 */
export function getExecutionById(executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutionRead;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/executions/${encodeURIComponent(executionId)}`, {
        ...opts
    });
}
/**
 * Get execution's artifacts by ID
 */
export function getExecutionArtifacts(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Artifact[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/executions/${encodeURIComponent(id)}/artifacts`, {
        ...opts
    });
}
/**
 * Get execution's logs by ID
 */
export function getExecutionLogs(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutorOutput[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/executions/${encodeURIComponent(id)}/logs`, {
        ...opts
    });
}
/**
 * Download artifact
 */
export function downloadFile(id: string, filename: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Blob;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/executions/${encodeURIComponent(id)}/artifacts/${encodeURIComponent(filename)}`, {
        ...opts
    });
}
/**
 * Download artifact archive
 */
export function downloadArchive(id: string, { mask }: {
    mask?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Blob;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/executions/${encodeURIComponent(id)}/artifact-archive${QS.query(QS.explode({
        mask
    }))}`, {
        ...opts
    });
}
/**
 * List tests
 */
export function listTests({ selector, textSearch }: {
    selector?: string;
    textSearch?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Test[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests${QS.query(QS.explode({
        selector,
        textSearch
    }))}`, {
        ...opts
    });
}
/**
 * Create new test
 */
export function createTest(testUpsertRequest: TestUpsertRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: string;
    } | {
        status: 201;
        data: Test;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/tests", oazapfts.json({
        ...opts,
        method: "POST",
        body: testUpsertRequest
    }));
}
/**
 * Delete tests
 */
export function deleteTests({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Update test
 */
export function updateTest(id: string, testUpdateRequest: TestUpdateRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Test;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}`, oazapfts.json({
        ...opts,
        method: "PATCH",
        body: testUpdateRequest
    }));
}
/**
 * Get test
 */
export function getTest(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Test;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}`, {
        ...opts
    });
}
/**
 * Delete test
 */
export function deleteTest(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Abort all executions of a test
 */
export function abortTestExecutions(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}/abort`, {
        ...opts,
        method: "POST"
    });
}
/**
 * Get test metrics
 */
export function getTestMetrics(id: string, { last, limit }: {
    last?: number;
    limit?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutionsMetrics;
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}/metrics${QS.query(QS.explode({
        last,
        limit
    }))}`, {
        ...opts
    });
}
/**
 * List test with executions
 */
export function listTestWithExecutions({ selector, textSearch, status, pageSize, page }: {
    selector?: string;
    textSearch?: string;
    status?: ExecutionStatus;
    pageSize?: number;
    page?: number;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestWithExecutionSummary[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-with-executions${QS.query(QS.explode({
        selector,
        textSearch,
        status,
        pageSize,
        page
    }))}`, {
        ...opts
    });
}
/**
 * Get test with execution
 */
export function getTestWithExecution(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestWithExecution;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-with-executions/${encodeURIComponent(id)}`, {
        ...opts
    });
}
/**
 * Starts new test execution
 */
export function executeTest(id: string, executionRequest: ExecutionRequest, { $namespace }: {
    $namespace?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 201;
        data: ExecutionResult;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}/executions${QS.query(QS.explode({
        "namespace": $namespace
    }))}`, oazapfts.json({
        ...opts,
        method: "POST",
        body: executionRequest
    }));
}
/**
 * Get all test executions
 */
export function listTestExecutions(id: string, { last, pageSize, page, status, startDate, endDate }: {
    last?: number;
    pageSize?: number;
    page?: number;
    status?: ExecutionStatus;
    startDate?: string;
    endDate?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutionsResult;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}/executions${QS.query(QS.explode({
        last,
        pageSize,
        page,
        status,
        startDate,
        endDate
    }))}`, {
        ...opts
    });
}
/**
 * Get test execution
 */
export function getTestExecution(id: string, executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutionRead;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}/executions/${encodeURIComponent(executionId)}`, {
        ...opts
    });
}
/**
 * Aborts execution
 */
export function abortExecution(id: string, executionId: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 502;
        data: Problem[];
    }>(`/tests/${encodeURIComponent(id)}/executions/${encodeURIComponent(executionId)}`, {
        ...opts,
        method: "PATCH"
    });
}
/**
 * List executors
 */
export function listExecutors({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Executor[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/executors${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts
    });
}
/**
 * Create new executor
 */
export function createExecutor(executorUpsertRequest: ExecutorUpsertRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: string;
    } | {
        status: 201;
        data: ExecutorDetails;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/executors", oazapfts.json({
        ...opts,
        method: "POST",
        body: executorUpsertRequest
    }));
}
/**
 * Delete executors
 */
export function deleteExecutors({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 502;
        data: Problem[];
    }>(`/executors${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Delete executor
 */
export function deleteExecutor(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 502;
        data: Problem[];
    }>(`/executors/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Get executor details
 */
export function getExecutor(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutorDetails;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/executors/${encodeURIComponent(id)}`, {
        ...opts
    });
}
/**
 * Update executor
 */
export function updateExecutor(id: string, executorUpdateRequest: ExecutorUpdateRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: ExecutorDetails;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/executors/${encodeURIComponent(id)}`, oazapfts.json({
        ...opts,
        method: "PATCH",
        body: executorUpdateRequest
    }));
}
/**
 * List labels
 */
export function listLabels(opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: {
            [key: string]: string[];
        };
    } | {
        status: 502;
        data: Problem[];
    }>("/labels", {
        ...opts
    });
}
/**
 * List webhooks
 */
export function listWebhooks({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Webhook[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/webhooks${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts
    });
}
/**
 * Create new webhook
 */
export function createWebhook(webhookCreateRequest: WebhookCreateRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: string;
    } | {
        status: 201;
        data: Webhook;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/webhooks", oazapfts.json({
        ...opts,
        method: "POST",
        body: webhookCreateRequest
    }));
}
/**
 * Delete webhooks
 */
export function deleteWebhooks({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 502;
        data: Problem[];
    }>(`/webhooks${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Delete webhook
 */
export function deleteWebhook(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/webhooks/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Get webhook details
 */
export function getWebhook(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Webhook;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/webhooks/${encodeURIComponent(id)}`, {
        ...opts
    });
}
/**
 * Update config
 */
export function updateConfigKey(config: Config, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Config;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>("/config", oazapfts.json({
        ...opts,
        method: "PATCH",
        body: config
    }));
}
/**
 * Get config
 */
export function getConfig(opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: Config;
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>("/config", {
        ...opts
    });
}
/**
 * Get debug information
 */
export function getDebugInfo(opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: DebugInfo;
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/debug", {
        ...opts
    });
}
/**
 * List test sources
 */
export function listTestSources({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSource[];
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-sources${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts
    });
}
/**
 * Create new test source
 */
export function createTestSource(testSourceUpsertRequest: TestSourceUpsertRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: string;
    } | {
        status: 201;
        data: TestSource;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/test-sources", oazapfts.json({
        ...opts,
        method: "POST",
        body: testSourceUpsertRequest
    }));
}
/**
 * Process test source batch (create, update, delete)
 */
export function processTestSourceBatch(testSourceBatchRequest: TestSourceBatchRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSourceBatchResult;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/test-sources", oazapfts.json({
        ...opts,
        method: "PATCH",
        body: testSourceBatchRequest
    }));
}
/**
 * Delete test sources
 */
export function deleteTestSources({ selector }: {
    selector?: string;
} = {}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-sources${QS.query(QS.explode({
        selector
    }))}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Update test source
 */
export function updateTestSource(id: string, testSourceUpdateRequest: TestSourceUpdateRequest, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSource;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-sources/${encodeURIComponent(id)}`, oazapfts.json({
        ...opts,
        method: "PATCH",
        body: testSourceUpdateRequest
    }));
}
/**
 * Delete test source
 */
export function deleteTestSource(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-sources/${encodeURIComponent(id)}`, {
        ...opts,
        method: "DELETE"
    });
}
/**
 * Get test source data
 */
export function getTestSource(id: string, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: TestSource;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 404;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>(`/test-sources/${encodeURIComponent(id)}`, {
        ...opts
    });
}
/**
 * Upload file
 */
export function uploads(body: {
    parentName?: string;
    parentType?: "test" | "execution";
    filePath?: string;
}, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 200;
        data: string;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    }>("/uploads", oazapfts.multipart({
        ...opts,
        method: "POST",
        body
    }));
}
/**
 * Validate new repository
 */
export function validateRepository(repository: Repository, opts?: Oazapfts.RequestOpts) {
    return oazapfts.fetchJson<{
        status: 204;
    } | {
        status: 400;
        data: Problem[];
    } | {
        status: 500;
        data: Problem[];
    } | {
        status: 502;
        data: Problem[];
    }>("/repositories", oazapfts.json({
        ...opts,
        method: "POST",
        body: repository
    }));
}
