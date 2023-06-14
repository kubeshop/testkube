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
		return errors.New("please provide Testkube Cloud URI")
	}

	if cfg.CloudContext.ApiKey == "" {
		return errors.New("please provide Testkube Cloud API token")
	}

	if cfg.CloudContext.Environment == "" {
		return errors.New("please provide environment")
	}

	if cfg.CloudContext.Organization == "" {
		return errors.New("please provide organization")
	}

	return nil
}
