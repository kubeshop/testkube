package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/k8sclient"
)

const (
	pollTimeout  = 24 * time.Hour
	pollInterval = 200 * time.Millisecond
)

// NewRunner creates scraper runner
func NewRunner(ctx context.Context, params envs.Params) (*ScraperRunner, error) {
	var err error
	r := &ScraperRunner{
		Params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	if params.SidecarScraperMode {
		r.clientset, err = k8sclient.ConnectToK8s()
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

// ScraperRunner prepares data for executor
type ScraperRunner struct {
	Params    envs.Params
	Scraper   scraper.Scraper
	clientset kubernetes.Interface
}

var _ runner.Runner = &ScraperRunner{}

// Run prepares data for executor
func (r *ScraperRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	// check that the artifact dir exists
	if execution.ArtifactRequest == nil {
		return *result.Err(errors.Errorf("executor only support artifact based tests")), nil
	}

	if execution.ArtifactRequest.StorageClassName == "" && !execution.ArtifactRequest.UseDefaultStorageClassName &&
		!execution.ArtifactRequest.SidecarScraper {
		return *result.Err(errors.Errorf("artifact request should have not empty storage class name")), nil
	}

	if r.Params.ScrapperEnabled {
		var mountPath string
		if r.Params.SidecarScraperMode {
			podsClient := r.clientset.CoreV1().Pods(execution.TestNamespace)
			pods, err := executor.GetJobPods(ctx, podsClient, execution.Id, 1, 10)
			if err != nil {
				return *result.Err(errors.Wrap(err, "error getting job pods")), nil
			}

			for _, pod := range pods.Items {
				if pod.Labels["job-name"] == execution.Id {
					if err = wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, r.isContainerTerminated(pod.Name, execution.Id, execution.TestNamespace)); err != nil {
						return *result.Err(errors.Wrap(err, "waiting for executor pod complete error")), nil
					}
				}
			}
		} else {
			mountPath = filepath.Join(r.Params.DataDir, "artifacts")
			if execution.ArtifactRequest.VolumeMountPath != "" {
				mountPath = execution.ArtifactRequest.VolumeMountPath
			}

			_, err = os.Stat(mountPath)
			if errors.Is(err, os.ErrNotExist) {
				return result, err
			}
		}

		directories := execution.ArtifactRequest.Dirs
		if len(directories) == 0 {
			directories = []string{"."}
		}

		for i := range directories {
			directories[i] = filepath.Join(mountPath, directories[i])
		}

		masks := execution.ArtifactRequest.Masks
		output.PrintLog(fmt.Sprintf("Scraping directories: %v with masks: %v", directories, masks))

		if err := r.Scraper.Scrape(ctx, directories, masks, execution); err != nil {
			return *result.Err(err), errors.Wrap(err, "error scraping artifacts from container executor")
		}
	}

	return result, nil
}

// GetType returns runner type
func (r *ScraperRunner) GetType() runner.Type {
	return runner.TypeFin
}

// isContainerTerminated checks if pod container is terminated through kubernetes API
func (r *ScraperRunner) isContainerTerminated(podName, containerName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		pod, err := r.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			return true, nil
		}

		if err = executor.IsPodFailed(pod); err != nil {
			return true, err
		}

		for _, s := range pod.Status.ContainerStatuses {
			if s.Name != containerName {
				continue
			}

			if s.State.Terminated != nil {
				return true, nil
			}
		}

		return false, nil
	}
}
