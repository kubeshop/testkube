package parser

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

func ParseJobTemplates(cfg *config.Config) (t executor.Templates, err error) {
	t.Job, err = LoadConfigFromStringOrFile(
		cfg.TestkubeTemplateJob,
		cfg.TestkubeConfigDir,
		"job-template.yml",
		"job template",
	)
	if err != nil {
		return t, err
	}

	t.Slave, err = LoadConfigFromStringOrFile(
		cfg.TestkubeTemplateSlavePod,
		cfg.TestkubeConfigDir,
		"slave-pod-template.yml",
		"slave pod template",
	)
	if err != nil {
		return t, err
	}

	t.PVC, err = LoadConfigFromStringOrFile(
		cfg.TestkubeContainerTemplatePVC,
		cfg.TestkubeConfigDir,
		"pvc-template.yml",
		"pvc template",
	)
	if err != nil {
		return t, err
	}

	return t, nil
}

func ParseContainerTemplates(cfg *config.Config) (t executor.Templates, err error) {
	t.Job, err = LoadConfigFromStringOrFile(
		cfg.TestkubeContainerTemplateJob,
		cfg.TestkubeConfigDir,
		"job-container-template.yml",
		"job container template",
	)
	if err != nil {
		return t, err
	}

	t.Scraper, err = LoadConfigFromStringOrFile(
		cfg.TestkubeContainerTemplateScraper,
		cfg.TestkubeConfigDir,
		"job-scraper-template.yml",
		"job scraper template",
	)
	if err != nil {
		return t, err
	}

	t.PVC, err = LoadConfigFromStringOrFile(
		cfg.TestkubeContainerTemplatePVC,
		cfg.TestkubeConfigDir,
		"pvc-template.yml",
		"pvc template",
	)
	if err != nil {
		return t, err
	}

	return t, nil
}

func LoadConfigFromStringOrFile(inputString, configDir, filename, configType string) (raw string, err error) {
	var data []byte

	if inputString != "" {
		if utils.IsBase64Encoded(inputString) {
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
