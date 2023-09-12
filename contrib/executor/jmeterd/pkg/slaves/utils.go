package slaves

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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

func getSlavePodConfiguration(testName string, runnerExecution testkube.Execution, envVariables map[string]testkube.Variable, envParams envs.Params) (*v1.Pod, error) {
	runnerExecutionStr, err := json.Marshal(runnerExecution)
	if err != nil {
		return nil, err
	}

	podName := fmt.Sprintf("%s-jmeter-slave", testName)
	if err != nil {
		return nil, err
	}
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyAlways,
			InitContainers: []v1.Container{
				{
					Name:            "init",
					Image:           "kubeshop/testkube-init-executor:1.14.3",
					Command:         []string{"/bin/runner", string(runnerExecutionStr)},
					Env:             getSlaveRunnerEnv(envParams, runnerExecution),
					ImagePullPolicy: v1.PullIfNotPresent,
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/data",
							Name:      "data-volume",
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:            "main",
					Image:           "kubeshop/testkube-jmeterd-slaves:999.0.0",
					Env:             getSlaveConfigurationEnv(envVariables),
					ImagePullPolicy: v1.PullIfNotPresent,
					Ports: []v1.ContainerPort{
						{
							ContainerPort: serverPort,
							Name:          "server-port",
						}, {
							ContainerPort: localPort,
							Name:          "local-port",
						},
					},
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/data",
							Name:      "data-volume",
						},
					},
					LivenessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.FromInt(serverPort),
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   1,
					},
					ReadinessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.FromInt(serverPort),
							},
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 10,
						PeriodSeconds:       5,
						TimeoutSeconds:      1,
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name:         "data-volume",
					VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
				},
			},
		},
	}, nil
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
