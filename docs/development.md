# Development

## Running with CRDs only on kubernetes cluster

The minimial compoenent which must be deployed on kubernetes cluster is kubtest-operator with project CRDs

## Running on local machine

api-server need postman-executor (until we apply custom executors and operator for executors), 
so those two need to be started to play with api-server

api-server and postman-executor need MongoDB 

You can start mongo with:
```sh
make run-mongo-dev
```

You can start Postman executor and api server with: 
```sh
make run-executor
make run-api-server
```

### Installing local executors

You can development executors by running 

```sh
make dev-install-local-executors
```

It'll register Custom Resources for 
- postman/collection
- cypress/project
- curl/test
script types

## Intercepting api server on cluster

In case of debugging on Kubernetes we can intercept whole API Server (or Postman executor) service 
with use of [Telepresence](https://telepresence.io).

Simply intercept API server with local instance

You can start API Server with telepresence mode (executor pointed to in-cluster executor) with: 
```
make run-api-server-telepresence
```
