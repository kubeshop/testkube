package main

import (
	"net"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/testkube-operator/client"
	executorscr "github.com/kubeshop/testkube-operator/client/executors"
	scriptscr "github.com/kubeshop/testkube-operator/client/scripts"
	v1API "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/storage"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

type MongoConfig struct {
	DSN string `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	DB  string `envconfig:"API_MONGO_DB" default:"testkube"`
}

var Config MongoConfig

func init() {
	envconfig.Process("mongo", &Config)
}

func main() {

	telemetry.CollectAnonymousInfo()

	port := os.Getenv("APISERVER_PORT")

	ln, err := net.Listen("tcp", ":"+port)
	ui.ExitOnError("Checking if port "+port+"is free", err)
	ln.Close()
	ui.Info("TCP Port is available", port)

	// DI
	db, err := storage.GetMongoDataBase(Config.DSN, Config.DB)
	ui.ExitOnError("Getting mongo database", err)

	kubeClient := client.GetClient()

	scriptsClient := scriptscr.NewClient(kubeClient)

	executorsClient := executorscr.NewClient(kubeClient)

	repository := result.NewMongoRespository(db)
	err = v1API.NewServer(repository, scriptsClient, executorsClient).Run()
	ui.ExitOnError("Running API Server", err)
}
