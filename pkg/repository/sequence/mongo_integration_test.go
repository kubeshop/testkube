package sequence

import (
	"context"
	"fmt"
	"sync"
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

func TestNewMongoRepository_GetNextExecutionNumber_Parallel_Integration(t *testing.T) {
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
		name          string
		executionType ExecutionType
	}{
		{
			1,
			"test",
			ExecutionTypeTest,
		},
		{
			2,
			"test",
			ExecutionTypeTest,
		},
		{
			1,
			"testsuite",
			ExecutionTypeTestSuite,
		},
		{
			2,
			"testsuite",
			ExecutionTypeTestSuite,
		},
		{
			1,
			"testworkflow",
			ExecutionTypeTestWorkflow,
		},
		{
			2,
			"testworkflow",
			ExecutionTypeTestWorkflow,
		},
		{
			1,
			"ts-old-testsuite",
			ExecutionTypeTest,
		},
		{
			1,
			"old-testworkflow",
			ExecutionTypeTest,
		},
		{
			2,
			"old-testsuite",
			ExecutionTypeTestSuite,
		},
		{
			2,
			"old-testworkflow",
			ExecutionTypeTestWorkflow,
		},
	}

	var results sync.Map
	var wg sync.WaitGroup

	for i := range tests {
		wg.Add(1)
		go func(executionType ExecutionType) {
			defer wg.Done()

			num, err := repo.GetNextExecutionNumber(ctx, tests[i].name, executionType)
			assert.NoError(t, err)

			results.Store(fmt.Sprintf("%s_%d", executionType, num), num)
		}(tests[i].executionType)
	}

	wg.Wait()

	for _, tt := range tests {
		num, ok := results.Load(fmt.Sprintf("%s_%d", tt.executionType, tt.expectedValue))
		assert.Equal(t, true, ok)

		value, ok := num.(int32)
		assert.Equal(t, true, ok)

		assert.Subset(t, []int32{1, 2}, []int32{value})
	}
}
