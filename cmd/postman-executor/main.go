package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/kubetest/internal/app/postman"
	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/postman/storage"
)

const DatabaseName = "postman-executor"

type MongoConfig struct {
	DSN string `envconfig:"EXECUTOR_MONGO_DSN" default:"mongodb://localhost:27017"`
	DB  string `envconfig:"EXECUTOR_MONGO_DB" default:"postman-executor"`
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

	executor := postman.NewPostmanExecutor(result.NewMongoRespository(db))
	executor.Init()
	panic(executor.Run())

}
