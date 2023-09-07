package cronjob

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	batchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	"k8s.io/client-go/kubernetes"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

// Client data struct for managing running cron jobs
type Client struct {
	ClientSet       *kubernetes.Clientset
	Log             *zap.SugaredLogger
	serviceName     string
	servicePort     int
	cronJobTemplate string
	Namespace       string
}

type CronJobOptions struct {
	Schedule                  string
	Resource                  string
	Data                      string
	Labels                    map[string]string
	CronJobTemplate           string
	CronJobTemplateExtensions string
}

type templateParameters struct {
	Id                        string
	Name                      string
	Namespace                 string
	ServiceName               string
	ServicePort               int
	Schedule                  string
	Resource                  string
	CronJobTemplate           string
	CronJobTemplateExtensions string
	Data                      string
	Labels                    map[string]string
}

// NewClient is a method to create new cron job client
func NewClient(serviceName string, servicePort int, cronJobTemplate string, namespace string) (*Client, error) {
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
		Namespace:       namespace,
	}, nil
}

// Get is a method to retrieve an existing cron job
func (c *Client) Get(name string) (*v1.CronJob, error) {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(c.Namespace)
	ctx := context.Background()

	cronJob, err := cronJobClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return cronJob, nil
}

// Apply is a method to create or update a cron job
func (c *Client) Apply(id, name string, options CronJobOptions) error {
	template := c.cronJobTemplate
	if options.CronJobTemplate != "" {
		template = options.CronJobTemplate
	}

	cronJobClient := c.ClientSet.BatchV1().CronJobs(c.Namespace)
	ctx := context.Background()

	parameters := templateParameters{
		Id:                        id,
		Name:                      name,
		Namespace:                 c.Namespace,
		ServiceName:               c.serviceName,
		ServicePort:               c.servicePort,
		Schedule:                  options.Schedule,
		Resource:                  options.Resource,
		CronJobTemplate:           template,
		CronJobTemplateExtensions: options.CronJobTemplateExtensions,
		Data:                      options.Data,
		Labels:                    options.Labels,
	}

	cronJobSpec, err := NewApplySpec(c.Log, parameters)
	if err != nil {
		return err
	}

	if _, err := cronJobClient.Apply(ctx, cronJobSpec, metav1.ApplyOptions{
		FieldManager: "application/apply-patch"}); err != nil {
		return err
	}

	return nil
}

// UpdateLabels is a method to update an existing cron job labels
func (c *Client) UpdateLabels(cronJobSpec *v1.CronJob, oldLabels, newLabels map[string]string) error {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(c.Namespace)
	ctx := context.Background()

	for key := range oldLabels {
		delete(cronJobSpec.Labels, key)
	}

	for key, value := range newLabels {
		cronJobSpec.Labels[key] = value
	}

	if _, err := cronJobClient.Update(ctx, cronJobSpec, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// Delete is a method to delete an existing cron job
func (c *Client) Delete(name string) error {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(c.Namespace)
	ctx := context.Background()

	if err := cronJobClient.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// DeleteAll is a method to delete all existing cron jobs
func (c *Client) DeleteAll(resource, selector string) error {
	cronJobClient := c.ClientSet.BatchV1().CronJobs(c.Namespace)
	ctx := context.Background()

	filter := fmt.Sprintf("testkube=%s", resource)
	if selector != "" {
		filter += "," + selector
	}

	if err := cronJobClient.DeleteCollection(ctx, metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: filter}); err != nil {
		return err
	}

	return nil
}

// NewApplySpec is a method to return cron job apply spec
func NewApplySpec(log *zap.SugaredLogger, parameters templateParameters) (*batchv1.CronJobApplyConfiguration, error) {
	tmpl, err := utils.NewTemplate("cronJob").Parse(parameters.CronJobTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating cron job spec from options.CronJobTemplate error: %w", err)
	}

	parameters.Data = strings.ReplaceAll(parameters.Data, "'", "''''")
	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "cronJob", parameters); err != nil {
		return nil, fmt.Errorf("executing cron job spec template: %w", err)
	}

	var cronJob batchv1.CronJobApplyConfiguration
	cronJobSpec := buffer.String()
	if parameters.CronJobTemplateExtensions != "" {
		tmplExt, err := utils.NewTemplate("cronJobExt").Parse(parameters.CronJobTemplateExtensions)
		if err != nil {
			return nil, fmt.Errorf("creating cron job extensions spec from default template error: %w", err)
		}

		var bufferExt bytes.Buffer
		if err = tmplExt.ExecuteTemplate(&bufferExt, "cronJobExt", parameters); err != nil {
			return nil, fmt.Errorf("executing cron job extensions spec default template: %w", err)
		}

		if cronJobSpec, err = merge2.MergeStrings(bufferExt.String(), cronJobSpec, false, kyaml.MergeOptions{}); err != nil {
			return nil, fmt.Errorf("merging cron job spec templates: %w", err)
		}
	}

	log.Debug("Cron job specification", cronJobSpec)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(cronJobSpec), len(cronJobSpec))
	if err := decoder.Decode(&cronJob); err != nil {
		return nil, fmt.Errorf("decoding cron job spec error: %w", err)
	}

	for key, value := range parameters.Labels {
		cronJob.Labels[key] = value
	}

	return &cronJob, nil
}

// GetMetadataName returns cron job metadata name
func GetMetadataName(name, resource string) string {
	return fmt.Sprintf("%s-%s", name, resource)
}
