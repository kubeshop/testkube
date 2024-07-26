package control

import "time"

type ServerOptions struct {
	HandlePause  func(ts time.Time) error
	HandleResume func(ts time.Time) error
}

func (p ServerOptions) Pause(ts time.Time) error {
	return p.HandlePause(ts)
}

func (p ServerOptions) Resume(ts time.Time) error {
	return p.HandleResume(ts)
}
