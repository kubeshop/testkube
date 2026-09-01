package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeHelmSetValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "plain dsn", value: "postgresql://user:pass@host:5432/db?sslmode=require", want: "postgresql://user:pass@host:5432/db?sslmode=require"},
		{name: "comma", value: "a,b", want: `a\,b`},
		{name: "backslash", value: `a\b`, want: `a\\b`},
		{name: "comma and backslash", value: `a\,b`, want: `a\\\,b`},
		{name: "mongo dsn with query", value: "mongodb://user:pass@host/db?replicaSet=rs0&authSource=admin", want: "mongodb://user:pass@host/db?replicaSet=rs0&authSource=admin"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, escapeHelmSetValue(tt.value))
		})
	}
}

func TestFormatHelmSetArg(t *testing.T) {
	t.Parallel()

	got := formatHelmSetArg("mongo.dsn", "mongodb://user:pass@host/db?opt=a,b")
	assert.Equal(t, `mongo.dsn=mongodb://user:pass@host/db?opt=a\,b`, got)
}

func TestRedactHelmSetValueForLog(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "[REDACTED]", redactHelmSetValueForLog("postgres.dsn", "postgresql://secret"))
	assert.Equal(t, "[REDACTED]", redactHelmSetValueForLog("credentials.masterPassword.value", "secret"))
	assert.Equal(t, "[REDACTED]", redactHelmSetValueForLog("enterpriseLicenseKey", "lic"))
	assert.Equal(t, "true", redactHelmSetValueForLog("postgres.enabled", "true"))
	assert.Equal(t, "pg-creds", redactHelmSetValueForLog("postgres.secretRef.name", "pg-creds"))
}

func TestRedactHelmArgsForLog(t *testing.T) {
	t.Parallel()

	args := []string{
		"upgrade", "--install", "rel", "chart",
		"--set", "postgres.enabled=true",
		"--set", "postgres.dsn=postgresql://user:pass@host/db",
	}

	got := redactHelmArgsForLog(args)
	assert.Equal(t, "postgres.enabled=true", got[5])
	assert.Equal(t, "postgres.dsn=[REDACTED]", got[7])
}
