package validator

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func PersistentPreRunVersionCheck(cmd *cobra.Command, version string) {
	// version validation
	// if client version is less than server version show warning
	client, _ := common.GetClient(cmd)

	err := ValidateVersions(client, version)
	if err != nil {
		ui.Warn(err.Error())
	}
}

func ValidateVersions(c apiclient.Client, version string) error {
	info, err := c.GetServerInfo()
	if err != nil {
		return fmt.Errorf("getting server info: %w", err)
	}

	if info.Version == "" {
		return fmt.Errorf("server version not set")
	}

	serverVersion, err := semver.NewVersion(info.Version)
	if err != nil {
		return fmt.Errorf("parsing server version '%s': %w", info.Version, err)
	}

	if version == "" {
		return fmt.Errorf("client version not set")
	}

	clientVersion, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("parsing client version %s: %w", version, err)
	}

	if clientVersion.LessThan(serverVersion) {
		ui.Warn("Your Testkube API version is newer than your `kubectl testkube` plugin")
		ui.Info("Testkube API version", serverVersion.String())
		ui.Info("Testkube kubectl plugin client", clientVersion.String())
		ui.Info("It's recommended to upgrade client to version close to API server version")
		ui.NL()
	}

	return nil
}
