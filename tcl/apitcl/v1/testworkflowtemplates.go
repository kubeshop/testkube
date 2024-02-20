// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"github.com/gofiber/fiber/v2"
)

func (s *apiTCL) ListTestWorkflowTemplatesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return s.NotImplemented(c)
	}
}

func (s *apiTCL) GetTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_ = c.Params("id")
		return s.NotImplemented(c)
	}
}

func (s *apiTCL) DeleteTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_ = c.Params("id")
		return s.NotImplemented(c)
	}
}

func (s *apiTCL) CreateTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_ = c.Params("id")
		return s.NotImplemented(c)
	}
}

func (s *apiTCL) PatchTestWorkflowTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_ = c.Params("id")
		return s.NotImplemented(c)
	}
}
