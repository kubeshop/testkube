package slaves

import "context"

type Interface interface {
	CreateSlaves(context.Context) (SlaveMeta, error)
	DeleteSlaves(context.Context, SlaveMeta) error
}
