import Admonition from "@theme/Admonition";

# JMeter

[JMeter](https://jmeter.apache.org/) is an integral part of Testkube. The Testkube JMeter executor is installed by default during the Testkube installation.

* Default command for this executor: `<entryPoint>`
* Default arguments for this executor command: `-n` `-j` `<logFile>` `-t` `<runPath>` `-l` `<jtlFile>` `-e` `-o` `<reportFile>` `<envVars>`

Parameters in `<>` are calculated at test execution:

* `<entryPoint>` - The entrypoint for the JMeter runner set by the environment variable `ENTRYPOINT_CMD`, defaults to the file in `contrib/executor/jmeter/scripts/entrypoint.sh`.
* `<logFile>` - JMeter log path
* `<runPath>` - path to the test files
* `<jtlFile>` - path to the jrl report file
* `<reportFile>` - path to the report
* `<envVars>` - list of environment variables

[See more at "Redefining the Prebuilt Executor Command and Arguments" on the Creating Test page.](../articles/creating-tests.md#redefining-the-prebuilt-executor-command-and-arguments)

<iframe width="100%" height="315" src="https://www.youtube.com/embed/iF7BcVqTeO0" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>


export const ExecutorInfo = () => {
  return (
    <div>
      <Admonition type="info" icon="🎓" title="What is JMeter?">
        <ul>
          <li>Apache JMeter is an open-source, pure Java application designed to load test functional behavior and measure performance. It was originally designed for testing Web Applications but has since expanded to other test functions.</li>
        </ul>
        <b>What can I do with JMeter?</b>
        <ul>
          <li>JMeter can be used to test performance both on static and dynamic resources and Web dynamic applications.</li>
          <li>It can also be used to simulate a heavy load on a server, group of servers, network, or object to test its strength or to analyze overall performance under different load types.</li>
        </ul>
      </Admonition>
    </div>
  );
}

<ExecutorInfo />


**Check out our [blog post](https://testkube.io/blog/jmeter-and-kubernetes-how-to-run-tests-efficiently-with-testkube) to follow tutorial steps for end-to-end testing of your Kubernetes applications with JMeter.**


## Running a JMeter Test

### Using Files as Input

Let's save our JMeter test in file e.g. `test.jmx`. 

```xml 
<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.5">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Kubeshop site" enabled="true">
      <stringProp name="TestPlan.comments">Kubeshop site simple perf test</stringProp>
      <boolProp name="TestPlan.functional_mode">false</boolProp>
      <boolProp name="TestPlan.tearDown_on_shutdown">true</boolProp>
      <boolProp name="TestPlan.serialize_threadgroups">false</boolProp>
      <elementProp name="TestPlan.user_defined_variables" elementType="Arguments" guiclass="ArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
        <collectionProp name="Arguments.arguments">
          <elementProp name="PATH" elementType="Argument">
            <stringProp name="Argument.name">PATH</stringProp>
            <stringProp name="Argument.value">/pricing</stringProp>
            <stringProp name="Argument.metadata">=</stringProp>
          </elementProp>
        </collectionProp>
      </elementProp>
      <stringProp name="TestPlan.user_define_classpath"></stringProp>
    </TestPlan>
    <hashTree>
      <ThreadGroup guiclass="ThreadGroupGui" testclass="ThreadGroup" testname="Thread Group" enabled="true">
        <stringProp name="ThreadGroup.on_sample_error">continue</stringProp>
        <elementProp name="ThreadGroup.main_controller" elementType="LoopController" guiclass="LoopControlPanel" testclass="LoopController" testname="Loop Controller" enabled="true">
          <boolProp name="LoopController.continue_forever">false</boolProp>
          <stringProp name="LoopController.loops">1</stringProp>
        </elementProp>
        <stringProp name="ThreadGroup.num_threads">1</stringProp>
        <stringProp name="ThreadGroup.ramp_time">1</stringProp>
        <boolProp name="ThreadGroup.scheduler">false</boolProp>
        <stringProp name="ThreadGroup.duration"></stringProp>
        <stringProp name="ThreadGroup.delay"></stringProp>
        <boolProp name="ThreadGroup.same_user_on_next_iteration">true</boolProp>
      </ThreadGroup>
      <hashTree>
        <HTTPSamplerProxy guiclass="HttpTestSampleGui" testclass="HTTPSamplerProxy" testname="HTTP Request" enabled="true">
          <elementProp name="HTTPsampler.Arguments" elementType="Arguments" guiclass="HTTPArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
            <collectionProp name="Arguments.arguments">
              <elementProp name="PATH" elementType="HTTPArgument">
                <boolProp name="HTTPArgument.always_encode">false</boolProp>
                <stringProp name="Argument.value">$PATH</stringProp>
                <stringProp name="Argument.metadata">=</stringProp>
                <boolProp name="HTTPArgument.use_equals">true</boolProp>
                <stringProp name="Argument.name">PATH</stringProp>
              </elementProp>
            </collectionProp>
          </elementProp>
          <stringProp name="HTTPSampler.domain">testkube.io</stringProp>
          <stringProp name="HTTPSampler.port">80</stringProp>
          <stringProp name="HTTPSampler.protocol">https</stringProp>
          <stringProp name="HTTPSampler.contentEncoding"></stringProp>
          <stringProp name="HTTPSampler.path">https://testkube.io</stringProp>
          <stringProp name="HTTPSampler.method">GET</stringProp>
          <boolProp name="HTTPSampler.follow_redirects">true</boolProp>
          <boolProp name="HTTPSampler.auto_redirects">false</boolProp>
          <boolProp name="HTTPSampler.use_keepalive">true</boolProp>
          <boolProp name="HTTPSampler.DO_MULTIPART_POST">false</boolProp>
          <stringProp name="HTTPSampler.embedded_url_re"></stringProp>
          <stringProp name="HTTPSampler.connect_timeout"></stringProp>
          <stringProp name="HTTPSampler.response_timeout"></stringProp>
        </HTTPSamplerProxy>
        <hashTree>
          <ResponseAssertion guiclass="AssertionGui" testclass="ResponseAssertion" testname="Response Assertion" enabled="true">
            <collectionProp name="Asserion.test_strings">
              <stringProp name="-1081444641">Testkube</stringProp>
            </collectionProp>
            <stringProp name="Assertion.custom_message"></stringProp>
            <stringProp name="Assertion.test_field">Assertion.response_data</stringProp>
            <boolProp name="Assertion.assume_success">false</boolProp>
            <intProp name="Assertion.test_type">16</intProp>
          </ResponseAssertion>
          <hashTree/>
        </hashTree>
      </hashTree>
    </hashTree>
  </hashTree>
</jmeterTestPlan>

```

The Testkube JMeter executor accepts a test file as an input.

```bash
kubectl testkube create test --file test.jmx --name jmeter-test --type jmeter/test
```
You don't need to pass a type here, Testkube will autodetect it. 


To run the test, pass the previously created test name: 

```bash 
kubectl testkube run test -f jmeter-test
```

You can also create a Test based on a Git repository:

```bash
# create test from this Git repository
kubectl testkube create test --test-content-type git --git-uri https://github.com/kubeshop/testkube-executor-jmeter.git --git-branch main --git-path examples/kubeshop.jmx --type jmeter/test --name jmeter-test-from-git
```

Testkube will clone the repository and create a Testkube Test Custom Resource in your cluster automatically on each test run. 

### Using Additional JMeter Arguments in Your Tests

You can also pass additional arguments to the `jmeter` binary thanks to the `--args` flag:

```bash
$ kubectl testkube run test -f jmeter-test --args '-LsutHost=https://staging.kubeshop.com -LsomeParam=someValue'
```

### **JMeter Test Results**

A JMeter test will be successful in Testkube when all checks and thresholds are successful. In the case of an error, the test will have `failed` status,
and the JMeter executor is configured to store the `report.jtl` file after the test run. You can get the file from the "Artifacts" tab in the execution results in the Testkube Dashboard, 
or download it with the `testkube get artifacts EXECUTION_ID` command.

### **JMeter Memory Consumption**

When running load tests, it is common to run into memory-related issues, for example the `OOMKilled` status in Kubernetes. There are three areas where the memory requests and limits can be set:
* on JVM level, inside the pod
* on pod/job level
* on cluster level

In most cases, it is wise to start from setting the correct variables for the JVM. As the [README of the underlying image](https://github.com/justb4/docker-jmeter#adjust-java-memory-options) suggests, the available values are `JVM_XMN`, `JVM_XMS` and `JVM_XMX`.

>By default, JMeter reads out the available memory from the host machine and uses a fixed value of 80% of it as a maximum. If this causes Issues, there is the option to use environment variables to adjust the JVM memory Parameters:
>
>JVM_XMN to adjust maximum nursery size
>
>JVM_XMS to adjust initial heap size
>
>JVM_XMX to adjust maximum heap size
>
>All three use values in Megabyte range.

If this does not fix your issue, look into using Testkube job templates to set job-level [requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).

In the very rare case that you are still bumping into memory issues, consider asking your Kubernetes cluster manager about any [resource quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/).
