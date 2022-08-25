package testkube

import (
	"github.com/kubeshop/testkube/pkg/process"
)

/***
kubectl testkube init
kubectl get pods                         # should return 3 pods
kubectl testkube version
create new Postman collection (in Postman and save it somewhere)
kubectl testkube create test --name test1 --file jw.postman_collection.json
kubectl testkube get tests            # check test name
kubectl testkube start test test1
kubectl testkube get executions      # copy last execution id
kubectl testkube get execution TEST_NAME EXECUTION_ID
***/

func NewTestkube(namespace string) Testkube {
	return Testkube{
		Namespace: namespace,
		Output:    "raw",
	}
}

type Testkube struct {
	Namespace string
	Output    string
}

func (k Testkube) Uninstall() ([]byte, error) {
	return process.Execute("helm", "uninstall", "testkube", "--namespace", k.Namespace)
}

func (k Testkube) Install() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "install", "--namespace", k.Namespace)
}

func (k Testkube) CreateTest(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "tests", "create", "--file", path, "--name", name, "--namespace", k.Namespace)
}

func (k Testkube) DeleteTest(name string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "tests", "delete", "--name", name, "--namespace", k.Namespace)
}

func (k Testkube) DeleteTests() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "tests", "delete", "--all", "--namespace", k.Namespace)
}

func (k Testkube) StartTest(testName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "tests", "start", testName, "--name", executionName, "--namespace", k.Namespace)
}

func (k Testkube) Version() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "version")
}

func (k Testkube) List() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "tests", "list", "--namespace", k.Namespace, "--output", k.Output)
}

func (k Testkube) Executions(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "tests", "executions", "--namespace", k.Namespace, "--output", k.Output)
}

func (k Testkube) Execution(testName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "tests", "execution", "--namespace", k.Namespace, "--output", k.Output, testName, executionName)
}
