package v1

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/pkg/analytics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	phttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

const (
	// cliIngressHeader is cli ingress header
	cliIngressHeader = "X-CLI-Ingress"
)

// HandleEmitterLogs is a handler to emit logs
func (s TestkubeAPI) HandleEmitterLogs() {
	go func() {
		s.Log.Debug("Listening for workers results")
		for resp := range s.EventsEmitter.Responses {
			if resp.Error != nil {
				s.Log.Errorw("got error when sending webhooks", "response", resp)
				continue
			}
			s.Log.Debugw("got webhook response", "response", resp)
		}
	}()
}

// AuthHandler is auth middleware
func (s TestkubeAPI) AuthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get(cliIngressHeader, "") != "" {
			client := phttp.NewClient()
			req, err := http.NewRequest(http.MethodGet, os.Getenv("TESTKUBE_USERINFO_URI"), nil)
			if err != nil {
				s.Log.Errorf("error preparing oauth request", "error", err)
				c.Status(http.StatusUnauthorized)
				return err
			}

			req.Header.Add("Authorization", c.Get("Authorization", ""))
			resp, err := client.Do(req)
			if err != nil {
				s.Log.Errorf("error sending oauth request", "error", err)
				c.Status(http.StatusUnauthorized)
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				return fmt.Errorf("status: %s", resp.Status)
			}

			if _, err = ioutil.ReadAll(resp.Body); err != nil {
				s.Log.Errorf("error reading oauth response", "error", err)
				c.Status(http.StatusUnauthorized)
				return err
			}
		}

		return c.Next()
	}
}

// InfoHandler is a handler to get info
func (s TestkubeAPI) InfoHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(testkube.ServerInfo{
			Commit:  api.Commit,
			Version: api.Version,
		})
	}
}

// RoutesHandler is a handler to get existing routes
func (s TestkubeAPI) RoutesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		routes := []fiber.Route{}

		stack := s.Mux.Stack()
		for _, e := range stack {
			for _, s := range e {
				route := *s
				routes = append(routes, route)
			}
		}

		return c.JSON(routes)
	}
}

// AnalyticsHandler is analytics recording middleware
func (s TestkubeAPI) AnalyticsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		go func(host, path, method string) {
			out, err := analytics.SendAnonymousAPIRequestInfo(host, path, api.Version, method, s.ClusterID)
			l := s.Log.With("measurmentId", analytics.TestkubeMeasurementID, "secret", text.Obfuscate(analytics.TestkubeMeasurementSecret), "path", path)
			if err != nil {
				l.Debugw("sending analytics event error", "error", err)
			} else {
				l.Debugw("anonymous info to tracker sent", "output", out)
			}
		}(c.Hostname(), c.Route().Path, c.Method()) // log route path in form /v1/tests/:name

		return c.Next()
	}
}
