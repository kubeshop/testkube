package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// ScriptsKubernetesAPI. Struct which will hold Scripts returned via k8s API calls.
type ScriptsKubernetesAPI struct {
	name map[string]string
}

// ScriptsNamesListKubernetesAPI is the list of all Scripts-kind CRDs found in the namespace
type ScriptsNamesListKubernetesAPI struct {
	Names []string
}

// Get. Function for returning newman CRD script. Accepts ScriptName. Returng value(string).
func (s *ScriptsKubernetesAPI) Get(scriptName string) (string, error) {

	// checking if there is no empty name on ScriptsFromCRD
	if s.name[scriptName] == "" {
		return "", errors.NewGone("Returned match is empty")
	}
	fmt.Println("Found ", s.name[scriptName])
	return s.name[scriptName], nil
}

// GetScriptsNamesListAPI. Returns ScriptsNamesListKubernetesAPI struct from k8s API.
func (s *ScriptsNamesListKubernetesAPI) GetScriptsNamesListAPI(namespace string) (ScriptsNamesListKubernetesAPI, error) {
	//initialize Client:
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
		os.Exit(1)
	}

	// Example of Using a typed object.
	// pod := &corev1.PodList{}

	// cl is a created client. Using structured object (Pod)
	// err = cl.List(context.Background(), pod, client.InNamespace("kube-system"))

	// Using a unstructured (CRD) object.
	un := &unstructured.UnstructuredList{}
	un.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kubetest.kubetest.io",
		Kind:    "Script",
		Version: "v1",
	})

	err = cl.List(context.Background(), un, client.InNamespace(namespace))

	// List of Scripts names
	var res = ScriptsNamesListKubernetesAPI{}

	// Handling if there is an error in getting scripts:
	if err != nil {
		fmt.Printf("failed to list CRDs in namespace default: %v\n", err)
		os.Exit(1)
	} else {
		for un_item := 0; un_item < len(un.Items); un_item++ {
			res.Names = append(res.Names, un.Items[un_item].GetName())
		}
	}
	return res, err
}

func main() {

	// Defining ScriptsKubernetesAPI struct variable for testing purposes
	var test = ScriptsKubernetesAPI{
		name: map[string]string{
			"First":  "firstValue",
			"Second": "secondValue",
			"Third":  "thirdValue",
		},
	}
	// Skeleton for getting actuall Objectb based on the Script name
	name, _ := test.Get("First")
	fmt.Println("Reading with the Get() method: ", name)

	// Defining Scripts structure for holding results.
	var res = ScriptsNamesListKubernetesAPI{
		Names: []string{},
	}

	// Getting list of Scripts within namespace:
	ScriptsNames, _ := res.GetScriptsNamesListAPI("default")

	fmt.Printf("script: %v\n", ScriptsNames.Names)
}
