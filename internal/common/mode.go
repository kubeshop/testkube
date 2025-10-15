package common

import "errors"

const (
	ModeStandalone = "standalone"
	ModeAgent      = "agent"

	// note: Testkube Agent should not really have knowledge about organizations and environments.
	// It appears that over time this has seeped into this codebase. Often empty strings are used
	// to make function works. These constants make usages explicit to determine if we can and should get rid of this.

	// WARNING: Be extremely cautious renaming this from an empty string to anything else. I've discovered that
	// these empty strings might down the line be replaced by a defaultEnvironment which is set through ProContext.
	StandaloneEnvironment      = ""
	StandaloneEnvironmentSlug  = ""
	StandaloneOrganization     = ""
	StandaloneOrganizationSlug = ""
	StandaloneRunner           = "default-agent"
	StandaloneRunnerName       = "Testkube Agent"
)

var ErrNotSupported = errors.New("feature is not supported in standalone mode")
