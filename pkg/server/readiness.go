package server

import (
	"sync"

	"github.com/gofiber/fiber/v2"
)

// ReadinessCheck reports whether one subsystem is still doing its job. A nil
// error means ready.
type ReadinessCheck func() error

// readinessRegistry holds the checks.
//
// It is referenced by pointer so that the registry survives HTTPServer being
// copied by value, which NewServer does.
type readinessRegistry struct {
	mu     sync.RWMutex
	checks map[string]ReadinessCheck
}

func newReadinessRegistry() *readinessRegistry {
	return &readinessRegistry{checks: make(map[string]ReadinessCheck)}
}

// AddReadinessCheck registers a check under a name. Registering the same name
// twice replaces the previous check.
func (s HTTPServer) AddReadinessCheck(name string, check ReadinessCheck) {
	if s.readiness == nil || check == nil {
		return
	}
	s.readiness.mu.Lock()
	defer s.readiness.mu.Unlock()
	s.readiness.checks[name] = check
}

// ReadyEndpoint answers 200 when every registered check passes, and 503 with the
// failing checks otherwise.
//
// This is deliberately separate from /health, which answers unconditionally.
// A process whose execution poll loop has stalled is still serving HTTP, so
// /health cannot see the failure and Kubernetes cannot act on it. Existing
// probes keep pointing at /health; only readiness should move here, because a
// stalled poll usually means the control plane is unreachable and restarting
// every runner in the fleet over that is worse than the stall.
func (s HTTPServer) ReadyEndpoint() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if s.readiness == nil {
			return c.JSON(fiber.Map{"status": "ready"})
		}

		s.readiness.mu.RLock()
		failures := make(map[string]string)
		for name, check := range s.readiness.checks {
			if err := check(); err != nil {
				failures[name] = err.Error()
			}
		}
		s.readiness.mu.RUnlock()

		if len(failures) > 0 {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not ready",
				"checks": failures,
			})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	}
}
