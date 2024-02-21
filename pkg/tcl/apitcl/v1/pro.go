// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
)

func (s *apiTCL) isPro() bool {
	return s.ProContext != nil
}

//nolint:unused
func (s *apiTCL) isProPaid() bool {
	// TODO: Replace with proper implementation
	return s.isPro()
}

func (s *apiTCL) pro(h fiber.Handler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if s.isPro() {
			return h(ctx)
		}
		return s.Error(ctx, http.StatusPaymentRequired, errors.New("this functionality is only for the Pro/Enterprise subscription"))
	}
}

//nolint:unused
func (s *apiTCL) proPaid(h fiber.Handler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if s.isProPaid() {
			return h(ctx)
		}
		return s.Error(ctx, http.StatusPaymentRequired, errors.New("this functionality is only for paid plans of Pro/Enterprise subscription"))
	}
}
