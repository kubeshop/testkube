package runner

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// ArtilleryExecutionResult ...
type ArtilleryExecutionResult struct {
	Output string
	Result ArtilleryTestResult
}

// Mapping ...
type Mapping struct {
	RelativeAccuracy float64 `json:"relativeAccuracy"`
	Offset           float64 `json:"_offset"`
	Gamma            float64 `json:"gamma"`
	Multiplier       float64 `json:"_multiplier"`
	MinPossible      float64 `json:"minPossible"`
	MaxPossible      float64 `json:"maxPossible"`
}

// Store ...
type Store struct {
	ChunkSize float64   `json:"chunkSize"`
	Bins      []float64 `json:"bins"`
	Count     float64   `json:"count"`
	MinKey    float64   `json:"minKey"`
	MaxKey    float64   `json:"maxKey"`
	Offset    float64   `json:"offset"`
}

// HistogramMetrics ...
type HistogramMetrics struct {
	Mapping       Mapping `json:"mapping"`
	Store         Store   `json:"store"`
	NegativeStore Store   `json:"negativeStore"`
	ZeroCount     int     `json:"zeroCount"`
	Count         int     `json:"count"`
	Min           float64 `json:"min"`
	Max           float64 `json:"max"`
	Sum           float64 `json:"sum"`
}

// Summary ...
type Summary struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Count  float64 `json:"count"`
	P50    float64 `json:"p50"`
	Median float64 `json:"median"`
	P75    float64 `json:"p75"`
	P90    float64 `json:"p90"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	P999   float64 `json:"p999"`
}

type Counters struct {
	VusersCreated                               int `json:"vusers.created"`
	HTTPRequests                                int `json:"http.requests"`
	HTTPCodes200                                int `json:"http.codes.200"`
	HTTPResponses                               int `json:"http.responses"`
	PluginsExpectOk                             int `json:"plugins.expect.ok"`
	PluginsExpectOkStatusCode                   int `json:"plugins.expect.ok.statusCode"`
	PluginsExpectOkContentType                  int `json:"plugins.expect.ok.contentType"`
	PluginsMetricsByEndpointGetRequestCodes200  int `json:"plugins.metrics-by-endpoint.getRequest.codes.200"`
	VusersFailed                                int `json:"vusers.failed"`
	VusersCompleted                             int `json:"vusers.completed"`
	PluginsMetricsByEndpointPostRequestCodes200 int `json:"plugins.metrics-by-endpoint.postRequest.codes.200"`
}
type Histograms struct {
	HTTPResponseTime                                HistogramMetrics `json:"http.response_time"`
	PluginsMetricsByEndpointResponseTimeGetRequest  HistogramMetrics `json:"plugins.metrics-by-endpoint.response_time.getRequest"`
	VusersSessionLength                             HistogramMetrics `json:"vusers.session_length"`
	PluginsMetricsByEndpointResponseTimePostRequest HistogramMetrics `json:"plugins.metrics-by-endpoint.response_time.postRequest"`
}
type Summaries struct {
	HTTPResponseTime                                Summary `json:"http.response_time"`
	PluginsMetricsByEndpointResponseTimeGetRequest  Summary `json:"plugins.metrics-by-endpoint.response_time.getRequest"`
	VusersSessionLength                             Summary `json:"vusers.session_length"`
	PluginsMetricsByEndpointResponseTimePostRequest Summary `json:"plugins.metrics-by-endpoint.response_time.postRequest"`
}
type Metrics struct {
	Counters         Counters   `json:"counters"`
	Histograms       Histograms `json:"histograms"`
	HTTPRequestRate  float64    `json:"http.request_rate"`
	FirstCounterAt   float64    `json:"firstCounterAt"`
	FirstHistogramAt float64    `json:"firstHistogramAt"`
	LastCounterAt    float64    `json:"lastCounterAt"`
	LastHistogramAt  float64    `json:"lastHistogramAt"`
	FirstMetricAt    float64    `json:"firstMetricAt"`
	LastMetricAt     float64    `json:"lastMetricAt"`
	Summaries        Summaries  `json:"summaries"`
}

// ArtilleryTestResult ...
type ArtilleryTestResult struct {
	Aggregate    Metrics   `json:"aggregate"`
	Intermediate []Metrics `json:"intermediate"`
}

// Validate checks if Execution has valid data in context of Artillery executor
func (r *ArtilleryRunner) Validate(execution testkube.Execution) error {

	if execution.Content == nil {
		return errors.Errorf("can't find any content to run in execution data: %+v", execution)
	}

	return nil
}

// GetArtilleryExecutionResult - fetch results from output report file
func (r *ArtilleryRunner) GetArtilleryExecutionResult(testReportFile string, out []byte) (ArtilleryExecutionResult, error) {
	result := ArtilleryExecutionResult{}
	result.Output = string(out)
	data, err := os.ReadFile(testReportFile)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result.Result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// MapTestSummaryToResults - map test results open-api format
func MapTestSummaryToResults(artilleryResult ArtilleryExecutionResult) testkube.ExecutionResult {

	status := testkube.StatusPtr(testkube.PASSED_ExecutionStatus)
	if artilleryResult.Result.Aggregate.Counters.VusersFailed > 0 {
		status = testkube.StatusPtr(testkube.FAILED_ExecutionStatus)

	}
	result := testkube.ExecutionResult{
		Output:     artilleryResult.Output,
		OutputType: "text/plain",
		Status:     status,
	}
	return result

}

func makeSuccessExecution(out []byte) (result testkube.ExecutionResult) {
	status := testkube.PASSED_ExecutionStatus
	result.Status = &status
	result.Output = string(out)
	result.OutputType = "text/plain"

	return result
}
