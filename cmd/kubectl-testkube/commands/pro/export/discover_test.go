package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelmSetFromCloudAPI_PostgresSecretRef(t *testing.T) {
	t.Parallel()

	env := []kubeEnvVar{
		{Name: envDatabaseUsername, ValueFrom: secretRefFrom("pg-creds", "username")},
		{Name: envDatabasePassword, ValueFrom: secretRefFrom("pg-creds", "password")},
		{Name: envDatabaseHost, ValueFrom: secretRefFrom("pg-creds", "host")},
		{Name: envDatabaseName, Value: "backend"},
		{Name: envPostgresURL, Value: "postgresql://$(DATABASE_USERNAME):$(DATABASE_PASSWORD)@$(DATABASE_HOST)/$(DATABASE_NAME)?sslmode=require"},
		{Name: envCredentialsMaster, ValueFrom: secretRefFrom("testkube-credentials-master", "password")},
		{Name: envEnterpriseLicense, ValueFrom: secretRefFrom("testkube-enterprise-license", "LICENSE_KEY")},
	}

	got, err := helmSetFromCloudAPI(env, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "true", got["postgres.enabled"])
	assert.Equal(t, "pg-creds", got["postgres.secretRef.name"])
	assert.Equal(t, "username", got["postgres.secretRef.usernameKey"])
	assert.Equal(t, "password", got["postgres.secretRef.passwordKey"])
	assert.Equal(t, "host", got["postgres.secretRef.endpointKey"])
	assert.Equal(t, "backend", got["postgres.database"])
	assert.Equal(t, "testkube-credentials-master", got["credentials.masterPassword.secretKeyRef.name"])
	assert.Equal(t, "testkube-enterprise-license", got["enterpriseLicenseSecretRef"])
}

func TestHelmSetFromCloudAPI_PostgresDSNSecret(t *testing.T) {
	t.Parallel()

	env := []kubeEnvVar{
		{Name: envPostgresURL, ValueFrom: secretRefFrom("pg-dsn", "API_POSTGRES_URL")},
		{Name: envCredentialsMaster, ValueFrom: secretRefFrom("testkube-credentials-master", "password")},
		{Name: envEnterpriseLicense, ValueFrom: secretRefFrom("testkube-enterprise-license", "LICENSE_KEY")},
	}

	got, err := helmSetFromCloudAPI(env, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "true", got["postgres.enabled"])
	assert.Equal(t, "pg-dsn", got["postgres.dsnSecretRef"])
	assert.Equal(t, "API_POSTGRES_URL", got["postgres.dsnSecretKey"])
}

func TestHelmSetFromCloudAPI_MongoSecretRef(t *testing.T) {
	t.Parallel()

	env := []kubeEnvVar{
		{Name: envMongoDSN, ValueFrom: secretRefFrom("mongo-dsn", "MONGO_DSN")},
		{Name: envMongoDB, Value: "testkubeEnterpriseDB"},
		{Name: envCredentialsMaster, ValueFrom: secretRefFrom("testkube-credentials-master", "password")},
		{Name: envEnterpriseLicense, ValueFrom: secretRefFrom("testkube-enterprise-license", "LICENSE_KEY")},
	}

	got, err := helmSetFromCloudAPI(env, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "true", got["mongo.enabled"])
	assert.Equal(t, "mongo-dsn", got["mongo.dsnSecretRef"])
	assert.Equal(t, "testkubeEnterpriseDB", got["mongo.database"])
}

func TestHelmSetFromCloudAPI_CustomCA(t *testing.T) {
	t.Parallel()

	env := []kubeEnvVar{
		{Name: envPostgresURL, Value: "postgresql://u:p@db:5432/t"},
		{Name: envCredentialsMaster, ValueFrom: secretRefFrom("testkube-credentials-master", "password")},
		{Name: envEnterpriseLicense, ValueFrom: secretRefFrom("testkube-enterprise-license", "LICENSE_KEY")},
		{Name: envSSLCertDir, Value: "/etc/testkube/certs"},
	}
	volumes := []kubeVolume{{Name: "custom-ca", Secret: &kubeVolumeSecret{SecretName: "custom-ca"}}}
	mounts := []kubeVolumeMount{{Name: "custom-ca", SubPath: "ca.crt"}}

	got, err := helmSetFromCloudAPI(env, volumes, mounts)
	require.NoError(t, err)

	assert.Equal(t, "custom-ca", got["customCaSecretRef"])
	assert.Equal(t, "ca.crt", got["customCaSecretKey"])
}

func TestHelmSetFromCloudAPI_OfflineLicenseFileWithoutKey(t *testing.T) {
	t.Parallel()

	env := []kubeEnvVar{
		{Name: envPostgresURL, Value: "postgresql://u:p@db:5432/t"},
		{Name: envCredentialsMaster, ValueFrom: secretRefFrom("testkube-credentials-master", "password")},
		{Name: envEnterpriseLicenseFile, Value: "/testkube/license.lic"},
	}

	_, err := helmSetFromCloudAPI(env, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "offline license")
}

func TestShouldAutoConfig(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldAutoConfig(nil, false, false))
	assert.False(t, shouldAutoConfig([]string{"values.yaml"}, false, false))
	assert.True(t, shouldAutoConfig([]string{"values.yaml"}, true, false))
	assert.False(t, shouldAutoConfig(nil, true, true))
}

func TestMergeHelmSets(t *testing.T) {
	t.Parallel()

	got := mergeHelmSets(map[string]string{"postgres.enabled": "true", "usageExport.weeks": "4"}, map[string]string{"usageExport.weeks": "8"})
	assert.Equal(t, "true", got["postgres.enabled"])
	assert.Equal(t, "8", got["usageExport.weeks"])
}

func secretRefFrom(name, key string) *kubeEnvVarFrom {
	return &kubeEnvVarFrom{SecretKeyRef: &kubeSecretKeyRef{Name: name, Key: key}}
}
