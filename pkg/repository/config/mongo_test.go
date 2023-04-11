package config

import (
	"context"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/kubeshop/testkube/pkg/repository/storage"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	mongoDns    = "mongodb://localhost:27017"
	mongoDbName = "testkube-test"
)

func getRepository() (*MongoRepository, error) {
	db, err := storage.GetMongoDatabase(mongoDns, mongoDbName, storage.TypeMongoDB, false, nil)
	repository := NewMongoRepository(db)
	return repository, err
}

func TestStorage_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	assert := require.New(t)

	repository, err := getRepository()
	assert.NoError(err)

	err = repository.Coll.Drop(context.TODO())
	assert.NoError(err)

	t.Run("GetUniqueClusterId should return same id for each call", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
		// given,
		clusterId := "uniq3"
		_, err := repository.Upsert(context.Background(), testkube.Config{
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
