package parser

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

func LoadConfigFromStringOrFile(inputString, configDir, filename, configType string) (raw string, err error) {
	var data []byte

	if inputString != "" {
		if utils.IsBase64Encoded(inputString) {
			data, err = base64.StdEncoding.DecodeString(inputString)
			if err != nil {
				return "", errors.Wrapf(err, "error decoding %s from base64", configType)
			}
			raw = string(data)
			log.DefaultLogger.Debugf("parsed %s from base64 env var", configType)
		} else {
			raw = inputString
			log.DefaultLogger.Debugf("parsed %s from plain env var", configType)
		}
	} else if raw, err = LoadConfigFromFile(configDir, filename, configType); err != nil {
		return "", err
	} else if raw == "" {
		log.DefaultLogger.Warnf("no %s config found", configType)
	}

	return raw, nil
}

func LoadConfigFromFile(configDir, filename, configType string) (raw string, err error) {
	f, err := os.Open(filepath.Join(configDir, filename))
	if err != nil {
		return "", nil
	}

	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", errors.Wrapf(err, "error reading file %s from config dir %s", filename, configDir)
	}

	log.DefaultLogger.Debugf("loaded %s from file %s", configType, filepath.Join(configDir, filename))
	return string(data), nil
}
