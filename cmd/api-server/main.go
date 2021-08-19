package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/kubtest-operator/client"
	scriptscr "github.com/kubeshop/kubtest-operator/client/scripts"
	v1API "github.com/kubeshop/kubtest/internal/app/api/v1"
	"github.com/kubeshop/kubtest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubtest/internal/pkg/postman/storage"
)

type MongoConfig struct {
	DSN string `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	DB  string `envconfig:"API_MONGO_DB" default:"kubtest"`
}

var Config MongoConfig

func init() {
	envconfig.Process("mongo", &Config)
}

func main() {
	// DI
	db, err := storage.GetMongoDataBase(Config.DSN, Config.DB)
	if err != nil {
		panic(err)
	}

	kubeClient := client.GetClient()
	scriptsClient := scriptscr.NewClient(kubeClient)

	repository := result.NewMongoRespository(db)
	v1API.NewServer(repository, scriptsClient).Run()
}
