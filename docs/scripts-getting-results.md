# Getting Testkube Scripts Execution Results

We saw how simple it is to create and run Testkube scripts. Getting results is also simple.

## **Getting Test Executions After a Script is Executed**

After each test script runs, Testkube informs you that you can get the results of given script execution.

```sh
kubectl testkube run test api-incluster-test
```

Output:

```sh
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Type          : postman/collection
Name          : api-incluster-test
Execution ID  : 615d6398b046f8fbd3d955d4
Execution name: openly-full-bream

Script queued for execution

Use the following command to get script execution details:
$ kubectl testkube get execution 615d6398b046f8fbd3d955d4

Or watch script execution until complete:
$ kubectl testkube watch execution 615d6398b046f8fbd3d955d4

```

`kubectl testkube get execution 615d6398b046f8fbd3d955d4` - is for getting string output of script execution.

Where `615d6398b046f8fbd3d955d4` is the script execution ID.

## **Change the Output Format of the Execution Result**

By default, Testkube returns the string output of a particular executor. It can also return JSON or Go-Template based outputs.

### **JSON Output**

Sometimes you need to parse test results programatically. To simplify this task, you can get results of the test execution in JSON format.

```sh

kubectl testkube get execution 615d7e1ab046f8fbd3d955d6 -ojson

 {"id":"615d7e1ab046f8fbd3d955d6","scriptName":"api-incluster-test","scriptType":"postman/collection","name":"monthly-sure-finch","executionResult":{"status":"success","startTime":"2021-10-06T10:44:46.338Z","endTime":"2021-10-06T10:44:46.933Z","output":"newman\n\nAPI-Health\n\n→ Health\n  GET http://testkube-api-server:8088/health [200 OK, 124B, 282ms]\n  ✓  Status code is 200\n\n┌─────────────────────────┬────────────────────┬───────────────────┐\n│                         │           executed │            failed │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              iterations │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│                requests │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│            test-scripts │                  2 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│      prerequest-scripts │                  1 │                 0 │\n├─────────────────────────┼────────────────────┼───────────────────┤\n│              assertions │                  1 │                 0 │\n├─────────────────────────┴────────────────────┴───────────────────┤\n│ total run duration: 519ms                                        │\n├──────────────────────────────────────────────────────────────────┤\n│ total data received: 8B (approx)                                 │\n├──────────────────────────────────────────────────────────────────┤\n│ average response time: 282ms [min: 282ms, max: 282ms, s.d.: 0µs] │\n└──────────────────────────────────────────────────────────────────┘\n","outputType":"text/plain","steps":[{"name":"Health","duration":"282ms","status":"success","assertionResults":[{"name":"Status code is 200","status":"success"}]}]}}

```

It's quite easy to parse data from test executions with tools like `jq` or in other programmatic ways.

### **Need Non-Standard Output? Go-Template to the Rescue**

For non-standard test execution output, use the ouput parameter `-ogo` with passed `--go-template` template content.


```sh
kubectl testkube get execution 615d7e1ab046f8fbd3d955d6 -ogo --go-template='{{.Name}} {{.Id}} {{.ExecutionResult.Status}}'
```

Output:

```sh
monthly-sure-finch 615d7e1ab046f8fbd3d955d6 success  
```

## **Getting a List of Test Script Executions**

Watch this video to see how to get script results in different formats: 

<iframe width="560" height="315" src="https://www.youtube.com/embed/ukHvS5x7TvM" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

### **Getting a list of Recent Executions**

Get a list of current executions with use of `executions` subcommand:

```sh
kubectl testkube get executions 
```

Output:

```sh
        SCRIPT        |        TYPE        | NAME |            ID            | STATUS   
+---------------------+--------------------+------+--------------------------+---------+
  api-incluster-test  | postman/collection |      | 615d7e1ab046f8fbd3d955d6 | success  
  api-incluster-test  | postman/collection |      | 615d6398b046f8fbd3d955d4 | success  
  kubeshop-cypress    | cypress/project    |      | 615d5372b046f8fbd3d955d2 | success  
  kubeshop-cypress    | cypress/project    |      | 615d5265b046f8fbd3d955d0 | error    
  cypress-example     | cypress/project    |      | 615d4fe6b046f8fbd3d955ce | error    
  cypress-example     | cypress/project    |      | 615d4556b046f8fbd3d955cc | error   
```

Select the execution ID and check the details of the result of the test:

```sh
kubectl testkube get execution 615d5265b046f8fbd3d955d0
```

### **Getting a List of Executions in Different Formats**

Table data in terminal mode is not always useful when processing results in code or shell scripts. To simplify this, Testkube allows the use of JSON or Go Template to obtain results in a list.

#### **JSON**

Getting JSON results is quite easy - just pass `-o json` flag to the test command:

```sh
kubectl testkube get executions -ojson

{"totals":{"results":17,"passed":7,"failed":10,"queued":0,"pending":0},"results":[{"id":"615d7e1ab046f8fbd3d955d6","name":"","scriptName":"api-incluster-test","scriptType":"postman/collection","status":"success","startTime":"2021-10-06T10:44:46.338Z","endTime":"2021-10-06T10:44:46.933Z"},{"id":"615d6398b046f8fbd3d955d4","name":"","scriptName":"api-incluster-test","scriptType":"postman/collection","status":"success","startTime":"2021-10-06T08:51:39.834Z","endTime":"2021-10-06T08:51:40.432Z"},{"id":"615d5372b046f8fbd3d955d2","name":"","scriptName":"kubeshop-cypress","scriptType":"cypress/project","status":"success","startTime":"0001-01-01T00:00:00Z","endTime":"2021-10-06T07:44:30.025Z"},{"id":"615d5265b046f8fbd3d955d0","name":"","scriptName":"kubeshop-cypress","scriptType":"cypress/project","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-10-06T07:40:09.261Z"},{"id":"615d4fe6b046f8fbd3d955ce","name":"","scriptName":"cypress-example","scriptType":"cypress/project","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-10-06T07:28:54.579Z"},{"id":"615d4556b046f8fbd3d955cc","name":"","scriptName":"cypress-example","scriptType":"cypress/project","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-10-06T06:43:44.1Z"},{"id":"615d43d3b046f8fbd3d955ca","name":"","scriptName":"cypress-example","scriptType":"cypress/project","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-10-06T06:37:52.601Z"},{"id":"6155cd7db046f8fbd3d955c8","name":"","scriptName":"postman-test-7f6qrm","scriptType":"postman/collection","status":"success","startTime":"2021-09-30T14:45:20.819Z","endTime":"2021-09-30T14:45:21.419Z"},{"id":"6155cd67b046f8fbd3d955c6","name":"","scriptName":"sanity","scriptType":"postman/collection","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-09-30T14:45:00.135Z"},{"id":"615322f3f47de75f31ae7a06","name":"","scriptName":"long-1","scriptType":"postman/collection","status":"success","startTime":"2021-09-28T14:13:11.293Z","endTime":"2021-09-28T14:13:45.271Z"},{"id":"61532298f47de75f31ae7a04","name":"","scriptName":"long-1","scriptType":"postman/collection","status":"success","startTime":"2021-09-28T14:11:39.179Z","endTime":"2021-09-28T14:12:15.202Z"},{"id":"6151b4b342189df67944968e","name":"","scriptName":"postman-test-7f6qrm","scriptType":"postman/collection","status":"success","startTime":"2021-09-27T12:10:31.581Z","endTime":"2021-09-27T12:10:32.105Z"},{"id":"6151b49d42189df67944968c","name":"","scriptName":"curl-test","scriptType":"curl/test","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-09-27T12:10:06.954Z"},{"id":"6151b41742189df67944968a","name":"","scriptName":"curl-test","scriptType":"curl/test","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-09-27T12:07:52.893Z"},{"id":"6151b41342189df679449688","name":"","scriptName":"curl-test","scriptType":"curl/test","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-09-27T12:07:48.868Z"},{"id":"6151b40f42189df679449686","name":"","scriptName":"curl-test","scriptType":"curl/test","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-09-27T12:07:44.89Z"},{"id":"6151b40b42189df679449684","name":"","scriptName":"curl-test","scriptType":"curl/test","status":"error","startTime":"0001-01-01T00:00:00Z","endTime":"2021-09-27T12:07:41.168Z"}]}
```

#### **Go-Template**

To get a simple list of script excution IDs with their corresponding statuses - use a Go template:

```sh
kubectl testkube get executions -ogo --go-template '{{.Id}}:{{.Status}} '

 615d7e1ab046f8fbd3d955d6:success 615d6398b046f8fbd3d955d4:success 615d5372b046f8fbd3d955d2:success 615d5265b046f8fbd3d955d0:error 615d4fe6b046f8fbd3d955ce:error 615d4556b046f8fbd3d955cc:error 615d43d3b046f8fbd3d955ca:error 6155cd7db046f8fbd3d955c8:success 6155cd67b046f8fbd3d955c6:error 615322f3f47de75f31ae7a06:success 61532298f47de75f31ae7a04:success 6151b4b342189df67944968e:success 6151b49d42189df67944968c:error 6151b41742189df67944968a:error 6151b41342189df679449688:error 6151b40f42189df679449686:error 6151b40b42189df679449684:error

```

### **Getting a List of Executions of a Given Test**

When there are multiple test cases, get executions of particular test by passing the script name as a parameter:

```sh
kubectl testkube get executions api-incluster-test
```

Output:

```sh
        SCRIPT       |        TYPE        | NAME |            ID            | STATUS   
+--------------------+--------------------+------+--------------------------+---------+
  api-incluster-test | postman/collection |      | 615d6398b046f8fbd3d955d4 | success  
  api-incluster-test | postman/collection |      | 615d7e1ab046f8fbd3d955d6 | success  
```
