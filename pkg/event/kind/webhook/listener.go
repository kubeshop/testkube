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
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
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

func NewWebhookListener(name, uri, selector string, events []testkube.EventType,
	payloadObjectField, payloadTemplate string, headers map[string]string, disabled bool,
	// NOTE(emil): not going to be supported in control plane
	deprecatedRepositories commons.DeprecatedRepositories,
	// NOTE(emil): use to get the previous execution result - GetPreviousFinishedState, maybe don't support in first version
	testWorkflowExecutionResults testworkflow.Repository,
	// NOTE(emil): not going to be supported in control plane
	metrics v1.Metrics,
	webhookRepository cloudwebhook.WebhookRepository,
	// NOTE(emil): not going to be supported in control plane
	secretClient secret.Interface,
	// NOTE(emil): used to generate uris to the dashboard in the template rendering - essentially need the ui uri, org, and env ids
	proContext *config.ProContext,
	// NOTE(emil): not going to be supported in control plane
	envs map[string]string,
	config map[string]executorv1.WebhookConfigValue,
	parameters []executorv1.WebhookParameterSchema,
) *WebhookListener {
	return &WebhookListener{
		name:                         name,
		Uri:                          uri,
		Log:                          log.DefaultLogger,
		HttpClient:                   thttp.NewClient(),
		selector:                     selector,
		events:                       events,
		payloadObjectField:           payloadObjectField,
		payloadTemplate:              payloadTemplate,
		headers:                      headers,
		disabled:                     disabled,
		deprecatedRepositories:       deprecatedRepositories,
		testWorkflowExecutionResults: testWorkflowExecutionResults,
		metrics:                      metrics,
		webhookRepository:            webhookRepository,
		secretClient:                 secretClient,
		proContext:                   proContext,
		envs:                         envs,
		config:                       config,
		parameters:                   parameters,
	}
}

type WebhookListener struct {
	name                         string
	Uri                          string
	Log                          *zap.SugaredLogger
	HttpClient                   *http.Client
	events                       []testkube.EventType
	selector                     string
	payloadObjectField           string
	payloadTemplate              string
	headers                      map[string]string
	disabled                     bool
	deprecatedRepositories       commons.DeprecatedRepositories
	testWorkflowExecutionResults testworkflow.Repository
	metrics                      v1.Metrics
	webhookRepository            cloudwebhook.WebhookRepository
	secretClient                 secret.Interface
	proContext                   *config.ProContext
	envs                         map[string]string
	config                       map[string]executorv1.WebhookConfigValue
	parameters                   []executorv1.WebhookParameterSchema
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

func (l *WebhookListener) Notify(event testkube.Event) (result testkube.EventResult) {
	var statusCode int
	var err error

	log := l.Log.With(event.Log()...)
	// load global envs to be able to use them in templates
	event.Envs = l.envs

	defer func() {
		var eventType, res string
		if event.Type_ != nil {
			eventType = string(*event.Type_)
		}

		res = "success"
		if result.Error() != "" {
			res = "error"
		}

		l.metrics.IncWebhookEventCount(l.name, eventType, res)
		errorMessage := ""
		if err != nil {
			errorMessage = err.Error()
		}

		if err = l.webhookRepository.CollectExecutionResult(context.Background(), event, l.name, errorMessage, statusCode); err != nil {
			log.Errorw("webhook collecting execution result error", "error", err)
		}
	}()

	switch {
	case l.disabled:
		l.Log.With(event.Log()...).Debug("webhook listener is disabled")
		result = testkube.NewSuccessEventResult(event.Id, "webhook listener is disabled")
		return
	case event.TestExecution != nil && event.TestExecution.DisableWebhooks:
		l.Log.With(event.Log()...).Debug("webhook listener is disabled for test execution")
		result = testkube.NewSuccessEventResult(event.Id, "webhook listener is disabled for test execution")
		return
	case event.TestSuiteExecution != nil && event.TestSuiteExecution.DisableWebhooks:
		l.Log.With(event.Log()...).Debug("webhook listener is disabled for test suite execution")
		result = testkube.NewSuccessEventResult(event.Id, "webhook listener is disabled for test suite execution")
		return
	case event.TestWorkflowExecution != nil && event.TestWorkflowExecution.DisableWebhooks:
		l.Log.With(event.Log()...).Debug("webhook listener is disabled for test workflow execution")
		result = testkube.NewSuccessEventResult(event.Id, "webhook listener is disabled for test workflow execution")
		return
	}

	if event.Type_ != nil && event.Type_.IsBecome() {
		// NOTE(emil): this makes queries for the previous state
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

	log.Debugw("got webhook send result", "response", responseStr)
	result = testkube.NewSuccessEventResult(event.Id, responseStr)
	return
}

func (l *WebhookListener) Kind() string {
	return "webhook"
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
	if err = tmpl.ExecuteTemplate(&buffer, field, NewTemplateVars(event, l.proContext, config)); err != nil {
		log.Errorw(fmt.Sprintf("executing webhook %s error", field), "error", err)
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (l *WebhookListener) hasBecomeState(event testkube.Event) (bool, error) {
	log := l.Log.With(event.Log()...)

	if l.deprecatedRepositories != nil && event.TestExecution != nil && event.Type_ != nil {
		prevStatus, err := l.deprecatedRepositories.TestResults().GetPreviousFinishedState(context.Background(), event.TestExecution.TestName, event.TestExecution.EndTime)
		if err != nil {
			return false, err
		}

		if prevStatus == "" {
			log.Debugw(fmt.Sprintf("no previous finished state for test %s", event.TestExecution.TestName))
			return true, nil
		}

		return event.Type_.IsBecomeExecutionStatus(prevStatus), nil
	}

	if l.deprecatedRepositories != nil && event.TestSuiteExecution != nil && event.TestSuiteExecution.TestSuite != nil && event.Type_ != nil {
		prevStatus, err := l.deprecatedRepositories.TestSuiteResults().GetPreviousFinishedState(context.Background(), event.TestSuiteExecution.TestSuite.Name, event.TestSuiteExecution.EndTime)
		if err != nil {
			return false, err
		}

		if prevStatus == "" {
			log.Debugw(fmt.Sprintf("no previous finished state for test suite %s", event.TestSuiteExecution.TestSuite.Name))
			return true, nil
		}

		return event.Type_.IsBecomeTestSuiteExecutionStatus(prevStatus), nil
	}

	if event.TestWorkflowExecution != nil && event.TestWorkflowExecution.Workflow != nil && event.Type_ != nil {
		prevStatus, err := l.testWorkflowExecutionResults.GetPreviousFinishedState(context.Background(), event.TestWorkflowExecution.Workflow.Name, event.TestWorkflowExecution.StatusAt)
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
