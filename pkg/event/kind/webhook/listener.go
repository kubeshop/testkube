package webhook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"text/template"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	thttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

var _ common.Listener = (*WebhookListener)(nil)

func NewWebhookListener(
	name, uri, selector string,
	events []testkube.EventType,
	payloadObjectField, payloadTemplate string,
	headers map[string]string,
	disabled bool,
	config map[string]executorv1.WebhookConfigValue,
	parameters []executorv1.WebhookParameterSchema,
	opts ...WebhookListenerOption,
) *WebhookListener {
	wl := &WebhookListener{
		name:               name,
		Uri:                uri,
		Log:                log.DefaultLogger,
		HttpClient:         thttp.NewClient(),
		selector:           selector,
		events:             events,
		payloadObjectField: payloadObjectField,
		payloadTemplate:    payloadTemplate,
		headers:            headers,
		disabled:           disabled,
		config:             config,
		parameters:         parameters,
	}

	for _, opt := range opts {
		opt(wl)
	}

	return wl
}

type WebhookListener struct {
	name string
	// TODO(emil): check if all these fields need to be exported
	Uri                string
	Log                *zap.SugaredLogger
	HttpClient         *http.Client
	selector           string
	events             []testkube.EventType
	payloadObjectField string
	payloadTemplate    string
	headers            map[string]string
	disabled           bool
	config             map[string]executorv1.WebhookConfigValue
	parameters         []executorv1.WebhookParameterSchema

	// Optional fields
	testWorkflowResultsRepository testworkflow.Repository
	webhookResultsRepository      cloudwebhook.WebhookRepository
	secretClient                  secret.Interface
	metrics                       v1.Metrics
	envs                          map[string]string
	dashboardURI                  string
	orgID                         string
	envID                         string
}

// WebhookListenerOption is a functional option for WebhookListener
type WebhookListenerOption func(*WebhookListener)

// listenerWithTestWorkflowResultsRepository configures the test workflow results repository for the webhook listener.
func listenerWithTestWorkflowResultsRepository(repo testworkflow.Repository) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.testWorkflowResultsRepository = repo
	}
}

// listenerWithWebhookResultsRepository sets the repository used for collecting webhook results
func listenerWithWebhookResultsRepository(repo cloudwebhook.WebhookRepository) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.webhookResultsRepository = repo
	}
}

// listenerWithSecretClient configures the secret client for the webhook listener.
func listenerWithSecretClient(secretClient secret.Interface) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.secretClient = secretClient
	}
}

// listenerWithMetrics configures the metrics for the webhook listener.
func listenerWithMetrics(metrics v1.Metrics) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.metrics = metrics
	}
}

// listenerWithEnvs sets the agent's environment variables to be used in templates.
func listenerWithEnvs(envs map[string]string) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.envs = envs
	}
}

// ListenerWithDashboardURI sets the dashboard URI for the connection to the
// control plane to be used in templates.
func ListenerWithDashboardURI(dashboardURI string) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.dashboardURI = dashboardURI
	}
}

// ListenerWithOrgID sets the organization ID for the connection to the
// control plane to be used in templates.
func ListenerWithOrgID(orgID string) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.orgID = orgID
	}
}

// ListenerWithEnvID sets the environment ID for the connection to the
// control plane to be used in templates.
func ListenerWithEnvID(envID string) WebhookListenerOption {
	return func(wl *WebhookListener) {
		wl.envID = envID
	}
}

func (l *WebhookListener) Name() string {
	return common.ListenerName(l.name)
}

func (l *WebhookListener) Selector() string {
	return l.selector
}

func (l *WebhookListener) Events() []testkube.EventType {
	return l.events
}

func (l *WebhookListener) Metadata() map[string]string {
	headers, err := getMapHashedMetadata(l.headers)
	if err != nil {
		l.Log.Errorw("headers hashing error", "error", err)
	}

	config, err := getMapHashedMetadata(l.config)
	if err != nil {
		l.Log.Errorw("config hashing error", "error", err)
	}

	parameters, err := getSliceHashedMetadata(l.parameters)
	if err != nil {
		l.Log.Errorw("parameters hashing error", "error", err)
	}

	return map[string]string{
		"name":               l.Name(),
		"uri":                l.Uri,
		"selector":           l.selector,
		"events":             fmt.Sprintf("%v", l.events),
		"payloadObjectField": l.payloadObjectField,
		"payloadTemplate":    getTextHashedMetadata([]byte(l.payloadTemplate)),
		"headers":            headers,
		"disabled":           fmt.Sprint(l.disabled),
		"config":             config,
		"parameters":         parameters,
	}
}

func (l *WebhookListener) PayloadObjectField() string {
	return l.payloadObjectField
}

func (l *WebhookListener) PayloadTemplate() string {
	return l.payloadTemplate
}

func (l *WebhookListener) Headers() map[string]string {
	return l.headers
}

func (l *WebhookListener) Disabled() bool {
	return l.disabled
}

func (l *WebhookListener) Match(event testkube.Event) bool {
	_, valid := event.Valid(l.Group(), l.Selector(), l.Events())
	if !valid {
		return false
	}
	// Handle disabled Webhooks
	switch {
	case l.disabled:
		l.Log.With(event.Log()...).Debug("webhook listener is disabled")
		return false
	case event.TestExecution != nil && event.TestExecution.DisableWebhooks:
		l.Log.With(event.Log()...).Debug("webhook listener is disabled for test execution")
		return false
	case event.TestSuiteExecution != nil && event.TestSuiteExecution.DisableWebhooks:
		l.Log.With(event.Log()...).Debug("webhook listener is disabled for test suite execution")
		return false
	case event.TestWorkflowExecution != nil && (event.TestWorkflowExecution.DisableWebhooks ||
		(event.TestWorkflowExecution.SilentMode != nil && event.TestWorkflowExecution.SilentMode.Webhooks)):
		l.Log.With(event.Log()...).Debug("webhook listener is disabled for test workflow execution")
		return false
	default:
		return true
	}
}

func (l *WebhookListener) Notify(event testkube.Event) (result testkube.EventResult) {
	var statusCode int
	var err error

	log := l.Log.With(event.Log()...)
	// load global envs to be able to use them in templates
	event.Envs = l.envs

	defer func() {
		// TODO(emil): using this deferred is a really strange/unreadable
		// pattern to process the results; simply wrap the inside logic to make
		// this more readable

		// Webhook metrics
		var eventType, res string
		if event.Type_ != nil {
			eventType = string(*event.Type_)
		}
		res = "success"
		if result.Error() != "" {
			res = "error"
		}
		l.metrics.IncWebhookEventCount(l.name, eventType, res)

		// Webhook telemetry
		if l.webhookResultsRepository == nil {
			return
		}
		errorMessage := ""
		if err != nil {
			errorMessage = err.Error()
		}
		if err = l.webhookResultsRepository.CollectExecutionResult(context.Background(), event, l.name, errorMessage, statusCode); err != nil {
			log.Errorw("webhook collecting execution result error", "error", err)
		}
	}()

	if event.Type_ != nil && event.Type_.IsBecome() {
		became, err := l.hasBecomeState(event)
		if err != nil {
			l.Log.With(event.Log()...).Errorw("could not get previous finished state", "error", err)
		}
		if !became {
			return testkube.NewSuccessEventResult(event.Id, "webhook is set to become state only; state has not become")
		}
	}

	body := bytes.NewBuffer([]byte{})

	var uri []byte
	uri, err = l.processTemplate("uri", l.Uri, event)
	if err != nil {
		log.Errorw("uri template processing error", "error", err)
		result = testkube.NewFailedEventResult(event.Id, err)
		return
	}

	if l.payloadTemplate != "" {
		var data []byte
		data, err = l.processTemplate("payload", l.payloadTemplate, event)
		if err != nil {
			log.Errorw("payload template processing error", "error", err)
			result = testkube.NewFailedEventResult(event.Id, err)
			return
		}

		_, err = body.Write(data)
	} else {
		// clean envs if not requested explicitly by payload template
		cleanEvent := event
		cleanEvent.Envs = nil
		err = json.NewEncoder(body).Encode(cleanEvent)
		if err == nil && l.payloadObjectField != "" {
			data := map[string]string{l.payloadObjectField: body.String()}
			body.Reset()
			err = json.NewEncoder(body).Encode(data)
		}
	}

	if err != nil {
		err = errors.Wrap(err, "webhook send encode error")
		log.Errorw("webhook send encode error", "error", err)
		result = testkube.NewFailedEventResult(event.Id, err)
		return
	}

	request, err := http.NewRequest(http.MethodPost, string(uri), body)
	if err != nil {
		log.Errorw("webhook request creating error", "error", err)
		result = testkube.NewFailedEventResult(event.Id, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")
	for key, value := range l.headers {
		values := []*string{&key, &value}
		for i := range values {
			data, err := l.processTemplate("header", *values[i], event)
			if err != nil {
				log.Errorw("header template processing error", "error", err)
				result = testkube.NewFailedEventResult(event.Id, err)
				return
			}

			*values[i] = string(data)
		}

		request.Header.Set(key, value)
	}

	var resp *http.Response
	resp, err = l.HttpClient.Do(request)
	if err != nil {
		log.Errorw("webhook send error", "error", err)
		result = testkube.NewFailedEventResult(event.Id, err)
		return
	}
	defer resp.Body.Close()

	var data []byte
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Errorw("webhook read response error", "error", err)
		result = testkube.NewFailedEventResult(event.Id, err)
		return
	}

	responseStr := string(data)
	statusCode = resp.StatusCode
	if resp.StatusCode >= 400 {
		err = fmt.Errorf("webhook response with bad status code: %d", resp.StatusCode)
		log.Errorw("webhook send error", "error", err, "status", resp.StatusCode, "response", responseStr)
		result = testkube.NewFailedEventResult(event.Id, err).WithResult(responseStr)
		return
	}

	log.Infow("webhook send result", "response", responseStr)
	result = testkube.NewSuccessEventResult(event.Id, responseStr)
	return
}

func (l *WebhookListener) Kind() string {
	return "webhook"
}

func (l *WebhookListener) Group() string {
	if l.envID != "" {
		return l.envID
	}
	return ""
}

func (l *WebhookListener) processTemplate(field, body string, event testkube.Event) ([]byte, error) {
	log := l.Log.With(event.Log()...)

	var tmpl *template.Template
	tmpl, err := utils.NewTemplate(field).Funcs(template.FuncMap{
		"tostr":                            text.ToStr,
		"executionstatustostring":          testkube.ExecutionStatusString,
		"testsuiteexecutionstatustostring": testkube.TestSuiteExecutionStatusString,
		"testworkflowstatustostring":       testkube.TestWorkflowStatusString,
	}).Parse(body)
	if err != nil {
		log.Errorw(fmt.Sprintf("creating webhook %s error", field), "error", err)
		return nil, err
	}

	config := make(map[string]string)
	for key, val := range l.config {
		var data string
		if val.Value != nil {
			data = *val.Value
		}

		if val.Secret != nil {
			if l.secretClient == nil {
				log.Errorw("secret references are unsupported in webhooks", "name", val.Secret.Name)
				return nil, errors.New("secret references are unsupported in webhooks")
			}
			var ns []string
			if val.Secret.Namespace != "" {
				ns = append(ns, val.Secret.Namespace)
			}

			elements, err := l.secretClient.Get(val.Secret.Name, ns...)
			if err != nil {
				log.Errorw("error secret loading", "error", err, "name", val.Secret.Name)
				return nil, err
			}

			if element, ok := elements[val.Secret.Key]; ok {
				data = element
			} else {
				log.Errorw("error secret key finding loading", "name", val.Secret.Name, "key", val.Secret.Key)
				return nil, errors.New("error secret key finding loading")
			}
		}

		config[key] = data
	}

	for _, parameter := range l.parameters {
		if _, ok := config[parameter.Name]; !ok {
			if parameter.Default_ != nil {
				config[parameter.Name] = *parameter.Default_
			} else if parameter.Required {
				log.Errorw("error missing required parameter", "name", parameter.Name)
				return nil, errors.New("error missing required parameter")
			}
		}

		if parameter.Pattern != "" {
			re, err := regexp.Compile(parameter.Pattern)
			if err != nil {
				log.Errorw("error compiling pattern", "error", err, "name", parameter.Name, "pattern", parameter.Pattern)
				return nil, err
			}

			if data, ok := config[parameter.Name]; ok && !re.MatchString(data) {
				log.Errorw("error matching pattern", "error", err, "name", parameter.Name, "pattern", parameter.Pattern)
				return nil, errors.New("error matching pattern")
			}
		}
	}

	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, field, NewTemplateVars(event, l.dashboardURI, l.orgID, l.envID, config)); err != nil {
		log.Errorw(fmt.Sprintf("executing webhook %s error", field), "error", err)
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (l *WebhookListener) hasBecomeState(event testkube.Event) (bool, error) {
	log := l.Log.With(event.Log()...)

	if event.TestExecution != nil && event.Type_ != nil {
		log.Warn("unable to determine become state, test execution results queries not supported")
		return false, nil
	}

	if event.TestSuiteExecution != nil && event.TestSuiteExecution.TestSuite != nil && event.Type_ != nil {
		log.Warn("unable to determine become state, testsuite execution results queries not supported")
		return false, nil
	}

	if event.TestWorkflowExecution != nil && event.TestWorkflowExecution.Workflow != nil && event.Type_ != nil {
		if l.testWorkflowResultsRepository == nil {
			log.Warn("unable to determine become state, testworkflow execution results queries not supported")
			return false, nil
		}
		prevStatus, err := l.testWorkflowResultsRepository.GetPreviousFinishedState(context.Background(), event.TestWorkflowExecution.Workflow.Name, event.TestWorkflowExecution.StatusAt)
		if err != nil {
			return false, err
		}

		if prevStatus == "" {
			log.Debugw(fmt.Sprintf("no previous finished state for test workflow %s", event.TestWorkflowExecution.Workflow.Name))
			return true, nil
		}

		return event.Type_.IsBecomeTestWorkflowExecutionStatus(prevStatus), nil
	}

	return false, nil
}

type configKeyValue[T any] struct {
	Key   string
	Value T
}

type configKeyValues[T any] []configKeyValue[T]

// getMapHashedMetadata returns map hashed metadata
func getMapHashedMetadata[T any](data map[string]T) (string, error) {
	var slice configKeyValues[T]
	for key, value := range data {
		slice = append(slice, configKeyValue[T]{Key: key, Value: value})
	}

	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Key < slice[j].Key
	})

	return getSliceHashedMetadata(slice)
}

// getSliceHashedMetadata returns slice hashed metadata
func getSliceHashedMetadata[T any](slice []T) (string, error) {
	result, err := json.Marshal(slice)
	if err != nil {
		return "", err
	}

	return getTextHashedMetadata(result), nil
}

// getTextHashedMetadata returns text hashed metadata
func getTextHashedMetadata(result []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(result))
}
