// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
)

type namespaceObj struct {
	clientSet kubernetes.Interface
	namespace string
	ns        *corev1.Namespace
}

func NewNamespace(clientSet kubernetes.Interface, namespace string) *namespaceObj {
	return &namespaceObj{
		clientSet: clientSet,
		namespace: namespace,
	}
}

func (n *namespaceObj) ServiceAccountName() string {
	return "devbox-account"
}

func (n *namespaceObj) Create() (err error) {
	if n.ns != nil {
		return nil
	}

	// Create namespace
	for {
		n.ns, err = n.clientSet.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: n.namespace,
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
			return errors.Wrap(err, "failed to create namespace")
		}
		break
	}

	// Create service account
	serviceAccount, err := n.clientSet.CoreV1().ServiceAccounts(n.namespace).Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: n.ServiceAccountName()},
	}, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create service account")
	}

	// Create service account role
	role, err := n.clientSet.RbacV1().Roles(n.namespace).Create(context.Background(), &rbacv1.Role{
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
	if err != nil {
		return errors.Wrap(err, "failed to create role binding")
	}

	// Create service account role binding
	_, err = n.clientSet.RbacV1().RoleBindings(n.namespace).Create(context.Background(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "devbox-account-rb"},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: n.namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}, metav1.CreateOptions{})

	return nil
}

func (n *namespaceObj) Destroy() error {
	err := n.clientSet.CoreV1().Namespaces().Delete(context.Background(), n.namespace, metav1.DeleteOptions{
		GracePeriodSeconds: common.Ptr(int64(0)),
		PropagationPolicy:  common.Ptr(metav1.DeletePropagationForeground),
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}
