package webhooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #4050: cat payload.json | testkube create webhook --payload-template -
func TestNewCreateWebhookOptionsFromFlags_ReadsThePayloadTemplateFromStdin(t *testing.T) {
	cmd := NewCreateWebhookCmd()
	if cmd.Flags().Lookup("namespace") == nil {
		cmd.Flags().String("namespace", "testkube", "")
	}
	cmd.SetIn(strings.NewReader(`{"text": "event id {{ .Id }}"}`))
	require.NoError(t, cmd.ParseFlags([]string{"--name", "bruno-test", "--payload-template", "-"}))

	options, err := NewCreateWebhookOptionsFromFlags(cmd)

	require.NoError(t, err)
	assert.Equal(t, `{"text": "event id {{ .Id }}"}`, options.PayloadTemplate)
}

func TestNewUpdateWebhookOptionsFromFlags_ReadsThePayloadTemplateFromStdin(t *testing.T) {
	cmd := UpdateWebhookCmd()
	if cmd.Flags().Lookup("namespace") == nil {
		cmd.Flags().String("namespace", "testkube", "")
	}
	cmd.SetIn(strings.NewReader(`{"text": "updated"}`))
	require.NoError(t, cmd.ParseFlags([]string{"--name", "bruno-test", "--payload-template", "-"}))

	options, err := NewUpdateWebhookOptionsFromFlags(cmd)

	require.NoError(t, err)
	require.NotNil(t, options.PayloadTemplate)
	assert.Equal(t, `{"text": "updated"}`, *options.PayloadTemplate)
}
