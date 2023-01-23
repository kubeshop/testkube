package data

//type CloudTestResultRepository struct {
//	apiURL     string
//	httpClient *http.Client
//}
//
//
//func (r *CloudTestResultRepository) Get(ctx context.Context, id string) (testkube.TestSuiteExecution, error) {
//	var result testkube.TestSuiteExecution
//	payload := map[string]interface{}{"id": id}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultGet, Payload: payload})
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
//
//func (r *CloudTestResultRepository) GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (testkube.TestSuiteExecution, error) {
//	var result testkube.TestSuiteExecution
//	payload := map[string]interface{}{"name": name, "test_suite_name": testSuiteName}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultGetByNameAndTestSuite, Payload: payload})
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
//
//func (r *CloudTestResultRepository) GetLatestByTestSuite(ctx context.Context, testSuiteName, sortField string) (testkube.TestSuiteExecution, error) {
//	var result testkube.TestSuiteExecution
//	payload := map[string]interface{}{"test_suite_name": testSuiteName, "sort_field": sortField}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultGetLatestByTestSuite, Payload: payload})
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
//
//func (r *CloudTestResultRepository) GetLatestByTestSuites(ctx context.Context, testSuiteNames []string, sortField string) ([]testkube.TestSuiteExecution, error) {
//	var result []testkube.TestSuiteExecution
//	payload := map[string]interface{}{"test_suite_names": testSuiteNames, "sort_field": sortField}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultGetLatestByTestSuites, Payload: payload})
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
//
//func (r *CloudTestResultRepository) GetExecutionsTotals(ctx context.Context, filter ...common.Filter) (testkube.ExecutionsTotals, error) {
//	var result testkube.ExecutionsTotals
//	payload := map[string]interface{}{"filter": filter}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultGetExecutionsTotals, Payload: payload})
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
//
//func (r *CloudTestResultRepository) GetExecutions(ctx context.Context, filter Filter) ([]testkube.TestSuiteExecution, error) {
//	var result []testkube.TestSuiteExecution
//	payload := map[string]interface{}{"filter": filter}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultGetExecutions, Payload: payload})
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
//
//func (r *CloudTestResultRepository) Insert(ctx context.Context, result testkube.TestSuiteExecution) error {
//	payload := map[string]interface{}{"result": result}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultInsert, Payload: payload})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r *CloudTestResultRepository) Update(ctx context.Context, result testkube.TestSuiteExecution) error {
//	payload := map[string]interface{}{"result": result}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultUpdate, Payload: payload})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r *CloudTestResultRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
//	payload := map[string]interface{}{"id": id, "start_time": startTime}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultStartExecution, Payload: payload})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r *CloudTestResultRepository) EndExecution(ctx context.Context, execution testkube.TestSuiteExecution) error {
//	payload := map[string]interface{}{"execution": execution}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultEndExecution, Payload: payload})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r *CloudTestResultRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
//	payload := map[string]interface{}{"test_suite_name": testSuiteName}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultDeleteByTestSuite, Payload: payload})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r *CloudTestResultRepository) DeleteAll(ctx context.Context) error {
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultDeleteAll, Payload: nil})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r *CloudTestResultRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
//	payload := map[string]interface{}{"test_suite_names": testSuiteNames}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultDeleteByTestSuites, Payload: payload})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r *CloudTestResultRepository) GetTestSuiteMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
//	var result testkube.ExecutionsMetrics
//	payload := map[string]interface{}{"name": name, "limit": limit, "last": last}
//	err := executeHTTP(ctx, r.httpClient, r.apiURL, CommandRequest{Command: CmdTestResultGetTestSuiteMetrics, Payload: payload})
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
