package client

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestProxy(t *testing.T) {
	t.Skip("Implement me please :)")
	clcfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		panic(err.Error())
	}
	restcfg, err := clientcmd.NewNonInteractiveClientConfig(
		*clcfg, "", &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(restcfg)

	req := clientset.CoreV1().RESTClient().Get().
		Namespace("default").
		Resource("services").
		Name("api-server-chart:8080").
		SubResource("proxy").
		// The server URL path, without leading "/" goes here...
		Suffix("v1/scripts").Param("namespace", "default")

	res := req.Do(context.Background())

	if err != nil {
		panic(err.Error())
	}
	rawbody, err := res.Raw()
	if err != nil {
		panic(err.Error())
	}
	fmt.Print(string(rawbody))

	t.Fail()

}
