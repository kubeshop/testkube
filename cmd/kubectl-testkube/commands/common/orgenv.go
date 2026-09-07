package common

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

// Gate the org/environment selectors on a TTY. The selector reads stdin, so on a
// non-interactive stream pterm returns immediately and ui.Select discards the
// error, yielding an empty id that only fails much later with an opaque message.
// Var (not func) so tests can override.
var selectorInteractive = func() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// ResolveNamedOrgAndEnv replaces the name flags on master with the ids they
// refer to, leaving levels where no name was given untouched. It never prompts,
// so it suits commands that treat a missing organization or environment as
// valid input rather than something to ask about.
//
// orgIDFallback supplies the organization for the environment lookup when the
// command itself did not set one, such as an organization already stored in the
// context. The id and name flags are mutually exclusive at the cobra level, so
// a caller cannot reach this with both set for the same level.
func ResolveNamedOrgAndEnv(apiURL, token string, master *config.Master, orgIDFallback string, skipTLS bool) error {
	if master.OrgName != "" {
		orgID, err := ResolveOrgID(apiURL, token, master.OrgName, skipTLS)
		if err != nil {
			return err
		}
		master.OrgId = orgID
	}

	if master.EnvName != "" {
		orgID := master.OrgId
		if orgID == "" {
			orgID = orgIDFallback
		}
		if orgID == "" {
			return fmt.Errorf("cannot resolve environment %q without an organization, pass --org-id or --org-name", master.EnvName)
		}

		envID, err := ResolveEnvID(apiURL, token, orgID, master.EnvName, skipTLS)
		if err != nil {
			return err
		}
		master.EnvId = envID
	}

	return nil
}

// ResolveOrgAndEnvIDs determines which organization and environment the command
// should act on, applying the same precedence at both levels: an explicit id
// wins, then a name is resolved against the Control Plane, and only when
// neither is given does the user get an interactive selector.
func ResolveOrgAndEnvIDs(apiURL, token string, master config.Master, skipTLS bool) (orgID, envID string, err error) {
	orgID, err = resolveOrgID(apiURL, token, master, skipTLS)
	if err != nil {
		return "", "", err
	}

	envID, err = resolveEnvID(apiURL, token, orgID, master, skipTLS)
	if err != nil {
		return "", "", err
	}

	return orgID, envID, nil
}

func resolveOrgID(apiURL, token string, master config.Master, skipTLS bool) (string, error) {
	if master.OrgId != "" {
		return master.OrgId, nil
	}

	if master.OrgName != "" {
		return ResolveOrgID(apiURL, token, master.OrgName, skipTLS)
	}

	if !selectorInteractive() {
		return "", fmt.Errorf("no organization selected and the terminal is not interactive, pass --org-id or --org-name")
	}

	orgID, _, err := UiGetOrganizationId(apiURL, token, skipTLS)
	return orgID, err
}

func resolveEnvID(apiURL, token, orgID string, master config.Master, skipTLS bool) (string, error) {
	if master.EnvId != "" {
		return master.EnvId, nil
	}

	if master.EnvName != "" {
		return ResolveEnvID(apiURL, token, orgID, master.EnvName, skipTLS)
	}

	if !selectorInteractive() {
		return "", fmt.Errorf("no environment selected and the terminal is not interactive, pass --env-id or --env-name")
	}

	envID, _, err := UiGetEnvironmentID(apiURL, token, orgID, skipTLS)
	return envID, err
}
