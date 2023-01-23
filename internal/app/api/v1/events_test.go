package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/server"

	"github.com/stretchr/testify/assert"
)

func TestTestkubeAPI_FluxEventHandler(t *testing.T) {
	// bootstrap api server fiber app
	app := fiber.New()
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
	}
	app.Post("/events/flux", s.FluxEventHandler())

	t.Run("test flux event", func(t *testing.T) {
		// given
		eventString := `{"involvedObject":{"kind":"Deployment","namespace":"my-ns","name":"my-deployment"},"severity":"info","timestamp":"2022-06-27T08:42:25Z","message":"some message","reason":"change","reportingController":"fluxcd"}`
		req := httptest.NewRequest("POST", "/events/flux", strings.NewReader(eventString))

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

}
