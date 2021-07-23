package main

import (
	"github.com/kelseyhightower/envconfig"
	v1API "github.com/kubeshop/kubetest/internal/app/api/v1"
	"github.com/kubeshop/kubetest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/postman/storage"
)

type MongoConfig struct {
	DSN string `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	DB  string `envconfig:"API_MONGO_DB" default:"kubetest"`
}

var Config MongoConfig

func init() {
	envconfig.Process("mongo", &Config)
}

func main() {
	db, err := storage.GetMongoDataBase(Config.DSN, Config.DB)
	if err != nil {
		panic(err)
	}

	repository := result.NewMongoRespository(db)
	v1API.NewServer(repository).Run()
}
