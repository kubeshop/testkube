# Development

## Running with CRDs only on kubernetes cluster

The minimial component which must be deployed on your local kubernetes cluster is testkube-operator with project CRDs (<https://github.com/kubeshop/testkube-operator>)

Checkout testkube-operator project and run:

```sh
make install 
```

to install CRD's into your local cluster

## Running on local machine

Next critical component is testkube API (<https://github.com/kubeshop/testkube>) and some executor you can build - your
own tests executor or existing one from Testkube.

Checkout testkube project and run local API server:

```sh
make run-mongo-dev run-api
```

Next go to testkube postman executor (<https://github.com/kubeshop/testkube-executor-postman>), checkout and run it
(Postman executor is also MongoDB based so it will use MongoDB launched with API server step):

```sh
make run-executor
```

### Installing local executors

You can install development executors by running them from testkube project (<https://github.com/kubeshop/testkube>)

```sh
make dev-install-local-executors
```

It'll register Custom Resources for

- local-postman/collection
- local-cypress/project
- local-curl/test

test types.

You'll need to create `Test` Custom Resource with type from above to
be executed on given executor. e.g.

```sh
kubectl testkube tests create --file my_collection_file.json --name my-test-name --type local-postman/collection
```

To summarize: `type` is the single relation between `Test` and `Executor`

## Intercepting api server on cluster

In case of debugging on Kubernetes we can intercept whole API Server (or Postman executor) service
with usage of [Telepresence](https://telepresence.io).

Simply intercept API server with local instance

You can start API Server with telepresence mode with:

```sh
make run-api-telepresence
```

and create/run tests pointed to in-cluster executors.
