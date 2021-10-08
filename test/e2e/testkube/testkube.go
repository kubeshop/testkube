package testkube

import (
	"github.com/kubeshop/testkube/pkg/process"
)

/***
kubectl testkube install
kubectl get pods                         # should return 3 pods
kubectl testkube version
create new Postman collection (in Postman and save it somewhere)
kubectl testkube scripts create --name test1 --file jw.postman_collection.json
kubectl testkube scripts list            # check script name
kubectl testkube scripts start test1
kubectl testkube scripts executions      # copy last execution id
kubectl testkube scripts execution SCRIPT_NAME EXECUTION_ID
***/

func NewTestKube(namespace string) TestKube {
	return TestKube{
		Namespace: namespace,
		Output:    "raw",
	}
}

type TestKube struct {
	Namespace string
	Output    string
}

func (k TestKube) Uninstall() ([]byte, error) {
	return process.Execute("helm", "uninstall", "testkube", "--namespace", k.Namespace)
}

func (k TestKube) Install() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "install", "--namespace", k.Namespace)
}

func (k TestKube) CreateScript(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "create", "--file", path, "--name", name, "--namespace", k.Namespace)
}

func (k TestKube) StartScript(scriptName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "start", scriptName, "--name", executionName, "--namespace", k.Namespace)
}

func (k TestKube) Version() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "version")
}

func (k TestKube) List() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "list", "--namespace", k.Namespace, "--output", k.Output)
}

func (k TestKube) Executions(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "executions", "--namespace", k.Namespace, "--output", k.Output)
}

func (k TestKube) Execution(scriptName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "execution", "--namespace", k.Namespace, "--output", k.Output, scriptName, executionName)
}
