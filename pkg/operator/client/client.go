package client

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	scriptv1 "github.com/kubeshop/testkube/api/script/v1"
	scriptv2 "github.com/kubeshop/testkube/api/script/v2"
	templatev1 "github.com/kubeshop/testkube/api/template/v1"
	testexecutionv1 "github.com/kubeshop/testkube/api/testexecution/v1"
	testsv1 "github.com/kubeshop/testkube/api/tests/v1"
	testsv2 "github.com/kubeshop/testkube/api/tests/v2"
	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	testsourcev1 "github.com/kubeshop/testkube/api/testsource/v1"
	testsuitev1 "github.com/kubeshop/testkube/api/testsuite/v1"
	testsuitev2 "github.com/kubeshop/testkube/api/testsuite/v2"
	testsuitev3 "github.com/kubeshop/testkube/api/testsuite/v3"
	testsuiteexecutionv1 "github.com/kubeshop/testkube/api/testsuiteexecution/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

// GetClient returns kubernetes CRD client with registered schemes
func GetClient() (client.Client, error) {
	scheme := runtime.NewScheme()

	utilruntime.Must(scriptv1.AddToScheme(scheme))
	utilruntime.Must(scriptv2.AddToScheme(scheme))
	utilruntime.Must(executorv1.AddToScheme(scheme))
	utilruntime.Must(testsv1.AddToScheme(scheme))
	utilruntime.Must(testsv2.AddToScheme(scheme))
	utilruntime.Must(testsv3.AddToScheme(scheme))
	utilruntime.Must(testsuitev1.AddToScheme(scheme))
	utilruntime.Must(testtriggersv1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(testsuitev2.AddToScheme(scheme))
	utilruntime.Must(testsuitev3.AddToScheme(scheme))
	utilruntime.Must(testsourcev1.AddToScheme(scheme))
	utilruntime.Must(testexecutionv1.AddToScheme(scheme))
	utilruntime.Must(testsuiteexecutionv1.AddToScheme(scheme))
	utilruntime.Must(templatev1.AddToScheme(scheme))
	utilruntime.Must(testworkflowsv1.AddToScheme(scheme))

	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	return client.New(kubeconfig, client.Options{Scheme: scheme})
}
