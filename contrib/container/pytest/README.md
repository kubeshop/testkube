# Pytest container executor
This is a simple python based executor for pytest framework https://docs.pytest.org/

## Docker image
Current Docker image is based on python 3.8.17 and a few basic modules, like pipenv, pytest and requests.
Feel free to change the python version, install missing dependencies, etc. Docker image should be placed in your 
favourite docker image registry, like local Docker image registry for Minikube, Kind, etc or Cloud provider one,
create it using `docker build -t pytest-executor -f Dockerfile`

## CRD installation
Create test and executor CRD using provided YAML specification, don't forget to point to a proper location
of the Docker image in executor CRD, use command
`kubectl apply -f container-executor-pytest.yaml`
`kubectl apply -f pytest_test.yaml`

## Run created tests
Use 
`kubectl testkube run test container-executor-pytest-failed-sample`
`kubectl testkube run test container-executor-pytest-passed-sample`
