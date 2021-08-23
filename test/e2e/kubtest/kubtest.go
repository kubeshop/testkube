package kubtest

import (
	"github.com/kubeshop/kubtest/pkg/process"
)

/***
kubectl kubtest install
kubectl get pods                         # should return 3 pods
kubectl kubtest version
create new Postman collection (in Postman and save it somewhere)
kubectl kubtest scripts create --name test1 --file jw.postman_collection.json
kubectl kubtest scripts list            # check script name
kubectl kubtest scripts start test1
kubectl kubtest scripts executions      # copy last execution id
kubectl kubtest scripts execution SCRIPT_NAME EXECUTION_ID
***/

func NewKubtest(namespace string) Kubtest {
	return Kubtest{
		Namespace: namespace,
		Output:    "raw",
	}
}

type Kubtest struct {
	Namespace string
	Output    string
}

func (k Kubtest) Uninstall() ([]byte, error) {
	return process.Execute("helm", "uninstall", "kubtest", "--namespace", k.Namespace)
}

func (k Kubtest) Install() ([]byte, error) {
	return process.Execute("kubectl", "kubtest", "install", "--namespace", k.Namespace)
}

func (k Kubtest) CreateScript(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "kubtest", "scripts", "create", "--file", path, "--name", name, "--namespace", k.Namespace)
}

func (k Kubtest) StartScript(scriptName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "kubtest", "scripts", "start", scriptName, "--name", executionName, "--namespace", k.Namespace)
}

func (k Kubtest) Version() ([]byte, error) {
	return process.Execute("kubectl", "kubtest", "version")
}

func (k Kubtest) List() ([]byte, error) {
	return process.Execute("kubectl", "kubtest", "scripts", "list", "--namespace", k.Namespace, "--output", k.Output)
}

func (k Kubtest) Executions(name, path string) ([]byte, error) {
	return process.Execute("kubectl", "kubtest", "scripts", "executions", "--namespace", k.Namespace, "--output", k.Output)
}

func (k Kubtest) Execution(scriptName, executionName string) ([]byte, error) {
	return process.Execute("kubectl", "kubtest", "scripts", "execution", "--namespace", k.Namespace, "--output", k.Output, scriptName, executionName)
}
