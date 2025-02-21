// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/internal/common"
)

var (
	ErrNotDevboxNamespace = errors.New("selected namespace exists and is not devbox")
)

type NamespaceObject struct {
	name       string
	clientSet  *kubernetes.Clientset
	restConfig *rest.Config
	namespace  *corev1.Namespace
}

func NewNamespace(kubeClient *kubernetes.Clientset, kubeRestConfig *rest.Config, name string) *NamespaceObject {
	return &NamespaceObject{
		name:       name,
		clientSet:  kubeClient,
		restConfig: kubeRestConfig,
	}
}

func (n *NamespaceObject) Name() string {
	return n.name
}

func (n *NamespaceObject) ServiceAccountName() string {
	return "devbox-account"
}

func (n *NamespaceObject) Pod(name string) *PodObject {
	return NewPod(n.clientSet, n.restConfig, n.name, name)
}

func (n *NamespaceObject) create() error {
	for {
		namespace, err := n.clientSet.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: n.name,
				Labels: map[string]string{
					"testkube.io/devbox": "namespace",
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "being deleted") {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if k8serrors.IsAlreadyExists(err) {
				namespace, err = n.clientSet.CoreV1().Namespaces().Get(context.Background(), n.name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if namespace.Labels["testkube.io/devbox"] != "namespace" {
					return ErrNotDevboxNamespace
				}
				err = n.clientSet.CoreV1().Namespaces().Delete(context.Background(), n.name, metav1.DeleteOptions{
					GracePeriodSeconds: common.Ptr(int64(0)),
					PropagationPolicy:  common.Ptr(metav1.DeletePropagationForeground),
				})
				if err != nil {
					return err
				}
				continue
			}
			return errors.Wrap(err, "failed to create namespace")
		}
		n.namespace = namespace
		return nil
	}
}

func (n *NamespaceObject) createServiceAccount() error {
	// Create service account
	serviceAccount, err := n.clientSet.CoreV1().ServiceAccounts(n.name).Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: n.ServiceAccountName()},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create service account")
	}

	// Create service account role
	role, err := n.clientSet.RbacV1().Roles(n.name).Create(context.Background(), &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-account-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list", "create", "delete", "deletecollection"},
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
			},
			{
				Verbs:     []string{"get", "watch", "list", "create", "patch", "update", "delete", "deletecollection"},
				APIGroups: []string{""},
				Resources: []string{"pods", "persistentvolumeclaims", "secrets", "configmaps"},
			},
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{""},
				Resources: []string{"pods/log", "events"},
			},
			{
				Verbs:     []string{"get", "watch", "list", "create", "patch", "update", "delete", "deletecollection"},
				APIGroups: []string{"testworkflows.testkube.io"},
				Resources: []string{"testworkflows", "testworkflows/status", "testworkflowtemplates"},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create roles")
	}

	// Create service account role binding
	_, err = n.clientSet.RbacV1().RoleBindings(n.name).Create(context.Background(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "devbox-account-rb"},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: n.name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create role bindings")
	}
	return nil
}

func (n *NamespaceObject) Create() error {
	if n.namespace != nil {
		return nil
	}

	if err := n.create(); err != nil {
		return err
	}
	if err := n.createServiceAccount(); err != nil {
		return err
	}
	return nil
}

func (n *NamespaceObject) Destroy() error {
	err := n.clientSet.CoreV1().Namespaces().Delete(context.Background(), n.name, metav1.DeleteOptions{
		GracePeriodSeconds: common.Ptr(int64(0)),
		PropagationPolicy:  common.Ptr(metav1.DeletePropagationForeground),
	})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	err = n.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("testkube.io/devbox-name=%s", n.name),
	})
	return err
}
