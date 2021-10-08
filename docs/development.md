# Development

## Running with CRDs only on kubernetes cluster

The minimial compoenent which must be deployed on your local kubernetes cluster is testkube-operator with project CRDs (https://github.com/kubeshop/testkube-operator)

Checkout this project and run: 
```sh
make install 
```
to install CRD's in your local cluster


## Running on local machine

Next critical component is API (https://github.com/kubeshop/testkube) and some executor you can build your
own tests executor or use one from TestKube. 

First let's run local API server:

```sh
make run-mongo-dev run-api-server
```

Next goto executor (https://github.com/kubeshop/testkube-executor-postman) and run it 
(Postman executor is also MongoDB based so it'll use database run in API server step):

```sh
make run-executor
```

### Installing local executors

You can install development executors by running 

```sh
make dev-install-local-executors
```

It'll register Custom Resources for 

- local-postman/collection
- local-cypress/project
- local-curl/test

script types. 

You'll need to create `Script` Custom Resource with type from above to 
be executed on given executor. e.g. 

```sh
kubectl testkube scripts create --file my_collection_file.json --name my-test-name --type local-postman/collection
```

To summarize: `type` is the single relation between `Script` and `Executor`

## Intercepting api server on cluster

In case of debugging on Kubernetes we can intercept whole API Server (or Postman executor) service 
with use of [Telepresence](https://telepresence.io).

Simply intercept API server with local instance

You can start API Server with telepresence mode with: 

```
make run-api-server-telepresence
```

and create/run test scripts pointed to in-cluster executors.
