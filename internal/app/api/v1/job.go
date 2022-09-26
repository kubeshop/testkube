package v1

import (
	"encoding/base64"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type JobTemplates struct {
	Job string
}

func NewJobTemplatesFromEnv(env string) (*JobTemplates, error) {
	j := JobTemplates{}
	err := envconfig.Process(env, &j)
	if err != nil {
		return nil, err
	}
	templates := []*string{&j.Job}
	for i := range templates {
		if *templates[i] != "" {
			dataDecoded, err := base64.StdEncoding.DecodeString(*templates[i])
			if err != nil {
				return nil, errors.WithMessage(err, "error decoding base64 string")
			}

			*templates[i] = string(dataDecoded)
		}
	}

	return &j, nil
}
