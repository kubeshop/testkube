# Getting list of recent test executions

To get recent results simply call `tests executions` subcommand 

```sh

kubectl testkube testsuites executions
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


             ID            |  TEST NAME   |           EXECUTION NAME            | STATUS  | STEPS  
+--------------------------+--------------+-------------------------------------+---------+-------+
  61e1142465e59a318346512b | test-example | test-example.equally-enabled-heron  | success |     3  
  61e1136165e59a3183465125 | test-example | test-example.fairly-humble-tick     | success |     3  
  61dff61867326ad521b2a0d6 | test-example | test-example.verbally-merry-hagfish | success |     3  
  61dfe0de69b7bfcb9058dad0 | test-example | test-example.overly-exciting-krill  | success |     3  

```


# Getting single test execution

Now when you know test execution ID you can get results 

```sh 
kubectl testkube testsuites execution 61e1136165e59a3183465125 
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Name: test-example.fairly-humble-tick
Status: success

             STEP            | STATUS  |            ID            | ERROR  
+----------------------------+---------+--------------------------+-------+
  run test: testkube/test1 | success | 61e1136165e59a3183465127 |        
  delay 2000ms               | success |                          |        
  run test: testkube/test1 | success | 61e1136765e59a3183465129 |        



Use following command to get test execution details:
$ kubectl testkube testsuites execution 61e1136165e59a3183465125
```

TestSuite steps which are running workflows based on `Test` Custom Resources have test execution ID - you can get details of each in separate command: 

```sh 
kubectl testkube tests execution 61e1136165e59a3183465127Name: test-example-test1, Status: success, Duration: 4.677s

newman

TODO

→ Create TODO
  POST http://34.74.127.60:8080/todos [201 Created, 296B, 100ms]
  ✓  Status code is 201 CREATED
  ┌
  │ 'creating', 'http://34.74.127.60:8080/todos/50'
  └
  ✓  Check if todo item craeted successfully
  GET http://34.74.127.60:8080/todos/50 [200 OK, 291B, 8ms]

→ Complete TODO item
  ┌
  │ 'completing', 'http://34.74.127.60:8080/todos/50'
  └
  PATCH http://34.74.127.60:8080/todos/50 [200 OK, 290B, 8ms]

→ Delete TODO item
  ┌
  │ 'deleting', 'http://34.74.127.60:8080/todos/50'
  └
  DELETE http://34.74.127.60:8080/todos/50 [204 No Content, 113B, 7ms]
  ✓  Status code is 204 no content

┌─────────────────────────┬───────────────────┬──────────────────┐
│                         │          executed │           failed │
├─────────────────────────┼───────────────────┼──────────────────┤
│              iterations │                 1 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│                requests │                 4 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│            test-scripts │                 5 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│      prerequest-scripts │                 6 │                0 │
├─────────────────────────┼───────────────────┼──────────────────┤
│              assertions │                 3 │                0 │
├─────────────────────────┴───────────────────┴──────────────────┤
│ total run duration: 283ms                                      │
├────────────────────────────────────────────────────────────────┤
│ total data received: 353B (approx)                             │
├────────────────────────────────────────────────────────────────┤
│ average response time: 30ms [min: 7ms, max: 100ms, s.d.: 39ms] │
└────────────────────────────────────────────────────────────────┘

```