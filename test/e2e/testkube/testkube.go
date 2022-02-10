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

func (k Testkube) CreateScript(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "create", "--file", path, "--name", name, "--namespace", k.Namespace)
}

func (k Testkube) DeleteScript(name string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "delete", "--name", name, "--namespace", k.Namespace)
}

func (k Testkube) DeleteScripts() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "delete", "--all", "--namespace", k.Namespace)
}

func (k Testkube) StartScript(scriptName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "start", scriptName, "--name", executionName, "--namespace", k.Namespace)
}

func (k Testkube) Version() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "version")
}

func (k Testkube) List() ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "list", "--namespace", k.Namespace, "--output", k.Output)
}

func (k Testkube) Executions(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "executions", "--namespace", k.Namespace, "--output", k.Output)
}

func (k Testkube) Execution(scriptName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "testkube", "scripts", "execution", "--namespace", k.Namespace, "--output", k.Output, scriptName, executionName)
}
