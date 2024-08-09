package config

import (
	"context"
	"testing"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/kubeshop/testkube/pkg/repository/storage"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	mongoDbName = "testkube-test"
)

var (
	cfg, _ = config.Get()
)

func getRepository(mongoDbName string) (*MongoRepository, error) {
	db, err := storage.GetMongoDatabase(cfg.APIMongoDSN, mongoDbName, storage.TypeMongoDB, false, nil)
	repository := NewMongoRepository(db)
	return repository, err
}

func TestStorage_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	assert := require.New(t)

	t.Run("GetUniqueClusterId should return same id for each call", func(t *testing.T) {
		repository, err := getRepository("testkube_storage_integration_test_get_uniqiu_cluster_id")
		assert.NoError(err)

		t.Cleanup(func() {
			err = repository.Coll.Drop(context.TODO())
			assert.NoError(err)
		})

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

		repository, err := getRepository("testkube_storage_integration_test_insert_new_config")
		assert.NoError(err)

		t.Cleanup(func() {
			err = repository.Coll.Drop(context.TODO())
			assert.NoError(err)
		})

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
