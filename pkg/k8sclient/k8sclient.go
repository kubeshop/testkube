package k8sclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"k8s.io/client-go/dynamic"

	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	apiServerDeploymentSelector = "app.kubernetes.io/name=api-server"
	operatorDeploymentSelector  = "control-plane=controller-manager"
)

// ConnectToK8s establishes a connection to the k8s and returns a *kubernetes.Clientset
func ConnectToK8s() (*kubernetes.Clientset, error) {
	config, err := GetK8sClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// ConnectToK8sDynamic establishes a connection to the k8s and returns a dynamic.Interface
func ConnectToK8sDynamic() (dynamic.Interface, error) {
	config, err := GetK8sClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func GetK8sClientConfig() (*rest.Config, error) {
	var err error
	var config *rest.Config
	k8sConfigExists := false
	homeDir, _ := os.UserHomeDir()
	cubeConfigPath := path.Join(homeDir, ".kube/config")

	if _, err = os.Stat(cubeConfigPath); err == nil {
		k8sConfigExists = true
	}

	if cfg, exists := os.LookupEnv("KUBECONFIG"); exists {
		config, err = clientcmd.BuildConfigFromFlags("", cfg)
	} else if k8sConfigExists {
		config, err = clientcmd.BuildConfigFromFlags("", cubeConfigPath)
	} else {
		config, err = rest.InClusterConfig()
		if err == nil {
			config.QPS = 40.0
			config.Burst = 400.0
		}
	}

	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetIngressAddress gets the hostname or ip address of the ingress with name.
func GetIngressAddress(clientSet kubernetes.Interface, ingressName string, namespace string) (string, error) {
	period := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), period)
	defer cancel()

	var ingress *networkv1.Ingress
	var err error

	processDone := make(chan bool)
	go func() {
		ingressCount := 0
		for ingressCount == 0 {
			ingress, err = clientSet.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
			if err == nil {
				ingressCount = len(ingress.Status.LoadBalancer.Ingress)
			}
			time.Sleep(time.Second)
		}
		processDone <- true
	}()

	select {
	case <-ctx.Done():
		err = fmt.Errorf("Getting ingress failed with timeout(%d sec) previous err: %w.", period, err)
	case <-processDone:
	}

	if err != nil {
		return "", err
	}

	address := ingress.Status.LoadBalancer.Ingress[0].Hostname
	if len(address) == 0 {
		address = ingress.Status.LoadBalancer.Ingress[0].IP
	}

	return address, nil
}

// IsPersistentVolumeClaimBound TODO: add description.
func IsPersistentVolumeClaimBound(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		pv, err := c.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pv.Status.Phase {
		case corev1.ClaimBound:
			return true, nil
		case corev1.ClaimLost:
			return false, nil
		}
		return false, nil
	}
}

// IsPodRunning check if the pod in question is running state
func IsPodRunning(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodRunning, corev1.PodSucceeded:
			return true, nil
		case corev1.PodFailed:
			return false, nil
		}
		return false, nil
	}
}

// HasPodSucceeded custom method for checing if Pod is succeded (handles PodFailed state too)
func HasPodSucceeded(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return true, nil
		case corev1.PodFailed:
			return false, nil
		}
		return false, nil
	}
}

// IsPodReady check if the pod in question is running state
func IsPodReady(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if len(pod.Status.ContainerStatuses) == 0 {
			return false, nil
		}

		for _, c := range pod.Status.ContainerStatuses {
			if !c.Ready {
				return false, nil
			}
		}
		return true, nil
	}
}

// WaitForPodsReady wait for pods to be running with a timeout, return error
func WaitForPodsReady(k8sClient kubernetes.Interface, namespace string, instance string, timeout time.Duration) error {
	ctx := context.TODO()
	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/instance=" + instance})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if err := wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, IsPodRunning(k8sClient, pod.Name, namespace)); err != nil {
			return err
		}
		if err := wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, IsPodReady(k8sClient, pod.Name, namespace)); err != nil {
			return err
		}
	}
	return nil
}

// GetClusterVersion returns the current version of the Kubernetes cluster
func GetClusterVersion(k8sClient kubernetes.Interface) (string, error) {
	version, err := k8sClient.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}

	return version.String(), nil
}

// GetAPIServerLogs returns the latest logs from the API server deployment
func GetAPIServerLogs(ctx context.Context, k8sClient kubernetes.Interface, namespace string) ([]string, error) {
	return GetPodLogs(ctx, k8sClient, namespace, apiServerDeploymentSelector)
}

// GetOperatorLogs returns the logs from the operator
func GetOperatorLogs(ctx context.Context, k8sClient kubernetes.Interface, namespace string) ([]string, error) {
	return GetPodLogs(ctx, k8sClient, namespace, operatorDeploymentSelector)
}

// GetPodLogs returns logs for pods specified by the label selector
func GetPodLogs(ctx context.Context, k8sClient kubernetes.Interface, namespace string, selector string) ([]string, error) {
	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return []string{}, fmt.Errorf("could not get operator pods: %w", err)
	}

	logs := []string{}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			podLogs, err := k8sClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
				Container: container.Name,
			}).Stream(ctx)
			if err != nil {
				return []string{}, fmt.Errorf("error in getting operator deployment: %w", err)
			}
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				return []string{}, fmt.Errorf("error in copy information from podLogs to buf")
			}
			logs = append(logs, fmt.Sprintf("Pod: %s \n Logs: \n %s", pod.Name, buf.String()))
		}
	}
	return logs, nil
}
