# Testkube installation walkthrough 

## Get your cluster first 

In this demo we're using GKE (Google Kuberntetes Engine) 

## Installing Testkube Kubectl CLI plugin. 


```sh
brew install testkube
```

After successful intallation 

```sh 
kubectl testkube version

Client Version 1.2.3
Server Version  api/GET-testkube.ServerInfo returned error: api server response: '{"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"services \"testkube-api-server\" not found","reason":"NotFound","details":{"name":"testkube-api-server","kind":"services"},"code":404}
'
error: services "testkube-api-server" not found
Commit 
Built by Homebrew
Build date 
```

We can see Client version but server version is not found yet we need to install Testkube cluster components. 
We can do this using: 

## Installing cluster components
```sh 
kubectl testkube install

.... 
....
LAST DEPLOYED: Wed May 25 11:04:14 2022
NAMESPACE: testkube
STATUS: deployed
REVISION: 1
NOTES:
`Enjoy testing with testkube!`
```

## Show UI

Now we're ready to check if Testkube works ok

First let's looks at dashboard 

```sh
kubectl testkube dashboard

The dashboard is accessible here: http://localhost:8080/apiEndpoint?apiEndpoint=localhost:8088/v1 🥇
The API is accessible here: http://localhost:8088/v1/info 🥇
Port forwarding is started for the test results endpoint, hit Ctrl+c (or Cmd+c) to stop 🥇
```

Browser should open automatically new and shiny Testkube Dasboard


## Configuration of the cluster to expose the UI and go through what components were installed


## Configure ingress

All components should be ready now, but none of them are public as we've used default Testkube installation
Ingresses and auth are optional.

TODO Ingress walkthrough

## Configure Github authorization


TODO Github / Google Auth walkthrough

## Put Example Service into cluster 




## Create a few tests from scratch using postman, cypress and k6

## Upload tests to Testkube using GUI and CLI

## Show Testkube CRDs

## Run the tests using UI and CLI

## Navigate GUI and CLI showing executions

## Final message with the slide which has the Discord, Twitter and github links.
