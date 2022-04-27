package validator

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

// PersistentPreRunVersionCheck will check versions based on commands client
func PersistentPreRunVersionCheck(cmd *cobra.Command, version string) {
	// version validation
	// if client version is less than server version show warning
	client, _ := common.GetClient(cmd)

	err := ValidateVersions(client, version)
	if err != nil {
		ui.Warn(err.Error())
	}
}

// ValidateVersions will check if kubectl plugins MINOR version is greater or equal Testkube API version
func ValidateVersions(c apiclient.Client, version string) error {
	info, err := c.GetServerInfo()
	if err != nil {
		return fmt.Errorf("getting server info: %w", err)
	}

	if info.Version == "" {
		return fmt.Errorf("server version not set")
	}

	apiMinorVersion := TrimPatchVersion(info.Version)
	apiVersion, err := semver.NewVersion(apiMinorVersion)
	if err != nil {
		return fmt.Errorf("parsing server version '%s': %w", info.Version, err)
	}

	if version == "" {
		return fmt.Errorf("client version not set")
	}

	minorVersion := TrimPatchVersion(version)
	clientVersion, err := semver.NewVersion(minorVersion)
	if err != nil {
		return fmt.Errorf("parsing client version %s: %w", version, err)
	}

	if clientVersion.LessThan(apiVersion) {
		ui.Warn("Your Testkube API version is newer than your `kubectl testkube` plugin")
		ui.Info("Testkube API version", apiVersion.String())
		ui.Info("Testkube kubectl plugin client", clientVersion.String())
		ui.Info("It's recommended to upgrade client to version close to API server version")
		ui.NL()
	}

	return nil
}

// TrimPatchVersion will trim
func TrimPatchVersion(version string) string {
	re := regexp.MustCompile("([0-9]+).([0-9]+).([0-9]+)(.*)")
	parts := re.FindStringSubmatch(version)
	if len(parts) == 5 {
		return fmt.Sprintf("%s.%s.%s", parts[1], parts[2], "0")
	}

	return version
}
