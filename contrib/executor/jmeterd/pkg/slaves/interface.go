package slaves

import "context"

type Interface interface {
	CreateSlaves(ctx context.Context, count int) (SlaveMeta, error)
	DeleteSlaves(ctx context.Context, meta SlaveMeta) error
}
