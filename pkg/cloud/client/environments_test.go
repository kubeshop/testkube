package client

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentsClient_Create(t *testing.T) {

	t.Run("error on response", func(t *testing.T) {
		// given
		o := NewEnvironmentsClient("testkube.dev", "token", "tkcorg_2bb1486705fb6997")
		o.RESTClient.Client = ClientMock{err: http.ErrNoLocation, validateRequestFunc: func(req *http.Request) error { return nil }}

		// when
		_, err := o.Create(Environment{})

		// then
		assert.Error(t, err)
	})

	t.Run("create org", func(t *testing.T) {
		// given
		o := NewEnvironmentsClient("testkube.dev", "token", "tkcorg_2bb1486705fb6997")
		resp := []byte(`{"id":"tkcenv_7991262606ff41ab","name":"env-name","connected":false,"organizationID":"tkcorg_2bb1486705fb6997","agentToken":"887179cf83a16","installCommand":"helm repo add kubeshop https://kubeshop.github.io/helm-charts ; helm repo update && helm upgrade --install --create-namespace testkube kubeshop/testkube --set testkube-api.cloud.key=tkcagnt_68cf83a16 --set testkube-api.minio.enabled=false --set mongodb.enabled=false --set testkube-dashboard.enabled=false --namespace testkube","installCommandCli":"testkube init --agent-uri agent.testkube.io:443 --agent-key tkcagnt_68dd1ed43a16"}`)
		o.RESTClient.Client = ClientMock{body: []byte(resp), validateRequestFunc: func(r *http.Request) error {
			d, _ := io.ReadAll(r.Body)
			assert.Equal(t, "{\"name\":\"env-name\",\"id\":\"\",\"connected\":false,\"owner\":\"\"}", string(d))
			return nil
		}}

		// when
		env, err := o.Create(Environment{Name: "env-name"})

		// then
		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_7991262606ff41ab", env.Id)
		assert.Equal(t, "env-name", env.Name)
		assert.Equal(t, "tkcorg_2bb1486705fb6997", env.OrganizationId)
		assert.Equal(t, "887179cf83a16", env.AgentToken)
	})
}
