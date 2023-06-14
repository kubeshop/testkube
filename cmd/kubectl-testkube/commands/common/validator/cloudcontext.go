package validator

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

type ContextErr struct {
	Msg string
}

func (e ContextErr) Error() string {
	return e.Msg
}

func ValidateCloudContext(cfg config.Data) error {
	if cfg.ContextType != config.ContextTypeCloud {
		return nil
	}

	if cfg.CloudContext.ApiUri == "" {
		return ContextErr{"please provide Testkube Cloud URI"}
	}

	if cfg.CloudContext.ApiKey == "" {
		return ContextErr{"please provide Testkube Cloud API token"}
	}

	if cfg.CloudContext.Environment == "" {
		return ContextErr{"please provide environment"}
	}

	if cfg.CloudContext.Organization == "" {
		return ContextErr{"please provide organization"}
	}

	return nil
}
