// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func TestMergePodConfig_RuntimeSchedulerAndHostUsers(t *testing.T) {
	dst := &testworkflowsv1.PodConfig{
		RuntimeClassName: common.Ptr("runc"),
		SchedulerName:    "default-scheduler",
	}
	include := &testworkflowsv1.PodConfig{
		RuntimeClassName: common.Ptr("gvisor"),
		HostUsers:        common.Ptr(false),
	}

	got := MergePodConfig(dst, include)

	assert.Equal(t, common.Ptr("gvisor"), got.RuntimeClassName)
	// Not set in the include, so the workflow's own value stays.
	assert.Equal(t, "default-scheduler", got.SchedulerName)
	assert.Equal(t, common.Ptr(false), got.HostUsers)
}

func TestMergePodConfig_HostNamespacesAndFQDN(t *testing.T) {
	dst := &testworkflowsv1.PodConfig{
		HostNetwork:       common.Ptr(true),
		SetHostnameAsFQDN: common.Ptr(true),
	}
	include := &testworkflowsv1.PodConfig{
		HostNetwork: common.Ptr(false),
		HostIPC:     common.Ptr(true),
	}

	got := MergePodConfig(dst, include)

	assert.Equal(t, common.Ptr(false), got.HostNetwork)
	assert.Equal(t, common.Ptr(true), got.HostIPC)
	// Not set in the include, so the workflow's own value stays.
	assert.Equal(t, common.Ptr(true), got.SetHostnameAsFQDN)
}
