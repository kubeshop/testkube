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
