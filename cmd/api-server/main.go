package main

import (
	"fmt"
	"net"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/kubtest-operator/client"
	executorscr "github.com/kubeshop/kubtest-operator/client/executors"
	scriptscr "github.com/kubeshop/kubtest-operator/client/scripts"
	v1API "github.com/kubeshop/kubtest/internal/app/api/v1"
	"github.com/kubeshop/kubtest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubtest/internal/pkg/api/repository/storage"
	"github.com/kubeshop/kubtest/pkg/ui"
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

	if os.Getenv("KUBECONFIG") == "" {
		fmt.Println("KUBECONFIG not present. Exiting...")
		os.Exit(1)
	}

	port := os.Getenv("APISERVER_PORT")

	ln, err := net.Listen("tcp", ":"+port)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't listen on port %q: %s", port, err)
		os.Exit(1)
	}

	ln.Close()
	fmt.Printf("TCP Port %q is available", port)

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
