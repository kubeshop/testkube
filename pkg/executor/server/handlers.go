package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubtest/internal/pkg/server"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/executor/repository"
)

type Handlers struct {
	server.HTTPServer
	Repository repository.Repository
}

func (p *Handlers) StartExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request kubtest.ExecutionRequest
		err := json.Unmarshal(c.Body(), &request)
		if err != nil {
			return p.Error(c, http.StatusBadRequest, err)
		}

		execution := kubtest.NewExecution()

		execution.WithContent(request.Content).
			WithParams(request.Params)

		if request.Repository != nil {
			execution.WithRepositoryData(
				request.Repository.Uri,
				request.Repository.Branch,
				request.Repository.Path,
			)
		}

		err = p.Repository.Insert(context.Background(), execution)
		if err != nil {
			return p.Error(c, http.StatusInternalServerError, err)

		}

		p.Log.Infow("starting new execution", "execution", execution)
		c.Response().Header.SetStatusCode(201)
		return c.JSON(execution)
	}
}

func (p Handlers) GetExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		execution, err := p.Repository.Get(context.Background(), c.Params("id"))
		if err != nil {
			return p.Error(c, http.StatusInternalServerError, err)
		}

		return c.JSON(execution)
	}
}
