package data

//
//type GetTestResultRequest struct {
//	ID string `json:"id"`
//}
//
//type GetTestResultResponse struct {
//	Execution testkube.TestSuiteExecution `json:"execution"`
//}
//
//type GetByNameAndTestSuiteTestResultRequest struct {
//	Name          string `json:"name"`
//	TestSuiteName string `json:"testSuiteName"`
//}
//
//type GetByNameAndTestSuiteTestResultResponse struct {
//	Execution testkube.TestSuiteExecution `json:"execution"`
//}
//
//type GetLatestByTestSuiteTestResultRequest struct {
//	TestSuiteName string `json:"testSuiteName"`
//	SortField     string `json:"sortField"`
//}
//
//type GetLatestByTestSuiteTestResultResponse struct {
//	Execution testkube.TestSuiteExecution `json:"execution"`
//}
//
//type GetLatestByTestSuitesTestResultRequest struct {
//	TestSuiteNames []string `json:"testSuiteNames"`
//	SortField      string   `json:"sortField"`
//}
//
//type GetLatestByTestSuitesTestResultResponse struct {
//	Executions []testkube.TestSuiteExecution `json:"executions"`
//}
//
//type GetExecutionsTotalsTestResultRequest struct {
//	Filter []testresult.Filter `json:"filter"`
//}
//
//type GetExecutionsTotalsTestResultResponse struct {
//	Totals testkube.ExecutionsTotals `json:"totals"`
//}
//
//type GetExecutionsTestResultRequest struct {
//	Filter testresult.Filter `json:"filter"`
//}
//
//type GetExecutionsTestResultResponse struct {
//	Executions []testkube.TestSuiteExecution `json:"executions"`
//}
//
//type InsertTestResultRequest struct {
//	Result testkube.TestSuiteExecution `json:"result"`
//}
//
//type InsertTestResultResponse struct {
//}
//
//type UpdateTestResultRequest struct {
//	Result testkube.TestSuiteExecution `json:"result"`
//}
//
//type UpdateTestResultResponse struct{}
//
//type StartExecutionTestResultRequest struct {
//	ID        string    `json:"id"`
//	StartTime time.Time `json:"startTime"`
//}
//
//type StartExecutionTestResultResponse struct{}
//
//type EndExecutionTestResultRequest struct {
//	Execution testkube.TestSuiteExecution `json:"execution"`
//}
//
//type EndExecutionTestResultResponse struct{}
//
//type DeleteByTestSuiteTestResultRequest struct {
//	TestSuiteName string `json:"testSuiteName"`
//}
//
//type DeleteByTestSuiteTestResultResponse struct{}
//
//type DeleteAllTestResultResponse struct{}
//
//type DeleteByTestSuitesTestResultRequest struct {
//	TestSuiteNames []string `json:"testSuiteNames"`
//}
//
//type DeleteByTestSuitesTestResultResponse struct{}
//
//type GetTestSuiteMetricsTestResultRequest struct {
//	Name  string `json:"name"`
//	Limit int    `json:"limit"`
//	Last  int    `json:"last"`
//}
//
//type GetTestSuiteMetricsTestResultResponse struct {
//	Metrics testkube.ExecutionsMetrics `json:"metrics"`
//}
