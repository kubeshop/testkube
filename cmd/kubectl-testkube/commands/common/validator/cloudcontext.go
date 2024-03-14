package validator

import (
	"errors"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

func ValidateCloudContext(cfg config.Data) error {
	if cfg.ContextType != config.ContextTypeCloud {
		return nil
	}

	if cfg.CloudContext.ApiUri == "" {
		return errors.New("please provide Testkube Pro URI")
	}

	if cfg.CloudContext.ApiKey == "" {
		return errors.New("please provide Testkube Pro API token")
	}

	if cfg.CloudContext.EnvironmentId == "" {
		return errors.New("please provide Environment")
	}

	if cfg.CloudContext.OrganizationId == "" {
		return errors.New("please provide Organization")
	}

	return nil
}
