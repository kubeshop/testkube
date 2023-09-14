package parser

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/log"
)

func ParseJobTemplate(cfg *config.Config) (template string, err error) {
	template, err = LoadConfigFromStringOrFile(
		cfg.TestkubeTemplateJob,
		cfg.TestkubeConfigDir,
		"job-template.yml",
		"job template",
	)
	if err != nil {
		return "", err
	}

	return template, nil
}

func IsBase64Encoded(base64Val string) bool {
	decoded, err := base64.StdEncoding.DecodeString(base64Val)
	if err != nil {
		return false
	}

	encoded := base64.StdEncoding.EncodeToString(decoded)
	return base64Val == encoded
}

func LoadConfigFromStringOrFile(inputString, configDir, filename, configType string) (raw string, err error) {
	var data []byte

	if inputString != "" {
		if IsBase64Encoded(inputString) {
			data, err = base64.StdEncoding.DecodeString(inputString)
			if err != nil {
				return "", errors.Wrapf(err, "error decoding %s from base64", configType)
			}
			raw = string(data)
			log.DefaultLogger.Infof("parsed %s from base64 env var", configType)
		} else {
			raw = inputString
			log.DefaultLogger.Infof("parsed %s from plain env var", configType)
		}
	} else if f, err := os.Open(filepath.Join(configDir, filename)); err == nil {
		data, err = io.ReadAll(f)
		if err != nil {
			return "", errors.Wrapf(err, "error reading file %s from config dir %s", filename, configDir)
		}
		raw = string(data)
		log.DefaultLogger.Infof("loaded %s from file %s", configType, filepath.Join(configDir, filename))
	} else {
		log.DefaultLogger.Infof("no %s config found", configType)
	}

	return raw, nil
}
