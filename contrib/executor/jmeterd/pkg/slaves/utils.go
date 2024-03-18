package slaves

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	defaultSlavesCount = 0
	serverPort         = 1099
	localPort          = 60001
)

func getSlaveRunnerEnv(envs map[string]string, runnerExecution testkube.Execution) []v1.EnvVar {
	var gitEnvs []v1.EnvVar
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

	var runnerEnvVars []v1.EnvVar
	for key, value := range envs {
		runnerEnvVars = append(runnerEnvVars, v1.EnvVar{Name: key, Value: value})
	}

	return append(runnerEnvVars, gitEnvs...)
}

func getSlaveConfigurationEnv(slaveEnv map[string]testkube.Variable, slavesPodNumber int) []v1.EnvVar {
	var envVars []v1.EnvVar
	for envKey, t := range slaveEnv {
		envVars = append(envVars, v1.EnvVar{Name: envKey, Value: t.Value})
	}

	envVars = append(envVars, v1.EnvVar{Name: "SLAVE_POD_NUMBER", Value: strconv.Itoa(slavesPodNumber)})
	return envVars
}

func isPodReady(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
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

func GetSlavesCount(vars map[string]testkube.Variable) (int, error) {
	count := vars[SlavesCount]
	if count.Value == "" {
		return defaultSlavesCount, nil
	}

	slavesCount, err := strconv.Atoi(count.Value)
	if err != nil {
		return 0, errors.Errorf("invalid SLAVES_COUNT value, expected integer, got: %v", count.Value)
	}
	if slavesCount < 0 {
		return 0, errors.Errorf("SLAVES_COUNT cannot be less than 0, got: %v", count.Value)
	}
	return slavesCount, err
}

func validateAndGetSlavePodName(testName string, executionId string, currentSlaveCount int) string {
	slavePodName := fmt.Sprintf("%s-slave-%v-%s", testName, currentSlaveCount, executionId)
	if len(slavePodName) > 64 {
		//Get first 20 chars from testName name if pod name > 64
		shortExecutionName := testName[:20]
		slavePodName = fmt.Sprintf("%s-slave-%v-%s", shortExecutionName, currentSlaveCount, executionId)
	}
	return slavePodName
}
