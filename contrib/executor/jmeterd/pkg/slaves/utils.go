package slaves

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

const (
	defaultSlavesCount = 1
	serverPort         = 1099
	localPort          = 60001
)

func getSlaveRunnerEnv(envParams envs.Params, runnerExecution testkube.Execution) []v1.EnvVar {

	gitEnvs := []v1.EnvVar{}
	if runnerExecution.Content.Type_ == "git" && runnerExecution.Content.Repository.UsernameSecret != nil && runnerExecution.Content.Repository.TokenSecret != nil {
		gitEnvs = append(gitEnvs, v1.EnvVar{

			Name: "RUNNER_GITUSERNAME",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: runnerExecution.Content.Repository.UsernameSecret.Name,
					},
					Key: runnerExecution.Content.Repository.UsernameSecret.Key,
				},
			},
		}, v1.EnvVar{
			Name: "RUNNER_GITTOKEN",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: runnerExecution.Content.Repository.TokenSecret.Name,
					},
					Key: runnerExecution.Content.Repository.TokenSecret.Key,
				},
			},
		},
		)
	}

	return append([]v1.EnvVar{
		{
			Name:  "RUNNER_ENDPOINT",
			Value: envParams.Endpoint,
		}, {
			Name:  "RUNNER_ACCESSKEYID",
			Value: envParams.AccessKeyID,
		}, {
			Name:  "RUNNER_SECRETACCESSKEY",
			Value: envParams.SecretAccessKey,
		}, {
			Name:  "RUNNER_TOKEN",
			Value: envParams.Token,
		}, {
			Name:  "RUNNER_BUCKET",
			Value: envParams.Bucket,
		}, {
			Name:  "RUNNER_SSL",
			Value: fmt.Sprintf("%v", envParams.Ssl),
		}, {
			Name:  "RUNNER_SCRAPPERENABLED",
			Value: fmt.Sprintf("%v", envParams.ScrapperEnabled),
		}, {
			Name:  "RUNNER_DATADIR",
			Value: envParams.DataDir,
		}, {
			Name:  "RUNNER_CLOUD_MODE",
			Value: fmt.Sprintf("%v", envParams.CloudMode),
		}, {
			Name:  "RUNNER_CLOUD_API_KEY",
			Value: envParams.CloudAPIKey,
		}, {
			Name:  "RUNNER_CLOUD_API_TLS_INSECURE",
			Value: fmt.Sprintf("%v", envParams.CloudAPITLSInsecure),
		}, {
			Name:  "RUNNER_CLOUD_API_URL",
			Value: envParams.CloudAPIURL,
		},
	}, gitEnvs...)
}

func getSlaveConfigurationEnv(slaveEnv map[string]testkube.Variable) []v1.EnvVar {
	envVars := []v1.EnvVar{}
	for envKey, t := range slaveEnv {
		envVars = append(envVars, v1.EnvVar{Name: envKey, Value: t.Value})
	}
	return envVars
}

func isPodReady(ctx context.Context, c kubernetes.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, condition := range pod.Status.Conditions {
			isReadyType := condition.Type == v1.PodReady
			isConditionTrue := condition.Status == v1.ConditionTrue
			isRunningPhase := pod.Status.Phase == v1.PodRunning
			ipNotEmpty := pod.Status.PodIP != ""
			if isReadyType && isConditionTrue && isRunningPhase && ipNotEmpty {
				return true, nil
			}
		}
		return false, nil
	}
}

func getSlavesCount(count testkube.Variable) (int, error) {
	if count.Value == "" {
		output.PrintLogf("Slaves count not provided in the SLAVES_COUNT env variable. Defaulting to %v slaves", defaultSlavesCount)
		return defaultSlavesCount, nil
	}

	rplicaCount, err := strconv.Atoi(count.Value)
	if err != nil {
		return 0, err
	}
	return rplicaCount, err
}

func GetSlavesIpString(podNameIpMap map[string]string) string {
	podIps := []string{}
	for _, ip := range podNameIpMap {
		podIps = append(podIps, ip)
	}
	return strings.Join(podIps, ",")
}

func ValidateAndGetSlavePodName(testName string, executionId string, currentSlaveCount int) string {
	slavePodName := fmt.Sprintf("%s-slave-%v-%s", testName, currentSlaveCount, executionId)
	if len(slavePodName) > 64 {
		//Get first 20 chars from testName name if pod name > 64
		shortExecutionName := testName[:20]
		slavePodName = fmt.Sprintf("%s-slave-%v-%s", shortExecutionName, currentSlaveCount, executionId)
	}
	return slavePodName

}
