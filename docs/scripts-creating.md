# TestKube test scripts

Test scripts are single executor oriented tests. Script can have different types, which depends what executors are installed in your cluster. 

TestKubes includes `postman/collection`, `cypress/project` and `curl/test` script types which are auto registered during testkube install by default. 

As TestKube was designed with flexibility in mind - you can add your own executor which will handle additional script types. 


## Script test source

Scripts can be currently created from two sources: 
- First one is simple `file` with test content e.g. for postman collections we're exporting collection as json file, or for curl executor we're passing json file with configured curl command.
- Second source handled by testkube is `git` - we can pass `repository`, `path` and `branch` where our tests is. This one is used in Cypress executor - as Cypress tests are more like npm-based projects which can have a lot of files. We're handling here sparse checkouts which are fast even in case of huge mono-repos 


## Create script

### Create your first script from file (Postman Collection test)

Let's assume you have exported Postman collection which want to test some internal service inside Kuberntertes cluster
We've saved it into `~/Downloads/API-Health.postman_collection.json` 

```sh
kubectl testkube scripts create --name api-incluster-test --file ~/Downloads/API-Health.postman_collection.json --type postman/collection 

██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                                 /kʌb tɛst/ by Kubeshop


Script created  🥇
```

Script created! Now we can run as many times as we want. 

We can check how TestKube stores this test internally by getting its details: 

```sh
kubectl get scripts api-incluster-test -oyaml

apiVersion: tests.testkube.io/v1
kind: Script
metadata:
  creationTimestamp: "2021-10-06T08:49:14Z"
  generation: 1
  name: api-incluster-test
  namespace: default
  resourceVersion: "31613607"
  uid: c1ba1129-95ba-49bc-8e94-2c18ae7854f7
spec:
  content: "{\n\t\"info\": {\n\t\t\"_postman_id\": \"4a2719c2-4ff0-4400-8d57-431e6e565ba4\",\n\t\t\"name\":
    \"API-Health\",\n\t\t\"schema\": \"https://schema.getpostman.com/json/collection/v2.1.0/collection.json\"\n\t},\n\t\"item\":
    [\n\t\t{\n\t\t\t\"name\": \"Health\",\n\t\t\t\"event\": [\n\t\t\t\t{\n\t\t\t\t\t\"listen\":
    \"test\",\n\t\t\t\t\t\"script\": {\n\t\t\t\t\t\t\"exec\": [\n\t\t\t\t\t\t\t\"pm.test(\\\"Status
    code is 200\\\", function () {\",\n\t\t\t\t\t\t\t\"    pm.response.to.have.status(200);\",\n\t\t\t\t\t\t\t\"});\"\n\t\t\t\t\t\t],\n\t\t\t\t\t\t\"type\":
    \"text/javascript\"\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t],\n\t\t\t\"request\": {\n\t\t\t\t\"method\":
    \"GET\",\n\t\t\t\t\"header\": [],\n\t\t\t\t\"url\": {\n\t\t\t\t\t\"raw\": \"http://testkube-api-server:8088/health\",\n\t\t\t\t\t\"protocol\":
    \"http\",\n\t\t\t\t\t\"host\": [\n\t\t\t\t\t\t\"testkube-api-server\"\n\t\t\t\t\t],\n\t\t\t\t\t\"port\":
    \"8088\",\n\t\t\t\t\t\"path\": [\n\t\t\t\t\t\t\"health\"\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t},\n\t\t\t\"response\":
    []\n\t\t}\n\t],\n\t\"event\": [\n\t\t{\n\t\t\t\"listen\": \"prerequest\",\n\t\t\t\"script\":
    {\n\t\t\t\t\"type\": \"text/javascript\",\n\t\t\t\t\"exec\": [\n\t\t\t\t\t\"\"\n\t\t\t\t]\n\t\t\t}\n\t\t},\n\t\t{\n\t\t\t\"listen\":
    \"test\",\n\t\t\t\"script\": {\n\t\t\t\t\"type\": \"text/javascript\",\n\t\t\t\t\"exec\":
    [\n\t\t\t\t\t\"\"\n\t\t\t\t]\n\t\t\t}\n\t\t}\n\t]\n}"
  type: postman/collection
```

As we see we get `Script` resource with `spec.content` field which has escaped file content from exported postman collection. and `type` which need to be one handled by executor. `metadata.name` is name passed in testkube scripts create command.


### Create script from git

Some executors can handle files and some could handle only git resources - You'll need to follow particular executor readme file to be aware what scripts type does he handle. 


Let's assume that some Cypress project is created in some git repository - Let's assume we've created one in Let's assume we've created one in https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples/tree/main/examples  

Where `examples` is test directory in `https://github.com/kubeshop/testkube-executor-cypress.git` repository.

Now we can create our Cypress based script

```sh
kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project
```

Let's check how created testkube test script is defined on cluster: 

```sh
kubectl get scripts kubeshop-cypress -oyaml
apiVersion: tests.testkube.io/v1
kind: Script
metadata:
  name: kubeshop-cypress
  namespace: default
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


