package k8sclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ConnectToK8s establishes a connection to the k8s and returns a *kubernetes.Clientset
func ConnectToK8s() (*kubernetes.Clientset, error) {
	var err error
	var config *rest.Config
	k8sConfigExists := false
	cubeConfigPath, _ := filepath.Abs("~/.kube/config")
	if _, err := os.Stat(cubeConfigPath); err == nil {
		k8sConfigExists = true
	}
	if cfg, exists := os.LookupEnv("KUBECONFIG"); !exists && !k8sConfigExists {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", cfg)
	}

	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// GetIngressAddress gets the hostname or ip address of the ingress with name.
func GetIngressAddress(clientSet *kubernetes.Clientset, ingressName string, namespace string) (string, error) {
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
func IsPersistentVolumeClaimBound(c *kubernetes.Clientset, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
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
func IsPodRunning(c *kubernetes.Clientset, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
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
