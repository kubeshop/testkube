import Admonition from "@theme/Admonition";

# SoapUI

[SoapUI](https://www.soapui.org) is an open-source tool used for the end-to-end testing of REST, SOAP and GraphQL APIs, as well as JMS, JDBC and other web services. Testkube supports the SoapUI executor implementation.

* Default command for this executor: `/bin/sh` `/usr/local/SmartBear/EntryPoint.sh`
* Default arguments for this executor command: `<runPath>`

Parameters in `<>` are calculated at test execution:

* `<runPath>` - path to the test files

[See more at "Redefining the Prebuilt Executor Command and Arguments" on the Creating Test page.](../articles/creating-tests.md#redefining-the-prebuilt-executor-command-and-arguments)

**Check out our [blog post](https://kubeshop.io/blog/run-kubernetes-tests-with-soapui-and-testkube) to follow tutorial steps to Learn how to run functional tests in Kubernetes with SoapUI and Testkube.**

Testkube supports the [SoapUI](https://www.soapui.org) executor implementation.

export const ExecutorInfo = () => {
   return (
    <div>
      <Admonition type="info" icon="🎓" title="What is SoapUI?">
        <ul>
          <li>SoapUI is an open-source tool used for end-to-end testing of REST, SOAP and GraphQL APIs, as well as JMS, JDBC and other web services.</li>
        </ul>
      </Admonition>
    </div>
  );
}

## **Running a SoapUI Test**

In order to run a SoapUI test using Testkube, begin by creating a Testkube Test.

An example of an exported SoapUI test looks the following:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<con:soapui-project xmlns:con="http://eviware.com/soapui/config" id="68931eeb-521d-4870-972f-9d0f99c75cc2" activeEnvironment="Default" name="Testkube project" resourceRoot="${projectDir}" soapui-version="5.7.0" abortOnError="false" runType="SEQUENTIAL">
   <con:settings>
      <con:setting id="com.smartbear.swagger.ExportSwaggerAction$FormBase Path" />
      <con:setting id="com.smartbear.swagger.ExportSwaggerAction$FormTarget File" />
      <con:setting id="com.smartbear.swagger.ExportSwaggerAction$FormFormat">json</con:setting>
      <con:setting id="com.smartbear.swagger.ExportSwaggerAction$FormAPI Version">Swagger 2.0</con:setting>
      <con:setting id="com.smartbear.swagger.ExportSwaggerAction$FormSwagger Version">Swagger 2.0</con:setting>
   </con:settings>
   <con:interface xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="con:RestService" id="4bcedf3f-ac64-4f45-9615-7d58b15b9ca1" wadlVersion="http://wadl.dev.java.net/2009/02" name="https://kubeshop.io" type="rest">
      <con:settings />
      <con:definitionCache type="TEXT" rootPart="" />
      <con:endpoints>
         <con:endpoint>https://kubeshop.io</con:endpoint>
      </con:endpoints>
      <con:resource name="" path="" id="538412a5-540d-4772-aca8-2b19e456af77">
         <con:settings />
         <con:parameters />
         <con:method name="1" id="fa5cf353-bafd-4aa1-9fa6-301d0d5c6e95" method="GET">
            <con:settings />
            <con:parameters />
            <con:representation type="RESPONSE">
               <con:mediaType>text/html</con:mediaType>
               <con:status>200</con:status>
               <con:params />
               <con:element>html</con:element>
            </con:representation>
            <con:representation type="RESPONSE">
               <con:mediaType>application/json</con:mediaType>
               <con:status>200</con:status>
               <con:params />
               <con:element xmlns:pet="https://kubeshop.io/">pet:Response</con:element>
            </con:representation>
            <con:representation type="RESPONSE">
               <con:mediaType>text/html; charset=utf-8</con:mediaType>
               <con:status>200</con:status>
               <con:params />
               <con:element>html</con:element>
            </con:representation>
            <con:request name="Request 1" id="e5ee1b97-e7a5-4fd0-9ecd-6671558f25bc" mediaType="application/json">
               <con:settings>
                  <con:setting id="com.eviware.soapui.impl.wsdl.WsdlRequest@request-headers">&lt;xml-fragment/&gt;</con:setting>
               </con:settings>
               <con:endpoint>https://kubeshop.io</con:endpoint>
               <con:request />
               <con:originalUri>https://kubeshop.io/</con:originalUri>
               <con:credentials>
                  <con:authType>No Authorization</con:authType>
               </con:credentials>
               <con:jmsConfig JMSDeliveryMode="PERSISTENT" />
               <con:jmsPropertyConfig />
               <con:parameters />
            </con:request>
         </con:method>
      </con:resource>
   </con:interface>
   <con:testSuite id="d01755ab-653a-42e9-82ef-97ffea557480" name="Testkube TestSuite">
      <con:settings />
      <con:runType>SEQUENTIAL</con:runType>
      <con:testCase id="8803628b-cc8d-4aaa-bbba-e1bc878f202b" failOnError="true" failTestCaseOnErrors="true" keepSession="false" maxResults="0" name="Kubeshop TestCase" searchProperties="true">
         <con:settings />
         <con:testStep type="restrequest" name="1 - Request 1" id="00e86b82-8ef2-483f-837b-7669bebe3c87">
            <con:settings />
            <con:config xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" service="https://kubeshop.io" resourcePath="" methodName="1" xsi:type="con:RestRequestStep">
               <con:restRequest name="1 - Request 1" id="e5ee1b97-e7a5-4fd0-9ecd-6671558f25bc" mediaType="application/json">
                  <con:settings>
                     <con:setting id="com.eviware.soapui.impl.wsdl.WsdlRequest@request-headers">&lt;xml-fragment/&gt;</con:setting>
                  </con:settings>
                  <con:endpoint>https://kubeshop.io</con:endpoint>
                  <con:request />
                  <con:originalUri>https://kubeshop.io</con:originalUri>
                  <con:assertion type="Valid HTTP Status Codes" id="c88e07cf-dd80-4b11-b21d-7cad8474202b" name="Valid HTTP Status Codes">
                     <con:configuration>
                        <codes>200</codes>
                     </con:configuration>
                  </con:assertion>
                  <con:assertion type="Simple Contains" id="7072dc46-6b93-43b4-b6bd-0464db7b249e" name="Contains">
                     <con:configuration>
                        <token>testkube</token>
                        <ignoreCase>true</ignoreCase>
                        <useRegEx>false</useRegEx>
                     </con:configuration>
                  </con:assertion>
                  <con:credentials>
                     <con:authType>No Authorization</con:authType>
                  </con:credentials>
                  <con:jmsConfig JMSDeliveryMode="PERSISTENT" />
                  <con:jmsPropertyConfig />
                  <con:parameters />
               </con:restRequest>
            </con:config>
         </con:testStep>
         <con:properties />
      </con:testCase>
      <con:testCase id="4c0fc01b-16db-41e9-9844-d4c2a88d27f2" failOnError="true" failTestCaseOnErrors="true" keepSession="false" maxResults="0" name="Testkube TestCase" searchProperties="true">
         <con:settings />
         <con:testStep type="restrequest" name="REST Request" id="e1be7e01-5d4d-424e-b2f9-7daf84aaafa9">
            <con:settings />
            <con:config xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" service="https://kubeshop.io" methodName="1" resourcePath="" xsi:type="con:RestRequestStep">
               <con:restRequest name="REST Request" id="191f16b4-cb04-4ffa-b8f9-da0e788c8262" mediaType="application/json">
                  <con:settings>
                     <con:setting id="com.eviware.soapui.impl.wsdl.WsdlRequest@request-headers">&lt;xml-fragment/&gt;</con:setting>
                  </con:settings>
                  <con:encoding>UTF-8</con:encoding>
                  <con:endpoint>https://docs.testkube.io</con:endpoint>
                  <con:request />
                  <con:originalUri>https://kubeshop.io</con:originalUri>
                  <con:assertion type="Simple Contains" id="d9497693-01e2-4e3e-8ce5-5a292f9b6e41" name="Contains">
                     <con:configuration>
                        <token>kubectl</token>
                        <ignoreCase>false</ignoreCase>
                        <useRegEx>false</useRegEx>
                     </con:configuration>
                  </con:assertion>
                  <con:credentials>
                     <con:authType>No Authorization</con:authType>
                  </con:credentials>
                  <con:jmsConfig JMSDeliveryMode="PERSISTENT" />
                  <con:jmsPropertyConfig />
                  <con:parameters />
               </con:restRequest>
            </con:config>
         </con:testStep>
         <con:properties />
      </con:testCase>
      <con:properties />
   </con:testSuite>
   <con:properties />
   <con:wssContainer />
   <con:oAuth2ProfileContainer />
   <con:oAuth1ProfileContainer />
   <con:sensitiveInformation />
</con:soapui-project>
```

### **Using Files as Input**

Testkube and the SoapUI executor accept a project file as input.

```bash
$ kubectl testkube create test --file example-project.xml --type soapui/xml --name example-test
Test created  / example-test 🥇

```

### **Using Strings as Input**

```bash
$ cat example-project.xml | kubectl testkube create test --type soapui/xml --name example-test-string
Test created  / example-test-string 🥇

```

### **Running the Tests**

To run the created test, use:

```bash
$ kubectl testkube run test example-test

Type          : soapui/xml
Name          : example-test
Execution ID  : 624eedd443ed8485ae9289e2
Execution name: illegally-credible-mouse



Test execution started

Watch test execution until complete:
$ kubectl testkube watch execution 624eedd443ed8485ae9289e2


Use following command to get test execution details:
$ kubectl testkube get execution 624eedd443ed8485ae9289e2

```

### **Using Parameters and Arguments in Your Tests**

SoapUI lets you configure your test runs using different parameters. To see all available command line arguments, check the [official SoapUI docs](https://www.soapui.org/docs/test-automation/running-functional-tests/).

To use parameters when working with Testkube, use the `kubectl testkube run` command with the `--args` parameter.

An example would be:

```bash
$ kubectl testkube run test -f example-test --args '-I -c "Testkube TestCase"'

Type          : soapui/xml
Name          : successful-test
Execution ID  : 625404e5a4cc6d2861193c60
Execution name: currently-amused-pug


Getting pod logs
Execution completed ================================
=
= SOAPUI_HOME = /usr/local/SmartBear/SoapUI-5.7.0
=
================================
SoapUI 5.7.0 TestCase Runner
08:14:18,671 INFO  [DefaultSoapUICore] Creating new settings at [/root/soapui-settings.xml]
08:14:27,457 INFO  [PluginManager] 0 plugins loaded in 11 ms
08:14:27,459 INFO  [DefaultSoapUICore] All plugins loaded
08:14:37,509 INFO  [WsdlProject] Loaded project from [file:/tmp/test-content2412821894]
08:14:37,579 INFO  [SoapUITestCaseRunner] Running SoapUI tests in project [Testkube project]
08:14:37,580 INFO  [SoapUITestCaseRunner] Running TestCase [Testkube TestCase]
08:14:37,653 INFO  [SoapUITestCaseRunner] Running SoapUI testcase [Testkube TestCase]
08:14:37,700 INFO  [SoapUITestCaseRunner] running step [REST Request]
08:14:41,816 INFO  [SoapUITestCaseRunner] Assertion [Contains] has status VALID
08:14:41,878 INFO  [SoapUITestCaseRunner] Finished running SoapUI testcase [Testkube TestCase], time taken: 866ms, status: FINISHED
08:14:41,909 INFO  [SoapUITestCaseRunner] TestCase [Testkube TestCase] finished with status [FINISHED] in 866ms


.
Use the following command to get test execution details:
$ kubectl testkube get execution 625404e5a4cc6d2861193c60
```

## **Reports, Plugins and Extensions**

In order to use reports, add the plugins and extensions as described in the [SoapUI docs](https://www.soapui.org/docs/test-automation/running-in-docker/). This is currently not supported by Testkube.
If you are interested in using this feature, please create an [issue](https://github.com/kubeshop/testkube/issues) in the [Testkube repository](https://github.com/kubeshop/testkube).
