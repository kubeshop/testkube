---
sidebar_position: 4
sidebar_label: JMeter
---
# JMeter Performance Tests

The Apache JMeter™ application is open source software, a 100% pure Java application designed to load test functional behavior and measure performance. It was originally designed for testing Web Applications but has since expanded to other test functions.

## What can I do with it?

Apache JMeter may be used to test performance both on static and dynamic resources and Web dynamic applications.
It can be used to simulate a heavy load on a server, group of servers, network or object to test its strength or to analyze overall performance under different load types.

[JMeter](https://jmeter.apache.org/) 


## **Running a JMeter Test**

JMeter is integral part of Testkube. The Testkube JMeter executor is installed by default during the Testkube installation. To run a JMeter test in Testkube you need to create a Test. 

### **Using Files as Input**

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
kubectl testkube create test --test-content-type git-file --git-uri https://github.com/kubeshop/testkube-executor-jmeter.git --git-branch main --git-path examples/kubeshop.jmx --type jmeter/test --name jmeter-test-from-git
```

Testkube will clone the repository and create a Testkube Test Custom Resource in your cluster automatically on each test run. 

### **Using Additional JMeter Arguments in Your Tests**

You can also pass additional arguments to the `jmeter` binary thanks to the `--args` flag:

```bash
$ kubectl testkube run test -f jmeter-test --args '-LsutHost=https://staging.kubeshop.com -LsomeParam=someValue'
```

### **JMeter Test Results**

A JMeter test will be successful in Testkube when all checks and thresholds are successful. In the case of an error, the test will have `failed` status,
JMeter executor is configured to store the `report.jtl` file after the test run. You can get the file from the "Artifacts" tab in the execution results in Testkube UI, 
or download it with the `testkube get artifacts EXECUTION_ID` command.


