package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	badXML = `
12345
`

	successXML = `
<testResults version="1.2">
	<httpSample t="1259" it="0" lt="124" ct="80" ts="1690288938130" s="true" lb="Testkube - HTTP Request" rc="200" rm="OK" tn="Thread Group 1-1" dt="text" by="65222" sby="366" ng="1" na="1">
		<assertionResult>
			<name>Response Assertion</name>
			<failure>false</failure>
			<error>false</error>
		</assertionResult>
	</httpSample>
</testResults>
`

	failedXML = `
<testResults version="1.2">
	<httpSample t="51" it="0" lt="0" ct="51" ts="1690366471701" s="false" lb="Testkube - HTTP Request" rc="Non HTTP response code: java.net.UnknownHostException" rm="Non HTTP response message: testkube.fakeshop.io: Name does not resolve" tn="Thread Group 1-1" dt="text" by="2327" sby="0" ng="1" na="1">
		<assertionResult>
			<name>Response Assertion</name>
			<failure>true</failure>
			<error>false</error>
			<failureMessage>Test failed: code expected to equal / ****** received : [[[Non HTTP response code: java.net.UnknownHostException]]] ****** comparison: [[[200 ]]] /</failureMessage>
		</assertionResult>
	</httpSample>
</testResults>
`

	mixedXML = `
<testResults version="1.2">
	<sample t="2" it="0" lt="0" ct="0" ts="1690724003768" s="true" lb="========== Starting GR Alarm verification =========" rc="200" rm="OK" tn="Verify GR Alarms 1-1" dt="text" by="7109" sby="0" ng="1" na="1">
	</sample>
	<httpSample t="946" it="0" lt="935" ct="737" ts="1690724004825" s="true" lb="Get Token - HTTP Request" rc="200" rm="OK" tn="Verify GR Alarms 1-1" dt="text" by="3774" sby="350" ng="1" na="1">
	</httpSample>	
	<sample t="0" it="0" lt="0" ct="0" ts="1690724048711" s="true" lb="Dummy cmdb Alarm cleared at : 2022-03-22T09:05:01.000Z " rc="200" rm="OK" tn="Verify GR Alarms 1-1" dt="text" by="10328" sby="0" ng="1" na="1">
	</sample>	
	<sample t="1" it="0" lt="0" ct="0" ts="1690724048871" s="false" lb="Alarms status are inactive. Unexpected Result! Failing the test!" rc="200" rm="OK" tn="Verify GR Alarms 1-1" dt="text" by="10328" sby="0" ng="1" na="1">
		<assertionResult>
		<name>Fail Test</name>
		<failure>true</failure>
		<error>false</error>
		<failureMessage>Test FAILED</failureMessage>
		</assertionResult>
	</sample>		
</testResults>
`
)

func TestParseXML(t *testing.T) {
	t.Parallel()

	t.Run("parse XML success test", func(t *testing.T) {
		t.Parallel()

		results, err := ParseXML([]byte(successXML))

		assert.NoError(t, err)
		assert.Equal(t, 1, len(results.HTTPSamples))
		assert.True(t, results.HTTPSamples[0].Success)
		assert.Equal(t, 1259, results.HTTPSamples[0].Time)
		assert.Equal(t, "Testkube - HTTP Request", results.HTTPSamples[0].Label)
		assert.Equal(t, "Response Assertion", results.HTTPSamples[0].AssertionResult.Name)
	})

	t.Run("parse XML failed test", func(t *testing.T) {
		t.Parallel()

		results, err := ParseXML([]byte(failedXML))

		assert.NoError(t, err)
		assert.Equal(t, 1, len(results.HTTPSamples))
		assert.False(t, results.HTTPSamples[0].Success)
		assert.Equal(t, 51, results.HTTPSamples[0].Time)
		assert.Equal(t, "Testkube - HTTP Request", results.HTTPSamples[0].Label)
		assert.Equal(t, "Test failed: code expected to equal / ****** received : [[[Non HTTP response code: java.net.UnknownHostException]]] ****** comparison: [[[200 ]]] /", results.HTTPSamples[0].AssertionResult.FailureMessage)
	})

	t.Run("parse bad XML", func(t *testing.T) {
		t.Parallel()

		_, err := ParseXML([]byte(badXML))

		assert.EqualError(t, err, "EOF")
	})

	t.Run("parse XML mixed test", func(t *testing.T) {
		t.Parallel()

		results, err := ParseXML([]byte(mixedXML))

		assert.NoError(t, err)
		assert.Equal(t, 1, len(results.HTTPSamples))
		assert.True(t, results.HTTPSamples[0].Success)
		assert.Equal(t, 946, results.HTTPSamples[0].Time)
		assert.Equal(t, "Get Token - HTTP Request", results.HTTPSamples[0].Label)

		assert.Equal(t, 3, len(results.Samples))
		assert.False(t, results.Samples[2].Success)
		assert.Equal(t, 1, results.Samples[2].Time)
		assert.Equal(t, "Alarms status are inactive. Unexpected Result! Failing the test!", results.Samples[2].Label)
		assert.Equal(t, "Fail Test", results.Samples[2].AssertionResult.Name)
	})
}
