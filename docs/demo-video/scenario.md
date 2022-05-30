# Testkube Installation Walkthrough 

## Get Your Cluster First 

Prerequisite for this demo is to get some Kubernetes cluster. We're using GKE (Google Kuberntetes Engine) - but you can use whatever you want. 

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

We can see the `Client version` but the `Server version` is not found yet, as we need to install Testkube cluster components first. 

## Installing Testkube Cluster Components

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


## Go through what components were installed

Until now we have several components installed
- Testkube Kubectl plugin - on your machine 
- Testkube Orchestrator API - this one is on your cluster
- Testkube Dashboard - Frontend for our API 
- Testkube Operator - For CRD management
- MinIO for artifacts storage - S3 replacement
- MongoDB - API storage
- Jetstack Cert Manager 

We can look at them checking what pods are in `testkube` namespace. 

```sh 
kubectl get pods -ntestkube
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



## Put Example Service into cluster 

We'll create some very simple service which will be tested for valid responses. Service will be written in the  `go` programming language

First let's build our Docker image and push it into registry (we're using Docker Hub Registry)

```sh
docker build  --platform linux/x86_64 -t kubeshop/chuck-jokes .
docker push kubeshop/chuck-jokes
```

(you can omit platform if you're on linux x86 64 bit)

Now when our Docker image can be fetched by Kubernetes let's create the `Deployment` resource.
Deployment will create our service pods and allow to use it inside Kubernetes cluster - it will be enough 
for purpose of this demo. We'll add also `Service` to be able to connect to the Example Service Pod

Let's create `manifests.yaml` file: 

```yaml 

kind: Deployment
metadata:
  name: chuck-jokes
  labels:
    app: chuck-jokes
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chuck-jokes
  template:
    metadata:
      labels:
        app: chuck-jokes
    spec:
      containers:
        - name: chuck-jokes
          image: kubeshop/chuck-jokes:latest
          ports:
            - containerPort: 8881
          resources:
            limits:
              memory: 512Mi
              cpu: "1"
            requests:
              memory: 64Mi
              cpu: "0.2"
---
apiVersion: v1
kind: Service
metadata:
  name: chuck-jokes
spec:
  selector:
    app: chuck-jokes
  ports:
    - protocol: TCP
      port: 8881

```

And ask Kubernetes to sync this manifest with our cluster: 

```sh
kubectl apply -f manifests.yaml
```

After some time everything should be in place, Kubernetes scheduler will create new pod and add service to allow to connect to our service from cluster. 

## Create a few tests from scratch using postman, cypress and k6

### Postman test

Create new video and export it as file assuming file name is `Video-Chuck-Test.postman_collection.json` we can create the test with following command: 

```sh
kubectl testkube create test --file Video-Chuck-Test.postman_collection.json --name chuck-jokes-postman
```

Content of our file should be more or less like this: 

```json 
{
	"info": {
		"_postman_id": "046c7729-b816-498a-a07b-88407d4180dc",
		"name": "Video-Chuck-Test",
		"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		"_exporter_id": "3706349"
	},
	"item": [
		{
			"name": "Check if jokes are about Chuck",
			"event": [
				{
					"listen": "test",
					"script": {
						"exec": [
							"pm.test(\"Body matches string\", function () {",
							"    pm.expect(pm.response.text()).to.include(\"Chuck\");",
							"});"
						],
						"type": "text/javascript"
					}
				}
			],
			"request": {
				"method": "GET",
				"header": [],
				"url": {
					"raw": "{{API_URI}}/joke",
					"host": [
						"{{API_URI}}"
					],
					"path": [
						"joke"
					]
				}
			},
			"response": []
		}
	],
	"event": [
		{
			"listen": "prerequest",
			"script": {
				"type": "text/javascript",
				"exec": [
					""
				]
			}
		},
		{
			"listen": "test",
			"script": {
				"type": "text/javascript",
				"exec": [
					""
				]
			}
		}
	],
	"variable": [
		{
			"key": "API_URI",
			"value": "http://chuck-jokes.services:8881",
			"type": "string"
		}
	]
}
```


We can also add additional K6 test to check if our service is performant, let's check also if our service is talking about Chuck. And add it through dashboard.

```js 
import http from 'k6/http';
import { sleep,check } from 'k6';

export default function () {
  const baseURI = `${__ENV.API_URI || 'http://chuck-jokes.services:8881'}`;

  check(http.get(`${baseURI}/joke`), {
    'joke should be about Chuck': r => r.body.includes("Chuck")
  });
}
```

```sh
kubectl testkube create test --file chuck-jokes.k6.js --name chuck-jokes-k6 --type k6/script
```


## Upload tests to Testkube using GUI and CLI

To upload using API you need to 

## Show Testkube CRDs

## Run the tests using UI and CLI

## Navigate GUI and CLI showing executions

## Configure ingress to expose the the UI 

All components should be ready now, but none of them are public as we've used default Testkube installation
Ingresses and auth are optional.

TODO Ingress walkthrough

## Configure Github authorization


TODO Github / Google Auth walkthrough


## Final message with the slide which has the Discord, Twitter and github links.