# Testkube Installation Walkthrough 

## Get Your Cluster First 

In this demo we're using GKE (Google Kuberntetes Engine) - but you can use whatever you want. 

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


## Go through what components were installed

Until now we have several components installed
- Testkube Kubectl plugin - on your machine 
- Testkube Orchestrator API - this one is on your cluster
- Testkube Dashboard - Frontend for our API 
- Testkube Operator - For CRD management
- MinIO for artifacts storage - S3 replacement
- MongoDB - API storage
- Jetstack Cert Manager 


## Put Example Service into cluster 

We'll create some very simple service which will be tested for valid responses. Service will be written in the  `go` programming language

First let's build our Docker image and push it into registry (we're using Docker Hub Registry)

```sh
docker build  --platform linux/x86_64 -t kubeshop/testkube-example-service 
docker push kubeshop/testkube-example-service
```

(you can omit platform if you're on linux x86 64 bit)

Now when our Docker image can be fetched by Kubernetes let's create the `Deployment` resource.
Deployment will create our service pods and allow to use it inside Kubernetes cluster - it will be enough 
for purpose of this demo. We'll add also service to be able to connectg to our pod

Let's create file first: 

```yaml 

kind: Deployment
metadata:
  name: testkube-example-service
  labels:
    app: testkube-example-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: testkube-example-service
  template:
    metadata:
      labels:
        app: testkube-example-service
    spec:
      containers:
        - name: testkube-example-service
          image: kubeshop/testkube-example-service:latest
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
  name: testkube-example-service
spec:
  selector:
    app: testkube-example-service
  ports:
    - protocol: TCP
      port: 8881


```


## Create a few tests from scratch using postman, cypress and k6

Create new video and export it as file 

Content should be more or less like this: 

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
			"value": "http://testkube-example-service:8881",
			"type": "string"
		}
	]
}
```





## Upload tests to Testkube using GUI and CLI

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