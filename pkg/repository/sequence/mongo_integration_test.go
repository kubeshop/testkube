package sequence

import (
	"context"
	"testing"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	cfg, _ = config.Get()
)

func TestNewMongoRepository_GetNextExecutionNumber_Sequential_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.APIMongoDSN))
	if err != nil {
		t.Fatalf("error connecting to mongo: %v", err)
	}
	db := client.Database("sequence-mongo-repository-test")
	t.Cleanup(func() {
		db.Drop(ctx)
	})

	repo := NewMongoRepository(db)

	var tests = []struct {
		expectedValue int32
		executionType ExecutionType
	}{
		{
			1,
			ExecutionTypeTest,
		},
		{
			2,
			ExecutionTypeTest,
		},
		{
			1,
			ExecutionTypeTestSuite,
		},
		{
			2,
			ExecutionTypeTestSuite,
		},
		{
			1,
			ExecutionTypeTestWorkflow,
		},
		{
			2,
			ExecutionTypeTestWorkflow,
		},
	}

	for _, tt := range tests {
		num, err := repo.GetNextExecutionNumber(ctx, "name", tt.executionType)
		assert.NoError(t, err)
		assert.Equal(t, tt.expectedValue, num)
	}
}
