package mongo

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

var (
	cfg, _ = config.Get()
)

func TestNewMongoRepository_GetNextExecutionNumber_Sequential_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
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
		executionType sequence.ExecutionType
	}{
		// check for new resources
		{
			1,
			"test",
			sequence.ExecutionTypeTest,
		},
		{
			2,
			"test",
			sequence.ExecutionTypeTest,
		},
		{
			1,
			"testsuite",
			sequence.ExecutionTypeTestSuite,
		},
		{
			2,
			"testsuite",
			sequence.ExecutionTypeTestSuite,
		},
		{
			1,
			"testworkflow",
			sequence.ExecutionTypeTestWorkflow,
		},
		{
			2,
			"testworkflow",
			sequence.ExecutionTypeTestWorkflow,
		},
		// check for existing resources
		{
			1,
			"ts-old-testsuite",
			sequence.ExecutionTypeTest,
		},
		{
			1,
			"old-testworkflow",
			sequence.ExecutionTypeTest,
		},
		{
			2,
			"old-testsuite",
			sequence.ExecutionTypeTestSuite,
		},
		{
			2,
			"old-testworkflow",
			sequence.ExecutionTypeTestWorkflow,
		},
	}

	for _, tt := range tests {
		num, err := repo.GetNextExecutionNumber(ctx, tt.name, tt.executionType)
		assert.NoError(t, err)
		assert.Equal(t, tt.expectedValue, num)
	}
}

func TestNewMongoRepository_GetNextExecutionNumber_Parallel_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
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
		executionType sequence.ExecutionType
	}{
		{
			1,
			"test",
			sequence.ExecutionTypeTest,
		},
		{
			2,
			"test",
			sequence.ExecutionTypeTest,
		},
		{
			1,
			"testsuite",
			sequence.ExecutionTypeTestSuite,
		},
		{
			2,
			"testsuite",
			sequence.ExecutionTypeTestSuite,
		},
		{
			1,
			"testworkflow",
			sequence.ExecutionTypeTestWorkflow,
		},
		{
			2,
			"testworkflow",
			sequence.ExecutionTypeTestWorkflow,
		},
	}

	var results sync.Map
	var wg sync.WaitGroup

	for i := range tests {
		wg.Add(1)
		go func(name string, executionType sequence.ExecutionType) {
			defer wg.Done()

			num, err := repo.GetNextExecutionNumber(ctx, name, executionType)
			assert.NoError(t, err)

			results.Store(fmt.Sprintf("%s_%d", executionType, num), num)
		}(tests[i].name, tests[i].executionType)
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
