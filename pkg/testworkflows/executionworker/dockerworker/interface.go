package dockerworker

import (
	"time"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type ImageInspectorConfig struct {
	CacheEnabled bool
	CacheKey     string
	CacheTTL     time.Duration
}

type Config struct {
	ImageInspector    ImageInspectorConfig
	Connection        testworkflowconfig.WorkerConnectionConfig
	LogAbortedDetails bool
}
