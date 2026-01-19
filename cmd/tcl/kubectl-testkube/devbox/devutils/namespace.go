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
	"golang.org/x/sync/singleflight"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/internal/common"
)

const (
	roleName                = "devbox-role"
	executionRoleName       = "devbox-execution-role"
	jobServiceAccountName   = "devbox-job-account"
	agentServiceAccountName = "devbox-account"
)

var (
	ErrNotDevboxNamespace = errors.New("selected namespace exists and is not devbox")
)

type NamespaceObject struct {
	name               string
	executionName      string
	clientSet          *kubernetes.Clientset
	restConfig         *rest.Config
	namespace          *corev1.Namespace
	executionNamespace *corev1.Namespace
	sf                 singleflight.Group
}

func NewNamespace(kubeClient *kubernetes.Clientset, kubeRestConfig *rest.Config, name, executionName string) *NamespaceObject {
	if executionName == "" {
		executionName = name
	}
	return &NamespaceObject{
		name:          name,
		executionName: executionName,
		clientSet:     kubeClient,
		restConfig:    kubeRestConfig,
	}
}

func (n *NamespaceObject) Name() string {
	return n.name
}

func (n *NamespaceObject) ExecutionName() string {
	return n.executionName
}

func (n *NamespaceObject) ServiceAccountName() string {
	return agentServiceAccountName
}

func (n *NamespaceObject) JobServiceAccountName() string {
	return jobServiceAccountName
}

func (n *NamespaceObject) Pod(name string) *PodObject {
	return NewPod(n.clientSet, n.restConfig, n.name, name)
}

func (n *NamespaceObject) create() error {
	ns, err := n.createNs(n.name)
	if err != nil {
		return err
	}
	n.namespace = ns
	if n.name != n.executionName {
		_, err = n.createNs(n.executionName)
		if err != nil {
			return err
		}
	}
	n.executionNamespace = ns
	return nil
}

func (n *NamespaceObject) createNs(name string) (*corev1.Namespace, error) {
	for {
		namespace, err := n.clientSet.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
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
				namespace, err = n.clientSet.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
				if err != nil {
					return nil, err
				}
				if namespace.Labels["testkube.io/devbox"] != "namespace" {
					return nil, ErrNotDevboxNamespace
				}
				err = n.clientSet.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{
					GracePeriodSeconds: common.Ptr(int64(0)),
					PropagationPolicy:  common.Ptr(metav1.DeletePropagationForeground),
				})
				if err != nil {
					return nil, err
				}
				continue
			}
			return nil, errors.Wrap(err, "failed to create namespace")
		}
		return namespace, nil
	}
}

func (n *NamespaceObject) createRole() error {
	// Create the role
	_, err := n.clientSet.RbacV1().Roles(n.name).Create(context.Background(), &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list", "create", "patch", "update", "delete"},
				APIGroups: []string{""},
				Resources: []string{"secrets", "configmaps"},
			},
			{
				Verbs:     []string{"get", "watch", "list", "create", "patch", "update", "delete"},
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
			},
			{
				Verbs:     []string{"get", "watch", "list", "create", "patch", "update", "delete", "deletecollection"},
				APIGroups: []string{"testworkflows.testkube.io"},
				Resources: []string{"testworkflows", "testworkflows/status", "testworkflowtemplates", "testworkflowexecutions"},
			},
			{
				Verbs:     []string{"get", "watch", "list", "create", "patch", "update", "delete", "deletecollection"},
				APIGroups: []string{"tests.testkube.io"},
				Resources: []string{"testtriggers", "testexecutions", "testsuiteexecutions"},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	// Create the role binding for the Agent
	_, err = n.clientSet.RbacV1().RoleBindings(n.name).Create(context.Background(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: roleName + "-rb"},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      n.ServiceAccountName(),
				Namespace: n.name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (n *NamespaceObject) createExecutionRole() error {
	// Create the role
	_, err := n.clientSet.RbacV1().Roles(n.executionName).Create(context.Background(), &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: executionRoleName,
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
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	// Create the role binding for the Agent
	_, err = n.clientSet.RbacV1().RoleBindings(n.executionName).Create(context.Background(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: executionRoleName + "-rb"},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      n.ServiceAccountName(),
				Namespace: n.name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     executionRoleName,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	// Create the role binding for the jobs (parallel and services)
	_, err = n.clientSet.RbacV1().RoleBindings(n.executionName).Create(context.Background(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: executionRoleName + "-job-rb"},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      n.JobServiceAccountName(),
				Namespace: n.executionName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     executionRoleName,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (n *NamespaceObject) createServiceAccount() error {
	// Create service account for the Agent
	_, err := n.clientSet.CoreV1().ServiceAccounts(n.name).Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: n.ServiceAccountName()},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create agent service account")
	}

	// Create service account for the Jobs
	_, err = n.clientSet.CoreV1().ServiceAccounts(n.executionName).Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: n.JobServiceAccountName()},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create job service account")
	}

	// Create roles & bindings
	if err = n.createRole(); err != nil {
		return errors.Wrap(err, "failed to create management role")
	}
	if err = n.createExecutionRole(); err != nil {
		return errors.Wrap(err, "failed to create execution role")
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
	_, err, _ := n.sf.Do("destroy", func() (interface{}, error) {
		return nil, n.destroy()
	})
	return err
}

func (n *NamespaceObject) destroy() error {
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
	if n.executionName != n.name {
		err := n.clientSet.CoreV1().Namespaces().Delete(context.Background(), n.name, metav1.DeleteOptions{
			GracePeriodSeconds: common.Ptr(int64(0)),
			PropagationPolicy:  common.Ptr(metav1.DeletePropagationForeground),
		})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return err
}
