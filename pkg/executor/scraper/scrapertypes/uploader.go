package scrapertypes

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:generate mockgen -destination=./mock_uploader.go -package=scrapertypes "github.com/kubeshop/testkube/pkg/executor/scraper/scrapertypes" Uploader
type Uploader interface {
	Upload(ctx context.Context, object *Object, execution testkube.Execution) error
	Close() error
}
