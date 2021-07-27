package main

import (
	"github.com/kelseyhightower/envconfig"
	v1API "github.com/kubeshop/kubetest/internal/app/api/v1"
	"github.com/kubeshop/kubetest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/postman/storage"
	"github.com/kubeshop/kubetest/pkg/kubernetes/client"
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
	// DI
	db, err := storage.GetMongoDataBase(Config.DSN, Config.DB)
	if err != nil {
		panic(err)
	}

	kubeClient := client.GetClient()
	scriptsKubeAPI := client.NewScripts(kubeClient)

	repository := result.NewMongoRespository(db)
	v1API.NewServer(repository, scriptsKubeAPI).Run()
}
