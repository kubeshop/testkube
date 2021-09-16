package server

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

const (
	MaxLimit      = 1000
	DefaultLimit  = 100
	DefaultOffset = 100
)

type Pager struct {
	Limit  int
	Offset int
	NextID string
}

func (s HTTPServer) GetPagerParams(c *fiber.Ctx) Pager {
	limit, err := strconv.Atoi(c.Query("limit", "100"))
	if err != nil || limit < 1 {
		limit = DefaultLimit
	} else if limit > MaxLimit {
		limit = MaxLimit
	}

	offset, err := strconv.Atoi(c.Query("offset", "100"))
	if err != nil || limit < 1 {
		offset = DefaultLimit
	} else if limit > MaxLimit {
		offset = MaxLimit
	}

	return Pager{
		Limit:  limit,
		Offset: offset,
		NextID: c.Query("nextID", ""),
	}
}
