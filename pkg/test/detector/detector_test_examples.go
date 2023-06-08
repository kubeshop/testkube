package detector

const (
	examplePostmanContent = `{ "info": { "_postman_id": "3d9a6be2-bd3e-4cf7-89ca-354103aab4a7", "name": "Testkube", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" }, "item": [ { "name": "Health", "event": [ { "listen": "test", "script": { "exec": [ "pm.test(\"Status code is 200\", function () {", "    pm.response.to.have.status(200);", "});" ], "type": "text/javascript" } } ], "request": { "method": "GET", "header": [], "url": { "raw": "{{URI}}/health", "host": [ "{{URI}}" ], "path": [ "health" ] } }, "response": [] } ] } `

	exampleArtilleryFilename = "artillery.yaml"
	exampleArtilleryContent  = `config:
	target: "https://testkube.kubeshop.io/"
	phases:
	- duration: 1
	  arrivalRate: 1
  scenarios:
  - flow:
	- get:
		url: "/"
	- think: 0.1`

	exampleCurlFilename = "curl.json"
	exampleCurlContent  = `{
		"command": [
		  "curl",
		  "https://reqbin.com/echo/get/json",
		  "-H",
		  "'Accept: application/json'"
		],
		"expected_status": "200",
		"expected_body": "{\"success\":\"true\"}"
	  }`

	exampleJMeterFilename = "jmeter.jmx"
	exampleJMeterContent  = `<?xml version="1.0" encoding="UTF-8"?>
	<jmeterTestPlan version="1.2" properties="2.8" jmeter="2.13.19691222">
	  <hashTree>
		<TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true">
		  <stringProp name="TestPlan.comments"></stringProp>
		  <boolProp name="TestPlan.functional_mode">false</boolProp>
		  <boolProp name="TestPlan.serialize_threadgroups">false</boolProp>
		  <elementProp name="TestPlan.user_defined_variables" elementType="Arguments" guiclass="ArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
			<collectionProp name="Arguments.arguments"/>
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
			<longProp name="ThreadGroup.start_time">1668426657000</longProp>
			<longProp name="ThreadGroup.end_time">1668426657000</longProp>
			<boolProp name="ThreadGroup.scheduler">false</boolProp>
			<stringProp name="ThreadGroup.duration"></stringProp>
			<stringProp name="ThreadGroup.delay"></stringProp>
		  </ThreadGroup>
		  <hashTree>
			<HTTPSamplerProxy guiclass="HttpTestSampleGui" testclass="HTTPSamplerProxy" testname="Testkube - HTTP Request" enabled="true">
			  <elementProp name="HTTPsampler.Arguments" elementType="Arguments" guiclass="HTTPArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
				<collectionProp name="Arguments.arguments"/>
			  </elementProp>
			  <stringProp name="HTTPSampler.domain">testkube.kubeshop.io</stringProp>
			  <stringProp name="HTTPSampler.port"></stringProp>
			  <stringProp name="HTTPSampler.connect_timeout"></stringProp>
			  <stringProp name="HTTPSampler.response_timeout"></stringProp>
			  <stringProp name="HTTPSampler.protocol"></stringProp>
			  <stringProp name="HTTPSampler.contentEncoding"></stringProp>
			  <stringProp name="HTTPSampler.path"></stringProp>
			  <stringProp name="HTTPSampler.method">GET</stringProp>
			  <boolProp name="HTTPSampler.follow_redirects">true</boolProp>
			  <boolProp name="HTTPSampler.auto_redirects">false</boolProp>
			  <boolProp name="HTTPSampler.use_keepalive">true</boolProp>
			  <boolProp name="HTTPSampler.DO_MULTIPART_POST">false</boolProp>
			  <boolProp name="HTTPSampler.monitor">false</boolProp>
			  <stringProp name="HTTPSampler.embedded_url_re"></stringProp>
			</HTTPSamplerProxy>
			<hashTree/>
			<ResponseAssertion guiclass="AssertionGui" testclass="ResponseAssertion" testname="Response Assertion" enabled="true">
			  <collectionProp name="Asserion.test_strings">
				<stringProp name="49586">200</stringProp>
			  </collectionProp>
			  <stringProp name="Assertion.test_field">Assertion.response_code</stringProp>
			  <boolProp name="Assertion.assume_success">false</boolProp>
			  <intProp name="Assertion.test_type">8</intProp>
			</ResponseAssertion>
			<hashTree/>
			<ResultCollector guiclass="AssertionVisualizer" testclass="ResultCollector" testname="Assertion Results" enabled="true">
			  <boolProp name="ResultCollector.error_logging">false</boolProp>
			  <objProp>
				<name>saveConfig</name>
				<value class="SampleSaveConfiguration">
				  <time>true</time>
				  <latency>true</latency>
				  <timestamp>true</timestamp>
				  <success>true</success>
				  <label>true</label>
				  <code>true</code>
				  <message>true</message>
				  <threadName>true</threadName>
				  <dataType>true</dataType>
				  <encoding>false</encoding>
				  <assertions>true</assertions>
				  <subresults>true</subresults>
				  <responseData>false</responseData>
				  <samplerData>false</samplerData>
				  <xml>false</xml>
				  <fieldNames>false</fieldNames>
				  <responseHeaders>false</responseHeaders>
				  <requestHeaders>false</requestHeaders>
				  <responseDataOnError>false</responseDataOnError>
				  <saveAssertionResultsFailureMessage>false</saveAssertionResultsFailureMessage>
				  <assertionsResultsToSave>0</assertionsResultsToSave>
				  <bytes>true</bytes>
				  <threadCounts>true</threadCounts>
				</value>
			  </objProp>
			  <stringProp name="filename"></stringProp>
			</ResultCollector>
			<hashTree/>
		  </hashTree>
		</hashTree>
	  </hashTree>
	</jmeterTestPlan>`

	exampleK6Filename = "k6.js"
	exampleK6Content  = `import http from 'k6/http';
	import { check } from 'k6';
	
	if (__ENV.K6_ENV_FROM_PARAM != "K6_ENV_FROM_PARAM_value") {
	  throw new Error("Incorrect K6_ENV_FROM_PARAM ENV value");
	}
	
	if (__ENV.K6_SYSTEM_ENV != "K6_SYSTEM_ENV_value") {
	  throw new Error("Incorrect K6_SYSTEM_ENV ENV value");
	}
	
	export default function () {
	  http.get('https://testkube.kubeshop.io/');
	}`

	exampleSoapUIFilename = "soapui-project.xml"
	exampleSoapUIContent  = `<?xml version="1.0" encoding="UTF-8"?>
	<con:soapui-project id="a4d0c33f-8a17-4764-8657-7b1491450f1d" activeEnvironment="Default" name="soapui-smoke-test" resourceRoot="" soapui-version="5.7.0" xmlns:con="http://eviware.com/soapui/config"><con:settings/><con:interface xsi:type="con:RestService" id="3034130b-11fd-4fff-b8f6-ad001d746b11" wadlVersion="http://wadl.dev.java.net/2009/02" name="https://testkube.kubeshop.io" type="rest" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><con:settings/><con:definitionCache/><con:endpoints><con:endpoint>https://testkube.kubeshop.io</con:endpoint></con:endpoints><con:resource name="" path="/" id="9b7558bb-3f97-4eed-b6f9-784bf4197177"><con:settings/><con:parameters/><con:method name="1" id="1b46195c-80e1-43ee-9e11-406d5374b0e8" method="GET"><con:settings/><con:parameters/><con:request name="testkube-website" id="1a3b73f2-1eb1-47ec-a634-b865d11e1e2d" mediaType="application/json"><con:settings/><con:endpoint>https://testkube.kubeshop.io</con:endpoint><con:parameters/></con:request></con:method></con:resource></con:interface><con:properties/><con:wssContainer/><con:oAuth2ProfileContainer/><con:oAuth1ProfileContainer/></con:soapui-project>`
)
