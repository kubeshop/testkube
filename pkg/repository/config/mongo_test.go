package config

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var (
	cfg, _ = config.Get()
)

func getRepository(dbName string) (*MongoRepository, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, err
	}
	db := client.Database(dbName)
	repo := NewMongoRepository(db)
	return repo, nil
}

func TestStorage_Integration(t *testing.T) {
	test.IntegrationTest(t)

	assert := require.New(t)

	t.Run("GetUniqueClusterId should return same id for each call", func(t *testing.T) {
		repository, err := getRepository("testkube-test-generate-id")
		t.Cleanup(func() {
			err := repository.Coll.Drop(context.Background())
			assert.NoError(err)
		})
		assert.NoError(err)
		// given/when
		id1, err := repository.GetUniqueClusterId(context.Background())
		assert.NoError(err)

		id2, err := repository.GetUniqueClusterId(context.Background())
		assert.NoError(err)

		id3, err := repository.GetUniqueClusterId(context.Background())
		assert.NoError(err)

		// then
		assert.Equal(id1, id2)
		assert.Equal(id1, id3)

	})

	t.Run("Upsert should insert new config entry", func(t *testing.T) {
		repository, err := getRepository("testkube-test-insert-cluster-id")
		t.Cleanup(func() {
			err := repository.Coll.Drop(context.Background())
			assert.NoError(err)
		})
		assert.NoError(err)
		// given,
		clusterId := "uniq3"
		_, err = repository.Upsert(context.Background(), testkube.Config{
			ClusterId: clusterId,
		})
		assert.NoError(err)

		// when
		config, err := repository.Get(context.Background())
		assert.NoError(err)

		// then
		assert.Equal(clusterId, config.ClusterId)
	})
}
