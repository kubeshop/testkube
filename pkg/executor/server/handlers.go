package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/executor/repository/result"
	"github.com/kubeshop/kubtest/pkg/server"
)

func StartExecution(s server.HTTPServer, repository result.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request kubtest.ExecutionRequest
		err := json.Unmarshal(c.Body(), &request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
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

		err = repository.Insert(context.Background(), execution)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)

		}

		s.Log.Infow("starting new execution", "execution", execution)
		c.Response().Header.SetStatusCode(201)
		return c.JSON(execution)
	}
}

func GetExecution(s server.HTTPServer, repository result.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		execution, err := repository.Get(context.Background(), c.Params("id"))
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		return c.JSON(execution)
	}
}
