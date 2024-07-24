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

	num1, err := repo.GetNextExecutionNumber(ctx, "name", ExecutionTypeTest)
	assert.NoError(t, err)
	assert.Equal(t, 1, num1)

	num2, err := repo.GetNextExecutionNumber(ctx, "name", ExecutionTypeTest)
	assert.NoError(t, err)
	assert.Equal(t, 2, num2)

	num3, err := repo.GetNextExecutionNumber(ctx, "name", ExecutionTypeTestSuite)
	assert.NoError(t, err)
	assert.Equal(t, 1, num3)

	num4, err := repo.GetNextExecutionNumber(ctx, "name", ExecutionTypeTestSuite)
	assert.NoError(t, err)
	assert.Equal(t, 2, num4)

	num5, err := repo.GetNextExecutionNumber(ctx, "name", ExecutionTypeTestWorkflow)
	assert.NoError(t, err)
	assert.Equal(t, 1, num5)

	num6, err := repo.GetNextExecutionNumber(ctx, "name", ExecutionTypeTestWorkflow)
	assert.NoError(t, err)
	assert.Equal(t, 2, num6)
}
