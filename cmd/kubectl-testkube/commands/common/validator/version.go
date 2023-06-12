package validator

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

var ErrOldClientVersion = fmt.Errorf("client version is older than api version, please upgrade")

// PersistentPreRunVersionCheck will check versions based on commands client
func PersistentPreRunVersionCheck(cmd *cobra.Command, clientVersion string) {
	// version validation
	// if client version is less than server version show warning
	client, _, err := common.GetClient(cmd)
	if err != nil {
		return
	}

	info, err := client.GetServerInfo()
	if err != nil {
		// omit check of versions if we can't get server info
		// e.g. when there is not cloud token yet
		ui.Debug(err.Error())
		return
	}

	err = ValidateVersions(info.Version, clientVersion)
	if err != nil {
		ui.Warn(err.Error())
	} else if err == ErrOldClientVersion {
		ui.Warn("Your Testkube API version is newer than your `kubectl testkube` plugin")
		ui.Info("Testkube API version", info.Version)
		ui.Info("Testkube kubectl plugin client", clientVersion)
		ui.Info("It's recommended to upgrade client to version close to API server version")
		ui.NL()
	}
}

// ValidateVersions will check if kubectl plugins MINOR version is greater or equal Testkube API version
func ValidateVersions(apiVersionString, clientVersionString string) error {
	if apiVersionString == "" {
		return fmt.Errorf("server version not set")
	}

	apiMinorVersion := TrimPatchVersion(apiVersionString)
	apiVersion, err := semver.NewVersion(apiMinorVersion)
	if err != nil {
		return fmt.Errorf("parsing server version '%s': %w", apiVersionString, err)
	}

	if clientVersionString == "" {
		return fmt.Errorf("client version not set")
	}

	clientMinorVersion := TrimPatchVersion(clientVersionString)
	clientVersion, err := semver.NewVersion(clientMinorVersion)
	if err != nil {
		return fmt.Errorf("parsing client version %s: %w", clientVersionString, err)
	}

	if clientVersion.LessThan(apiVersion) {
		return ErrOldClientVersion
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
