package loader

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

type License struct {
	EnterpriseOfflineActivation     bool   `envconfig:"ENTERPRISE_OFFLINE_ACTIVATION" default:"false"`
	EnterpriseLicenseKey            string `envconfig:"ENTERPRISE_LICENSE_KEY"`
	EnterpriseLicenseKeyPath        string `envconfig:"ENTERPRISE_LICENSE_KEY_PATH" default:"/testkube/license.key"`
	EnterpriseLicenseFile           string `envconfig:"ENTERPRISE_LICENSE_FILE"`
	EnterpriseLicenseFilePath       string `envconfig:"ENTERPRISE_LICENSE_FILE_PATH" default:"/testkube/license.lic"`
	EnterpriseLicenseFileEncryption string `envconfig:"ENTERPRISE_LICENSE_FILE_ENCRYPTION"`
	EnterpriseLicenseName           string `envconfig:"ENTERPRISE_LICENSE_NAME"`
}

const DefaultSecretName = "testkube-enterprise-license"

func GetLicenseConfig(namespace, secretName string) (l License, err error) {
	if secretName == "" {
		secretName = DefaultSecretName
	}

	// get control plane api pod envs
	envs, err := common.KubectlGetPodEnvs("-l app.kubernetes.io/name=testkube-cloud-api", namespace)
	if err != nil {
		return l, err
	}
	ui.ExitOnError("getting env variables from pods", err)

	if offlineActivation, ok := envs["ENTERPRISE_OFFLINE_ACTIVATION"]; ok && offlineActivation == "true" {
		l.EnterpriseOfflineActivation = true
	}

	if f, ok := envs["ENTERPRISE_LICENSE_FILE"]; ok && f != "" {
		l.EnterpriseLicenseFile = f
	}

	if k, ok := envs["ENTERPRISE_LICENSE_KEY_PATH"]; ok && k != "" {
		l.EnterpriseLicenseKeyPath = k
	}

	if k, ok := envs["ENTERPRISE_LICENSE_FILE_PATH"]; ok && k != "" {
		l.EnterpriseLicenseFilePath = k
	}

	if k, ok := envs["ENTERPRISE_LICENSE_KEY"]; ok && k != "" {
		l.EnterpriseLicenseKey = k
	}

	// try to load from secret - there is no easy way of just stream the key content
	secrets, err := common.KubectlGetSecret(secretName, namespace)
	ui.WarnOnError("getting secrets from pods", err)

	const (
		keySecretKeyName  = "LICENSE_KEY"
		fileSecretKeyName = "license.lic"
	)

	// If no direct key value provided try to figure out it from secret mapped as file
	if l.EnterpriseLicenseKey == "" {
		// try to load from secret - there is no easy way of just stream the key content
		if k, ok := secrets[keySecretKeyName]; ok {
			l.EnterpriseLicenseKey = k
		}
	}

	// If no direct file value provided try to figure out it from secret mapped as file
	if l.EnterpriseLicenseFile == "" {
		if k, ok := secrets[fileSecretKeyName]; ok {
			l.EnterpriseLicenseFile = k
		}
	}

	return l, err
}
