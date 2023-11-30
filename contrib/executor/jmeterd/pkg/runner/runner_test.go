package runner

import (
	"context"
	"github.com/kubeshop/testkube/pkg/utils/test"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
)

func TestReplaceArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		path           string
		jtlPath        string
		reportPath     string
		jmeterLogPath  string
		expectedArgs   []string
		expectedJunit  bool
		expectedReport bool
	}{
		{
			name:           "All Placeholders",
			args:           []string{"<runPath>", "<jtlFile>", "<reportFile>", "<logFile>", "-l"},
			path:           "path",
			jtlPath:        "jtlPath",
			reportPath:     "reportPath",
			jmeterLogPath:  "jmeterLogPath",
			expectedArgs:   []string{"path", "jtlPath", "reportPath", "jmeterLogPath", "-l"},
			expectedJunit:  true,
			expectedReport: true,
		},
		{
			name:           "No Placeholders",
			args:           []string{"arg1", "arg2"},
			path:           "path",
			jtlPath:        "jtlPath",
			reportPath:     "reportPath",
			jmeterLogPath:  "jmeterLogPath",
			expectedArgs:   []string{"arg1", "arg2"},
			expectedJunit:  false,
			expectedReport: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hasJunit, hasReport := replacePlaceholderArgs(tt.args, tt.path, tt.jtlPath, tt.reportPath, tt.jmeterLogPath)

			for i, arg := range tt.args {
				assert.Equal(t, tt.expectedArgs[i], arg)
			}
			assert.Equal(t, tt.expectedJunit, hasJunit)
			assert.Equal(t, tt.expectedReport, hasReport)
		})
	}
}

func TestJMeterDRunner_Local_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	type testCase struct {
		name          string
		tempDirPrefix string
		jmxContent    string
		wantStatus    *testkube.ExecutionStatus
	}

	tests := []testCase{
		{
			name:          "error in test file",
			tempDirPrefix: "testkube-error-jmx-*",
			jmxContent:    errorJMX,
			wantStatus:    testkube.ExecutionStatusFailed,
		},
		{
			name:          "success",
			tempDirPrefix: "testkube-success-jmx-*",
			jmxContent:    successJMX,
			wantStatus:    testkube.ExecutionStatusPassed,
		},
		{
			name:          "failure",
			tempDirPrefix: "testkube-failure-jmx-*",
			jmxContent:    failureJMX,
			wantStatus:    testkube.ExecutionStatusFailed,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			// init test folder
			tempDir, err := os.MkdirTemp("", tt.tempDirPrefix)
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			writeTestContent(t, tempDir, tt.jmxContent)

			params := envs.Params{DataDir: tempDir}

			r, err := NewRunner(ctx, params)
			if err != nil {
				t.Fatalf("error creating %s executor: %v", r.GetType(), err)
			}

			execution := testkube.Execution{
				Id:            "test-id",
				TestName:      "test1",
				TestNamespace: "testkube",
				Name:          "test1",
				Command:       []string{"jmeter"},
				Args:          []string{"-n", "-j", "<logFile>", "-t", "<runPath>", "-l", "<jtlFile>", "-e", "-o", "<reportFile>", "<envVars>"},
				Content: &testkube.TestContent{
					Type_: string(testkube.TestContentTypeString),
					Data:  tt.jmxContent,
				},
			}

			result, err := r.Run(ctx, execution)
			if err != nil {
				t.Fatalf("error running %s executor: %v", r.GetType(), err)
			}

			assert.Equal(t, tt.wantStatus, result.Status)
		})
	}
}

func writeTestContent(t *testing.T, testDir, content string) {
	t.Helper()
	path := filepath.Join(testDir, "test-content")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}

const errorJMX = `<?xml version="1.0" encoding="UTF-8"?>
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
      <Arguments guiclass="ArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
        <collectionProp name="Arguments.arguments">
          <elementProp name="UC1_NBUSERS" elementType="Argument">
            <stringProp name="Argument.name">UC1_NBUSERS</stringProp>
            <stringProp name="Argument.value">${__property(JMETER_UC1_NBUSERS,,2)}</stringProp>
            <stringProp name="Argument.metadata">=</stringProp>
          </elementProp>
          <elementProp name="UC1_RAMPUP" elementType="Argument">
            <stringProp name="Argument.name">UC1_RAMPUP</stringProp>
            <stringProp name="Argument.value">${__property(JMETER_UC1_RAMPUP,,2)}</stringProp>
            <stringProp name="Argument.metadata">=</stringProp>
          </elementProp>
          <elementProp name="URI_PATH" elementType="Argument">
            <stringProp name="Argument.name">URI_PATH</stringProp>
            <stringProp name="Argument.value">${__property(JMETER_URI_PATH,,/pricing)}</stringProp>
            <stringProp name="Argument.metadata">=</stringProp>
          </elementProp>
          <elementProp name="PROJECT_HOME" elementType="Argument">
            <stringProp name="Argument.name">PROJECT_HOME</stringProp>
            <stringProp name="Argument.value">${__BeanShell(import org.apache.jmeter.services.FileServer; FileServer.getFileServer().getBaseDir();)}</stringProp>
            <stringProp name="Argument.metadata">=</stringProp>
          </elementProp>
        </collectionProp>
      </Arguments>
      <hashTree/>
      <CSVDataSet guiclass="TestBeanGUI" testclass="CSVDataSet" testname="CSV Data Set Config" enabled="true">
        <stringProp name="delimiter">,</stringProp>
        <stringProp name="fileEncoding"></stringProp>
        <stringProp name="filename">${__env(DATA_CONFIG,,${__eval(${PROJECT_HOME})})}/csvdata/Credentials.csv</stringProp>
        <boolProp name="ignoreFirstLine">false</boolProp>
        <boolProp name="quotedData">false</boolProp>
        <boolProp name="recycle">true</boolProp>
        <stringProp name="shareMode">shareMode.group</stringProp>
        <boolProp name="stopThread">false</boolProp>
        <stringProp name="variableNames">USERNAME,PASSWORD</stringProp>
      </CSVDataSet>
      <hashTree/>
      <ThreadGroup guiclass="ThreadGroupGui" testclass="ThreadGroup" testname="Thread Group" enabled="true">
        <stringProp name="ThreadGroup.on_sample_error">continue</stringProp>
        <elementProp name="ThreadGroup.main_controller" elementType="LoopController" guiclass="LoopControlPanel" testclass="LoopController" testname="Loop Controller" enabled="true">
          <boolProp name="LoopController.continue_forever">false</boolProp>
          <stringProp name="LoopController.loops">1</stringProp>
        </elementProp>
        <stringProp name="ThreadGroup.num_threads">${UC1_NBUSERS}</stringProp>
        <stringProp name="ThreadGroup.ramp_time">${UC1_RAMPUP}</stringProp>
        <boolProp name="ThreadGroup.scheduler">false</boolProp>
        <stringProp name="ThreadGroup.duration"></stringProp>
        <stringProp name="ThreadGroup.delay"></stringProp>
        <boolProp name="ThreadGroup.same_user_on_next_iteration">true</boolProp>
      </ThreadGroup>
      <hashTree>
        <TransactionController guiclass="TransactionControllerGui" testclass="TransactionController" testname="01_UC1-TEST" enabled="true">
          <boolProp name="TransactionController.includeTimers">false</boolProp>
          <boolProp name="TransactionController.parent">false</boolProp>
        </TransactionController>
        <hashTree>
          <BeanShellSampler guiclass="BeanShellSamplerGui" testclass="BeanShellSampler" testname="BeanShell Sampler" enabled="true">
            <stringProp name="BeanShellSampler.query">String uc1_nbusers = vars.get(&quot;UC1_NBUSERS&quot;);
String uc1_rampup = vars.get(&quot;UC1_RAMPUP&quot;);
String uri_path = vars.get(&quot;URI_PATH&quot;);
String username = vars.get(&quot;USERNAME&quot;);
String password = vars.get(&quot;PASSWORD&quot;);
log.info(&quot;=================================&quot;);
log.info(&quot;UC1_NBUSERS: &quot; + uc1_nbusers);
log.info(&quot;UC1_RAMPUP: &quot; + uc1_rampup);
log.info(&quot;URI_PATH: &quot; + uri_path);
log.info(&quot;USERNAME: &quot; + username);
log.info(&quot;PASSWORD: &quot; + password);
log.info(&quot;=================================&quot;);</stringProp>
            <stringProp name="BeanShellSampler.filename"></stringProp>
            <stringProp name="BeanShellSampler.parameters"></stringProp>
            <boolProp name="BeanShellSampler.resetInterpreter">false</boolProp>
          </BeanShellSampler>
          <hashTree/>
          <com.blazemeter.jmeter.controller.ParallelSampler guiclass="com.blazemeter.jmeter.controller.ParallelControllerGui" testclass="com.blazemeter.jmeter.controller.ParallelSampler" testname="02_UC1-PARALLEL" enabled="true">
            <intProp name="MAX_THREAD_NUMBER">6</intProp>
            <boolProp name="PARENT_SAMPLE">true</boolProp>
            <boolProp name="LIMIT_MAX_THREAD_NUMBER">false</boolProp>
          </com.blazemeter.jmeter.controller.ParallelSampler>
          <hashTree>
            <HTTPSamplerProxy guiclass="HttpTestSampleGui" testclass="HTTPSamplerProxy" testname="HTTP Request" enabled="true">
              <elementProp name="HTTPsampler.Arguments" elementType="Arguments" guiclass="HTTPArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
                <collectionProp name="Arguments.arguments"/>
              </elementProp>
              <stringProp name="HTTPSampler.domain">testkube.io</stringProp>
              <stringProp name="HTTPSampler.port"></stringProp>
              <stringProp name="HTTPSampler.protocol">https</stringProp>
              <stringProp name="HTTPSampler.contentEncoding"></stringProp>
              <stringProp name="HTTPSampler.path">${PATH}</stringProp>
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
                  <stringProp name="1620359203">SOME_NONExisting_String</stringProp>
                </collectionProp>
                <stringProp name="Assertion.custom_message"></stringProp>
                <stringProp name="Assertion.test_field">Assertion.response_data</stringProp>
                <boolProp name="Assertion.assume_success">false</boolProp>
                <intProp name="Assertion.test_type">20</intProp>
              </ResponseAssertion>
              <hashTree/>
            </hashTree>
            <HTTPSamplerProxy guiclass="HttpTestSampleGui" testclass="HTTPSamplerProxy" testname="HTTP Request" enabled="true">
              <elementProp name="HTTPsampler.Arguments" elementType="Arguments" guiclass="HTTPArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
                <collectionProp name="Arguments.arguments"/>
              </elementProp>
              <stringProp name="HTTPSampler.domain">testkube.io</stringProp>
              <stringProp name="HTTPSampler.port"></stringProp>
              <stringProp name="HTTPSampler.protocol">https</stringProp>
              <stringProp name="HTTPSampler.contentEncoding"></stringProp>
              <stringProp name="HTTPSampler.path">${PATH}</stringProp>
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
                  <stringProp name="1620359203">SOME_NONExisting_String</stringProp>
                </collectionProp>
                <stringProp name="Assertion.custom_message"></stringProp>
                <stringProp name="Assertion.test_field">Assertion.response_data</stringProp>
                <boolProp name="Assertion.assume_success">false</boolProp>
                <intProp name="Assertion.test_type">20</intProp>
              </ResponseAssertion>
              <hashTree/>
            </hashTree>
          </hashTree>
        </hashTree>
        <ResultCollector guiclass="ViewResultsFullVisualizer" testclass="ResultCollector" testname="View Results Tree" enabled="true">
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
              <fieldNames>true</fieldNames>
              <responseHeaders>false</responseHeaders>
              <requestHeaders>false</requestHeaders>
              <responseDataOnError>false</responseDataOnError>
              <saveAssertionResultsFailureMessage>true</saveAssertionResultsFailureMessage>
              <assertionsResultsToSave>2</assertionsResultsToSave>
              <bytes>true</bytes>
              <sentBytes>true</sentBytes>
              <url>true</url>
              <threadCounts>true</threadCounts>
              <idleTime>true</idleTime>
              <connectTime>true</connectTime>
            </value>
          </objProp>
          <stringProp name="filename"></stringProp>
        </ResultCollector>
        <hashTree/>
      </hashTree>
    </hashTree>
  </hashTree>
</jmeterTestPlan>
`

const failureJMX = `<?xml version="1.0" encoding="UTF-8"?>
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
              <stringProp name="-1081444641">SOME_NONExisting_String</stringProp>
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
`

const successJMX = `<?xml version="1.0" encoding="UTF-8"?>
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
`
