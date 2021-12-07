# TestKube test scripts

Test scripts are single executor oriented tests. Script can have different types, which depends what executors are installed in your cluster. 

TestKubes includes `postman/collection`, `cypress/project` and `curl/test` script types which are auto registered during testkube install by default. // provide examples for cypress and  curl

As TestKube was designed with flexibility in mind - you can add your own executor which will handle additional script types. 


## Script test source

Scripts can be currently created from two sources: 
- First one is simple `file` with test content e.g. for postman collections we're exporting collection as json file, or for curl executor we're passing json file with configured curl command.
- Second source handled by TestKube is `git` - we can pass `repository`, `path` and `branch` where our tests is. This one is used in Cypress executor - as Cypress tests are more like npm-based projects which can have a lot of files. We're handling here sparse checkouts which are fast even in case of huge mono-repos 


## Create script

### Create your first script from file (Postman Collection test)

To create your first postman collection in TestKube you'll first need to export your collection into a file

Right click on your collection name
![create postman colletion step 1](img/script-create-1.png)
Click "Export" button 
![create postman colletion step 2](img/script-create-1.png)
Save in convinient location (We're using `~/Downloads/TODO.postman_collection.json` path)
![create postman colletion step 3](img/script-create-1.png)

```sh
$ kubectl testkube scripts create --file ~/Downloads/TODO.postman_collection.json --name test 

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Detected test script type postman/collection
Script created test 🥇
```

Script created! Now we can run as many times as we want. 


### Updating scripts

If you need to update your script after change in Postman just re-export it to file and run update command: 

```sh
$ kubectl testkube scripts update --file ~/Downloads/TODO.postman_collection.json --name test 

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Detected test script type postman/collection
Script updated test 🥇
```

Testkube will override all script settings and content with `update` method.


### Checking scripts content:


Let's see what has been created:

```sh
$ kubectl get scripts -n testkube
NAME   AGE
test   32s
```

```sh
$ kubectl get scripts -n testkube test -o yaml
apiVersion: tests.testkube.io/v1
kind: Script
metadata:
  creationTimestamp: "2021-12-07T10:33:11Z"
  generation: 1
  name: test
  namespace: testkube
  resourceVersion: "2553046"
  uid: afa071c4-533d-4112-afae-7c6f40f03750
spec:
  content: "{\n\t\"info\": {\n\t\t\"_postman_id\": \"b40de9fe-9201-4b03-8ca2-3064d9027dd6\",\n\t\t\"name\":
    \"TODO\",\n\t\t\"schema\": \"https://schema.getpostman.com/json/collection/v2.1.0/collection.json\"\n\t},\n\t\"item\":
    [\n\t\t{\n\t\t\t\"name\": \"Create TODO\",\n\t\t\t\"event\": [\n\t\t\t\t{\n\t\t\t\t\t\"listen\":
    \"test\",\n\t\t\t\t\t\"script\": {\n\t\t\t\t\t\t\"exec\": [\n\t\t\t\t\t\t\t\"pm.test(\\\"Status
    code is 201 CREATED\\\", function () {\",\n\t\t\t\t\t\t\t\"    pm.response.to.have.status(201);\",\n\t\t\t\t\t\t\t\"});\",\n\t\t\t\t\t\t\t\"\",\n\t\t\t\t\t\t\t\"\",\n\t\t\t\t\t\t\t\"pm.test(\\\"Check
    if todo item craeted successfully\\\", function() {\",\n\t\t\t\t\t\t\t\"    var
    json = pm.response.json();\",\n\t\t\t\t\t\t\t\"    pm.environment.set(\\\"item\\\",
    json.url);\",\n\t\t\t\t\t\t\t\"    pm.sendRequest(json.url, function (err, response)
    {\",\n\t\t\t\t\t\t\t\"        var json = pm.response.json();\",\n\t\t\t\t\t\t\t\"
    \       pm.expect(json.title).to.eq(\\\"Create video for conference\\\");\",\n\t\t\t\t\t\t\t\"\",\n\t\t\t\t\t\t\t\"
    \   });\",\n\t\t\t\t\t\t\t\"    console.log(\\\"creating\\\", pm.environment.get(\\\"item\\\"))\",\n\t\t\t\t\t\t\t\"})\",\n\t\t\t\t\t\t\t\"\",\n\t\t\t\t\t\t\t\"\"\n\t\t\t\t\t\t],\n\t\t\t\t\t\t\"type\":
    \"text/javascript\"\n\t\t\t\t\t}\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"listen\":
    \"prerequest\",\n\t\t\t\t\t\"script\": {\n\t\t\t\t\t\t\"exec\": [\n\t\t\t\t\t\t\t\"\"\n\t\t\t\t\t\t],\n\t\t\t\t\t\t\"type\":
    \"text/javascript\"\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t],\n\t\t\t\"protocolProfileBehavior\":
    {\n\t\t\t\t\"disabledSystemHeaders\": {}\n\t\t\t},\n\t\t\t\"request\": {\n\t\t\t\t\"method\":
    \"POST\",\n\t\t\t\t\"header\": [\n\t\t\t\t\t{\n\t\t\t\t\t\t\"key\": \"Content-Type\",\n\t\t\t\t\t\t\"value\":
    \"application/json\",\n\t\t\t\t\t\t\"type\": \"text\"\n\t\t\t\t\t}\n\t\t\t\t],\n\t\t\t\t\"body\":
    {\n\t\t\t\t\t\"mode\": \"raw\",\n\t\t\t\t\t\"raw\": \"{\\\"title\\\":\\\"Create
    video for conference\\\",\\\"order\\\":1,\\\"completed\\\":false}\"\n\t\t\t\t},\n\t\t\t\t\"url\":
    {\n\t\t\t\t\t\"raw\": \"{{uri}}\",\n\t\t\t\t\t\"host\": [\n\t\t\t\t\t\t\"{{uri}}\"\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t},\n\t\t\t\"response\":
    []\n\t\t},\n\t\t{\n\t\t\t\"name\": \"Complete TODO item\",\n\t\t\t\"event\": [\n\t\t\t\t{\n\t\t\t\t\t\"listen\":
    \"prerequest\",\n\t\t\t\t\t\"script\": {\n\t\t\t\t\t\t\"exec\": [\n\t\t\t\t\t\t\t\"console.log(\\\"completing\\\",
    pm.environment.get(\\\"item\\\"))\"\n\t\t\t\t\t\t],\n\t\t\t\t\t\t\"type\": \"text/javascript\"\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t],\n\t\t\t\"request\":
    {\n\t\t\t\t\"method\": \"PATCH\",\n\t\t\t\t\"header\": [\n\t\t\t\t\t{\n\t\t\t\t\t\t\"key\":
    \"Content-Type\",\n\t\t\t\t\t\t\"value\": \"application/json\",\n\t\t\t\t\t\t\"type\":
    \"text\"\n\t\t\t\t\t}\n\t\t\t\t],\n\t\t\t\t\"body\": {\n\t\t\t\t\t\"mode\": \"raw\",\n\t\t\t\t\t\"raw\":
    \"{\\\"completed\\\": true}\"\n\t\t\t\t},\n\t\t\t\t\"url\": {\n\t\t\t\t\t\"raw\":
    \"{{item}}\",\n\t\t\t\t\t\"host\": [\n\t\t\t\t\t\t\"{{item}}\"\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t},\n\t\t\t\"response\":
    []\n\t\t},\n\t\t{\n\t\t\t\"name\": \"Delete TODO item\",\n\t\t\t\"event\": [\n\t\t\t\t{\n\t\t\t\t\t\"listen\":
    \"prerequest\",\n\t\t\t\t\t\"script\": {\n\t\t\t\t\t\t\"exec\": [\n\t\t\t\t\t\t\t\"console.log(\\\"deleting\\\",
    pm.environment.get(\\\"item\\\"))\"\n\t\t\t\t\t\t],\n\t\t\t\t\t\t\"type\": \"text/javascript\"\n\t\t\t\t\t}\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"listen\":
    \"test\",\n\t\t\t\t\t\"script\": {\n\t\t\t\t\t\t\"exec\": [\n\t\t\t\t\t\t\t\"pm.test(\\\"Status
    code is 204 no content\\\", function () {\",\n\t\t\t\t\t\t\t\"    pm.response.to.have.status(204);\",\n\t\t\t\t\t\t\t\"});\"\n\t\t\t\t\t\t],\n\t\t\t\t\t\t\"type\":
    \"text/javascript\"\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t],\n\t\t\t\"request\": {\n\t\t\t\t\"method\":
    \"DELETE\",\n\t\t\t\t\"header\": [],\n\t\t\t\t\"url\": {\n\t\t\t\t\t\"raw\": \"{{item}}\",\n\t\t\t\t\t\"host\":
    [\n\t\t\t\t\t\t\"{{item}}\"\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t},\n\t\t\t\"response\":
    []\n\t\t}\n\t],\n\t\"event\": [\n\t\t{\n\t\t\t\"listen\": \"prerequest\",\n\t\t\t\"script\":
    {\n\t\t\t\t\"type\": \"text/javascript\",\n\t\t\t\t\"exec\": [\n\t\t\t\t\t\"\"\n\t\t\t\t]\n\t\t\t}\n\t\t},\n\t\t{\n\t\t\t\"listen\":
    \"test\",\n\t\t\t\"script\": {\n\t\t\t\t\"type\": \"text/javascript\",\n\t\t\t\t\"exec\":
    [\n\t\t\t\t\t\"\"\n\t\t\t\t]\n\t\t\t}\n\t\t}\n\t],\n\t\"variable\": [\n\t\t{\n\t\t\t\"key\":
    \"uri\",\n\t\t\t\"value\": \"http://localhost:8080/todos\"\n\t\t},\n\t\t{\n\t\t\t\"key\":
    \"item\",\n\t\t\t\"value\": null\n\t\t}\n\t]\n}"
  type: postman/collection
```

As we see we get `Script` resource with `spec.content` field which has escaped file content from exported postman collection. and `type` which need to be one handled by executor. `metadata.name` is name passed in testkube scripts create command.


### Create script from git

Some executors can handle files and some could handle only git resources - You'll need to follow particular executor readme file to be aware what scripts type does he handle. 


Let's assume that some Cypress project is created in some git repository - Let's assume we've created one in Let's assume we've created one in https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples/tree/main/examples  

Where `examples` is test directory in `https://github.com/kubeshop/testkube-executor-cypress.git` repository.

Now we can create our Cypress based script (in git based scripts we need to pass script type)

```sh
$ kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Script created kubeshop-cypress 🥇
```

Let's check how created testkube test script is defined on cluster: 

```sh
$ kubectl get scripts -n testkube kubeshop-cypress -o yaml
apiVersion: tests.testkube.io/v1
kind: Script
metadata:
  creationTimestamp: "2021-11-17T12:29:32Z"
  generation: 1
  name: kubeshop-cypress
  namespace: testkube
  resourceVersion: "225162"
  uid: f0d856aa-04fc-4238-bb4c-156ff82b4741
spec:
  repository:
    branch: main
    path: examples
    type: git
    uri: https://github.com/kubeshop/testkube-executor-cypress.git
  type: cypress/project
```

As we can see this script has `spec.repository` with git repository data. Those data can now be used by executor to download script data.

## Summary

Scripts are main smallest abstractions over tests in TestKube, they can be created with different sources and used by executors to run on top of particular test framework.


