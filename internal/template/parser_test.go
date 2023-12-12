package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/internal/config"
)

// Test the ParseJobTemplate function
func TestParseJobTemplate(t *testing.T) {
	pwd, _ := os.Getwd()
	t.Setenv("TESTKUBE_CONFIG_DIR", filepath.Join(pwd, "../../config"))

	assertion := require.New(t)
	cfg, err := config.Get()
	assertion.NoError(err)

	templates, err := ParseJobTemplates(cfg)

	assertion.NoError(err)
	// t.Log(jobTemplate)
	assertion.NotEmpty(templates.Job)
	// t.Log(slavePodTemplate)
	assertion.NotEmpty(templates.Slave)
	// t.Log(pvcTemplate)
	assertion.NotEmpty(templates.PVC)
}
