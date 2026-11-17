package commands

import (
	"testing"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
)

func TestIsContextApprovedWithNoConfirm(t *testing.T) {
	options := common.HelmOptions{Name: "testkube", Namespace: "my-namespace", NoConfirm: true}

	if !isContextApproved(options, standaloneInstallationName) {
		t.Fatal("expected the installation to proceed without a prompt when --no-confirm is set")
	}
}
