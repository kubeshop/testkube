// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package env

import (
	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	UseProxyValue = false
)

type envObjectStorageConfig struct {
	Endpoint        string `envconfig:"TK_OS_ENDPOINT"`
	AccessKeyID     string `envconfig:"TK_OS_ACCESSKEY"`
	SecretAccessKey string `envconfig:"TK_OS_SECRETKEY"`
	Region          string `envconfig:"TK_OS_REGION"`
	Token           string `envconfig:"TK_OS_TOKEN"`
	Bucket          string `envconfig:"TK_OS_BUCKET"`
	Ssl             bool   `envconfig:"TK_OS_SSL" default:"false"`
	SkipVerify      bool   `envconfig:"TK_OS_SSL_SKIP_VERIFY" default:"false"`
	CertFile        string `envconfig:"TK_OS_CERT_FILE"`
	KeyFile         string `envconfig:"TK_OS_KEY_FILE"`
	CAFile          string `envconfig:"TK_OS_CA_FILE"`
}

type envCloudConfig struct {
	Url         string `envconfig:"TK_C_URL"`
	ApiKey      string `envconfig:"TK_C_KEY"`
	SkipVerify  bool   `envconfig:"TK_C_SKIP_VERIFY" default:"false"`
	TlsInsecure bool   `envconfig:"TK_C_TLS_INSECURE" default:"false"`
}

type envExecutionConfig struct {
	WorkflowName string `envconfig:"TK_WF"`
	Id           string `envconfig:"TK_EX"`
}

type envSystemConfig struct {
	Debug     string `envconfig:"DEBUG"`
	Ref       string `envconfig:"TK_REF"`
	Namespace string `envconfig:"TK_NS"`
	Ip        string `envconfig:"TK_IP"`
}

type envImagesConfig struct {
	Init    string `envconfig:"TK_IMG_INIT"`
	Toolkit string `envconfig:"TK_IMG_TOOLKIT"`
}

type envConfig struct {
	System        envSystemConfig
	ObjectStorage envObjectStorageConfig
	Cloud         envCloudConfig
	Execution     envExecutionConfig
	Images        envImagesConfig
}

var cfg envConfig
var cfgLoaded = false

func Config() *envConfig {
	if !cfgLoaded {
		err := envconfig.Process("", &cfg.System)
		ui.ExitOnError("configuring environment", err)
		err = envconfig.Process("", &cfg.ObjectStorage)
		ui.ExitOnError("configuring environment", err)
		err = envconfig.Process("", &cfg.Cloud)
		ui.ExitOnError("configuring environment", err)
		err = envconfig.Process("", &cfg.Execution)
		ui.ExitOnError("configuring environment", err)
		err = envconfig.Process("", &cfg.Images)
		ui.ExitOnError("configuring environment", err)
	}
	cfgLoaded = true
	return &cfg
}

func Debug() bool {
	return Config().System.Debug == "1"
}

func CloudEnabled() bool {
	return Config().Cloud.ApiKey != ""
}

func UseProxy() bool {
	return UseProxyValue
}

func Ref() string {
	return Config().System.Ref
}

func Namespace() string {
	return Config().System.Namespace
}

func IP() string {
	return Config().System.Ip
}

func WorkflowName() string {
	return Config().Execution.WorkflowName
}

func ExecutionId() string {
	return Config().Execution.Id
}
