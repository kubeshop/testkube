package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var c client.Client

func main() {
	// Using a typed object.
	pod := &corev1.PodList{}
	// c is a created client.
	fmt.Printf("%+v\n", c)
	fmt.Printf("%+v\n", pod)

	_ = c.List(context.Background(), pod)
	// Using a unstructured object.
	u := &unstructured.UnstructuredList{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "DeploymentList",
		Version: "v1",
	})
	_ = c.List(context.Background(), u)
}
