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
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/internal/common"
)

const (
	mediaTypeJSON      = "application/json"
	mediaTypeYAML      = "text/yaml"
	mediaTypePlainText = "text/plain"
)

func ExpectsYAML(c *fiber.Ctx) bool {
	return c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML || c.Query("_yaml") == "true"
}

func HasYAML(c *fiber.Ctx) bool {
	return string(c.Request().Header.ContentType()) == mediaTypeYAML
}

func SendResourceList[T interface{}, U interface{}](c *fiber.Ctx, kind string, groupVersion schema.GroupVersion, jsonMapper func(T) U, data ...T) error {
	if ExpectsYAML(c) {
		return SendCRDs(c, kind, groupVersion, data...)
	}
	result := make([]U, len(data))
	for i, item := range data {
		result[i] = jsonMapper(item)
	}
	return c.JSON(result)
}

func SendResource[T interface{}, U interface{}](c *fiber.Ctx, kind string, groupVersion schema.GroupVersion, jsonMapper func(T) U, data T) error {
	if ExpectsYAML(c) {
		return SendCRDs(c, kind, groupVersion, data)
	}
	return c.JSON(jsonMapper(data))
}

func SendCRDs[T interface{}](c *fiber.Ctx, kind string, groupVersion schema.GroupVersion, crds ...T) error {
	b, err := common.SerializeCRDs(crds, common.SerializeOptions{
		OmitCreationTimestamp: true,
		CleanMeta:             true,
		Kind:                  kind,
		GroupVersion:          &groupVersion,
	})
	if err != nil {
		return err
	}
	c.Context().SetContentType(mediaTypeYAML)
	return c.Send(b)
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mongo.ErrNoDocuments) || k8serrors.IsNotFound(err) {
		return true
	}
	if e, ok := status.FromError(err); ok {
		return e.Code() == codes.NotFound
	}
	return false
}
