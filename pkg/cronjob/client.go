package cronjob

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	batchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	testkubeCronJobLabel = "cronjob"
)

// Client data struct for managing running cron jobs
type Client struct {
	ClientSet       *kubernetes.Clientset
	Log             *zap.SugaredLogger
	serviceName     string
	servicePort     int
	cronJobTemplate string
}

type CronJobOptions struct {
	Schedule        string
	Resource        string
	CronJobTemplate string
	Data            string
}

type templateParameters struct {
	Name            string
	Namespace       string
	ServiceName     string
	ServicePort     int
	Schedule        string
	Resource        string
	CronJobTemplate string
	Data            string
}

// NewClient is a method to create new cron job client
func NewClient(serviceName string, servicePort int, cronJobTemplate string) (*Client, error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, err
	}

	return &Client{
		ClientSet:       clientSet,
		Log:             log.DefaultLogger,
		serviceName:     serviceName,
		servicePort:     servicePort,
		cronJobTemplate: cronJobTemplate,
	}, nil
}

// Get is a method to retrieve an existing secret
func (c *Client) Get(id, namespace string) (*v1.CronJob, error) {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(namespace)
	ctx := context.Background()

	cronJob, err := cronJobClient.Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return cronJob, nil
}

// Apply is a method to create or update a cron job
func (c *Client) Apply(id, namespace string, options CronJobOptions) error {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(namespace)
	ctx := context.Background()

	parameters := templateParameters{
		Name:        id,
		Namespace:   namespace,
		ServiceName: c.serviceName,
		ServicePort: c.servicePort,
		Schedule:    options.Schedule,
		Resource:    options.Resource,
		Data:        options.Data,
	}

	cronJobSpec, err := NewApplySpec(c.Log, parameters)
	if err != nil {
		return err
	}

	cronJobSpec.Labels = map[string]string{"testkube": testkubeCronJobLabel}
	if _, err := cronJobClient.Apply(ctx, cronJobSpec, metav1.ApplyOptions{
		FieldManager: "application/apply-patch"}); err != nil {
		return err
	}

	return nil
}

// Delete is a method to delete an existing secret
func (c *Client) Delete(id, namespace string) error {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(namespace)
	ctx := context.Background()

	if err := cronJobClient.Delete(ctx, id, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// DeleteAll is a method to delete all existing secrets
func (c *Client) DeleteAll(namespace string) error {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(namespace)
	ctx := context.Background()

	if err := cronJobClient.DeleteCollection(ctx, metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: fmt.Sprintf("testkube=%s", testkubeCronJobLabel)}); err != nil {
		return err
	}

	return nil
}

// NewApplySpec is a method to return cron job apply spec
func NewApplySpec(log *zap.SugaredLogger, parameters templateParameters) (*batchv1.CronJobApplyConfiguration, error) {
	tmpl, err := template.New("cronjob").Parse(parameters.CronJobTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating cron job spec from options.CronJobTemplate error: %w", err)
	}

	parameters.Data = strings.ReplaceAll(parameters.Data, "'", "''")
	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "cronjob", parameters); err != nil {
		return nil, fmt.Errorf("executing cron job spec template: %w", err)
	}

	var cronJob batchv1.CronJobApplyConfiguration
	cronJobSpec := buffer.String()
	log.Debug("Cron job specification", cronJobSpec)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(cronJobSpec), len(cronJobSpec))
	if err := decoder.Decode(&cronJob); err != nil {
		return nil, fmt.Errorf("decoding cron job spec error: %w", err)
	}

	return &cronJob, nil
}
