package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	cloudAPINameLabel        = "app.kubernetes.io/name=testkube-cloud-api"
	cloudAPIContainerName    = "testkube-cloud-api"
	envEnterpriseLicense     = "ENTERPRISE_LICENSE_KEY"
	envEnterpriseLicenseFile = "ENTERPRISE_LICENSE_FILE"
	envCredentialsMaster     = "CREDENTIALS_MASTER_PASSWORD"
	envPostgresURL           = "API_POSTGRES_URL"
	envMongoDSN              = "API_MONGO_DSN"
	envMongoDB               = "API_MONGO_DB"
	envMongoReadPref         = "API_MONGO_READ_PREFERENCE"
	envDatabaseUsername      = "DATABASE_USERNAME"
	envDatabasePassword      = "DATABASE_PASSWORD"
	envDatabaseHost          = "DATABASE_HOST"
	envDatabaseName          = "DATABASE_NAME"
	envSSLCertDir            = "SSL_CERT_DIR"
)

type discoveredConfig struct {
	HelmSet      map[string]string
	Deployment   string
	SourceDetail string
}

type kubeDeployment struct {
	Items []kubeDeploymentItem `json:"items"`
}

type kubeDeploymentItem struct {
	Metadata kubeObjectMeta     `json:"metadata"`
	Spec     kubeDeploymentSpec `json:"spec"`
}

type kubeObjectMeta struct {
	Name string `json:"name"`
}

type kubeDeploymentSpec struct {
	Template kubePodTemplateSpec `json:"template"`
}

type kubePodTemplateSpec struct {
	Spec kubePodSpec `json:"spec"`
}

type kubePodSpec struct {
	Containers []kubeContainer `json:"containers"`
	Volumes    []kubeVolume    `json:"volumes"`
}

type kubeContainer struct {
	Name         string            `json:"name"`
	Env          []kubeEnvVar      `json:"env"`
	VolumeMounts []kubeVolumeMount `json:"volumeMounts"`
}

type kubeEnvVar struct {
	Name      string          `json:"name"`
	Value     string          `json:"value"`
	ValueFrom *kubeEnvVarFrom `json:"valueFrom"`
}

type kubeEnvVarFrom struct {
	SecretKeyRef *kubeSecretKeyRef `json:"secretKeyRef"`
}

type kubeSecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type kubeVolume struct {
	Name   string            `json:"name"`
	Secret *kubeVolumeSecret `json:"secret"`
}

type kubeVolumeSecret struct {
	SecretName string `json:"secretName"`
}

type kubeVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	SubPath   string `json:"subPath"`
}

// DiscoverConfig reads the Testkube Enterprise cloud-api deployment in the target
// namespace and maps its env/volume wiring to testkube-usage-export Helm values.
func DiscoverConfig(opts Options) (discoveredConfig, *common.CLIError) {
	kubectlPath, cliErr := common.LookupKubectlPath()
	if cliErr != nil {
		return discoveredConfig{}, cliErr
	}

	args := kubectlBaseArgs(opts, "get", "deploy", "-l", cloudAPINameLabel, "-o", "json")
	raw, cliErr := common.RunKubectlCommand(kubectlPath, args)
	if cliErr != nil {
		return discoveredConfig{}, cliErr
	}

	var list kubeDeployment
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return discoveredConfig{}, common.NewCLIError(
			common.TKErrKubectlCommandFailed,
			"Failed to parse cloud-api deployment",
			"Verify kubectl access to the target namespace",
			err,
		)
	}

	deploy := pickCloudAPIDeployment(list.Items)
	if deploy == nil {
		args = kubectlBaseArgs(opts, "get", "deploy", config.EnterpriseApiName, "-o", "json")
		raw, cliErr = common.RunKubectlCommand(kubectlPath, args)
		if cliErr != nil {
			return discoveredConfig{}, common.NewCLIError(
				common.TKErrKubectlCommandFailed,
				"Testkube Enterprise cloud-api deployment not found",
				fmt.Sprintf("Install enterprise in namespace %q or pass -f values.yaml with --no-auto-config", opts.Namespace),
				fmt.Errorf("no deployment labeled %s", cloudAPINameLabel),
			)
		}
		var single kubeDeploymentItem
		if err := json.Unmarshal([]byte(raw), &single); err != nil {
			return discoveredConfig{}, common.NewCLIError(
				common.TKErrKubectlCommandFailed,
				"Failed to parse cloud-api deployment",
				"Verify kubectl access to the target namespace",
				err,
			)
		}
		deploy = &single
	}

	container := pickCloudAPIContainer(deploy.Spec.Template.Spec.Containers)
	if container == nil {
		return discoveredConfig{}, common.NewCLIError(
			common.TKErrKubectlCommandFailed,
			"cloud-api container not found",
			"Expected a container named testkube-cloud-api on the enterprise deployment",
			fmt.Errorf("deployment %q has no cloud-api container", deploy.Metadata.Name),
		)
	}

	helmSet, err := helmSetFromCloudAPI(container.Env, deploy.Spec.Template.Spec.Volumes, container.VolumeMounts)
	if err != nil {
		return discoveredConfig{}, common.NewCLIError(
			common.TKErrInvalidRuntimeParameter,
			"Could not derive usage-export config from cloud-api",
			"Provide -f values.yaml with --no-auto-config, or ensure cloud-api has database and license env configured",
			err,
		)
	}

	return discoveredConfig{
		HelmSet:      helmSet,
		Deployment:   deploy.Metadata.Name,
		SourceDetail: fmt.Sprintf("deployment/%s", deploy.Metadata.Name),
	}, nil
}

func pickCloudAPIDeployment(items []kubeDeploymentItem) *kubeDeploymentItem {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if items[i].Metadata.Name == config.EnterpriseApiName {
			return &items[i]
		}
	}
	return &items[0]
}

func pickCloudAPIContainer(containers []kubeContainer) *kubeContainer {
	for i := range containers {
		if containers[i].Name == cloudAPIContainerName {
			return &containers[i]
		}
	}
	if len(containers) == 0 {
		return nil
	}
	return &containers[0]
}

func helmSetFromCloudAPI(envVars []kubeEnvVar, volumes []kubeVolume, mounts []kubeVolumeMount) (map[string]string, error) {
	env := envVarsByName(envVars)
	out := map[string]string{}

	if err := applyPostgresHelmSet(env, out); err != nil {
		return nil, err
	}
	if _, ok := out["postgres.enabled"]; !ok {
		if err := applyMongoHelmSet(env, out); err != nil {
			return nil, err
		}
	}
	if _, ok := out["postgres.enabled"]; !ok {
		if _, ok := out["mongo.enabled"]; !ok {
			return nil, fmt.Errorf("cloud-api has neither postgres nor mongo configuration")
		}
	}

	if err := applyCredentialsHelmSet(env, out); err != nil {
		return nil, err
	}
	if err := applyLicenseHelmSet(env, out); err != nil {
		return nil, err
	}
	applyCustomCAHelmSet(env, volumes, mounts, out)

	return out, nil
}

func envVarsByName(envVars []kubeEnvVar) map[string]kubeEnvVar {
	out := make(map[string]kubeEnvVar, len(envVars))
	for _, e := range envVars {
		out[e.Name] = e
	}
	return out
}

func applyPostgresHelmSet(env map[string]kubeEnvVar, out map[string]string) error {
	if ref, ok := secretRef(env, envPostgresURL); ok {
		out["postgres.enabled"] = "true"
		out["postgres.dsnSecretRef"] = ref.name
		out["postgres.dsnSecretKey"] = ref.key
		return nil
	}
	if literal, ok := literalValue(env, envPostgresURL); ok && !isPostgresURLTemplate(literal) {
		out["postgres.enabled"] = "true"
		out["postgres.dsn"] = literal
		return nil
	}

	pgSecret, ok := postgresComponentSecret(env)
	if !ok {
		return nil
	}

	out["postgres.enabled"] = "true"
	out["postgres.secretRef.name"] = pgSecret
	if ref, ok := secretRef(env, envDatabaseUsername); ok {
		out["postgres.secretRef.usernameKey"] = ref.key
	} else if v, ok := literalValue(env, envDatabaseUsername); ok {
		out["postgres.username"] = v
	}
	if ref, ok := secretRef(env, envDatabasePassword); ok {
		out["postgres.secretRef.passwordKey"] = ref.key
	}
	if ref, ok := secretRef(env, envDatabaseHost); ok {
		out["postgres.secretRef.endpointKey"] = ref.key
	} else if v, ok := literalValue(env, envDatabaseHost); ok {
		out["postgres.endpoint"] = v
	}
	if v, ok := literalValue(env, envDatabaseName); ok {
		out["postgres.database"] = v
	}
	return nil
}

func postgresComponentSecret(env map[string]kubeEnvVar) (string, bool) {
	for _, name := range []string{envDatabaseUsername, envDatabasePassword, envDatabaseHost} {
		if ref, ok := secretRef(env, name); ok {
			return ref.name, true
		}
	}
	return "", false
}

func applyMongoHelmSet(env map[string]kubeEnvVar, out map[string]string) error {
	if ref, ok := secretRef(env, envMongoDSN); ok {
		out["mongo.enabled"] = "true"
		out["mongo.dsnSecretRef"] = ref.name
	} else if literal, ok := literalValue(env, envMongoDSN); ok {
		out["mongo.enabled"] = "true"
		out["mongo.dsn"] = literal
	} else {
		return nil
	}
	if v, ok := literalValue(env, envMongoDB); ok {
		out["mongo.database"] = v
	}
	if v, ok := literalValue(env, envMongoReadPref); ok {
		out["mongo.readPreference"] = v
	}
	return nil
}

func applyCredentialsHelmSet(env map[string]kubeEnvVar, out map[string]string) error {
	if ref, ok := secretRef(env, envCredentialsMaster); ok {
		out["credentials.masterPassword.secretKeyRef.name"] = ref.name
		out["credentials.masterPassword.secretKeyRef.key"] = ref.key
		return nil
	}
	if v, ok := literalValue(env, envCredentialsMaster); ok {
		out["credentials.masterPassword.value"] = v
		ui.Warn("Auto-config copied inline CREDENTIALS_MASTER_PASSWORD from cloud-api; prefer secret refs in production")
		return nil
	}
	return fmt.Errorf("CREDENTIALS_MASTER_PASSWORD not found on cloud-api deployment")
}

func applyLicenseHelmSet(env map[string]kubeEnvVar, out map[string]string) error {
	if _, hasFile := env[envEnterpriseLicenseFile]; hasFile {
		if _, hasKey := secretRef(env, envEnterpriseLicense); !hasKey {
			if _, hasLiteral := literalValue(env, envEnterpriseLicense); !hasLiteral {
				return fmt.Errorf("offline license file installs require ENTERPRISE_LICENSE_KEY for usage export")
			}
		}
	}
	if ref, ok := secretRef(env, envEnterpriseLicense); ok {
		out["enterpriseLicenseSecretRef"] = ref.name
		return nil
	}
	if v, ok := literalValue(env, envEnterpriseLicense); ok {
		out["enterpriseLicenseKey"] = v
		ui.Warn("Auto-config copied inline ENTERPRISE_LICENSE_KEY from cloud-api; prefer enterpriseLicenseSecretRef in production")
		return nil
	}
	return fmt.Errorf("ENTERPRISE_LICENSE_KEY not found on cloud-api deployment")
}

func applyCustomCAHelmSet(env map[string]kubeEnvVar, volumes []kubeVolume, mounts []kubeVolumeMount, out map[string]string) {
	if _, ok := literalValue(env, envSSLCertDir); !ok {
		return
	}
	volumeSecrets := map[string]string{}
	for _, vol := range volumes {
		if vol.Secret != nil && vol.Secret.SecretName != "" {
			volumeSecrets[vol.Name] = vol.Secret.SecretName
		}
	}
	for _, mount := range mounts {
		if secretName, ok := volumeSecrets[mount.Name]; ok && mount.SubPath != "" {
			out["customCaSecretRef"] = secretName
			out["customCaSecretKey"] = mount.SubPath
			return
		}
	}
}

type secretRefValue struct {
	name string
	key  string
}

func secretRef(env map[string]kubeEnvVar, name string) (secretRefValue, bool) {
	e, ok := env[name]
	if !ok || e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
		return secretRefValue{}, false
	}
	ref := e.ValueFrom.SecretKeyRef
	if strings.TrimSpace(ref.Name) == "" || strings.TrimSpace(ref.Key) == "" {
		return secretRefValue{}, false
	}
	return secretRefValue{name: ref.Name, key: ref.Key}, true
}

func literalValue(env map[string]kubeEnvVar, name string) (string, bool) {
	e, ok := env[name]
	if !ok {
		return "", false
	}
	v := strings.TrimSpace(e.Value)
	if v == "" {
		return "", false
	}
	return v, true
}

func isPostgresURLTemplate(dsn string) bool {
	return strings.Contains(dsn, "$(")
}

func mergeHelmSets(base, overrides map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overrides))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

func shouldAutoConfig(valuesFiles []string, autoConfig, noAutoConfig bool) bool {
	if noAutoConfig {
		return false
	}
	if autoConfig {
		return true
	}
	return len(valuesFiles) == 0
}

func applyAutoConfig(opts *Options, autoConfig, noAutoConfig bool) *common.CLIError {
	if !shouldAutoConfig(opts.ValuesFiles, autoConfig, noAutoConfig) {
		return nil
	}

	discovered, cliErr := DiscoverConfig(*opts)
	if cliErr != nil {
		return cliErr
	}

	opts.HelmSet = mergeHelmSets(discovered.HelmSet, opts.HelmSet)
	opts.CreateNamespace = false

	ui.Info("Auto-configured usage export from", discovered.SourceDetail)
	if ui.IsVerbose() {
		for k, v := range discovered.HelmSet {
			ui.Debug(fmt.Sprintf("  %s=%s", k, v))
		}
	}
	return nil
}
