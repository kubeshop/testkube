package main

import (
	"flag"
	"net"
	"os"

	"github.com/kelseyhightower/envconfig"
	kubeclient "github.com/kubeshop/testkube-operator/client"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors"
	testsclientv2 "github.com/kubeshop/testkube-operator/client/tests/v2"
	testsuitesclientv1 "github.com/kubeshop/testkube-operator/client/testsuites/v1"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/storage"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

type MongoConfig struct {
	DSN string `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	DB  string `envconfig:"API_MONGO_DB" default:"testkube"`
}

var Config MongoConfig

var verbose = flag.Bool("v", false, "enable verbosity level")

func init() {
	flag.Parse()
	ui.Verbose = *verbose
	envconfig.Process("mongo", &Config)
}

func main() {

	telemetry.CollectAnonymousInfo()

	port := os.Getenv("APISERVER_PORT")

	ln, err := net.Listen("tcp", ":"+port)
	ui.ExitOnError("Checking if port "+port+"is free", err)
	ln.Close()
	ui.Debug("TCP Port is available", port)

	// DI
	db, err := storage.GetMongoDataBase(Config.DSN, Config.DB)
	ui.ExitOnError("Getting mongo database", err)

	kubeClient, err := kubeclient.GetClient()
	ui.ExitOnError("Getting kubernetes client", err)

	secretClient, err := secret.NewClient()
	ui.ExitOnError("Getting secret client", err)

	testsClient := testsclientv2.NewClient(kubeClient)
	executorsClient := executorsclientv1.NewClient(kubeClient)
	testsuitesClient := testsuitesclientv1.NewClient(kubeClient)

	resultsRepository := result.NewMongoRespository(db)
	testResultsRepository := testresult.NewMongoRespository(db)

	err = apiv1.NewServer(
		resultsRepository,
		testResultsRepository,
		testsClient,
		executorsClient,
		testsuitesClient,
		secretClient,
	).Run()
	ui.ExitOnError("Running API Server", err)
}
