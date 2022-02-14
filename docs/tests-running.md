# **Running Tests**

To run tests, pass the `tests run` command with the test name to your `kubectl testkube` plugin. Tests are started asynchronously by default.

```sh
kubectl testkube tests run test-example

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Name: test-example.fairly-humble-tick
Status: pending

  STEP | STATUS | ID | ERROR  
+------+--------+----+-------+



Use the following command to get test execution details:
$ kubectl testkube tests execution 61e1136165e59a3183465125


Use the following command to get test execution details:
$ kubectl testkube tests watch 61e1136165e59a3183465125
```

After the test starts, you can check current test status with `tests execution EXECUTION_ID`. 


# Running Tests Synchronously 

You can also start tests synchronously by passing the `-f` flag (like --follow) to your command.

```sh
kubectl testkube tests run test-example -f

████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop


Name: test-example.equally-enabled-heron
Status: pending

  STEP | STATUS | ID | ERROR  
+------+--------+----+-------+

...


             STEP            | STATUS  |            ID            | ERROR  
+----------------------------+---------+--------------------------+-------+
  run script: testkube/test1 | success | 61e1142465e59a318346512d |        


Name: test-example.equally-enabled-heron
Status: pending

             STEP            | STATUS  |            ID            | ERROR  
+----------------------------+---------+--------------------------+-------+
  run script: testkube/test1 | success | 61e1142465e59a318346512d |        
  delay 2000ms               | success |                          |        


...


Name: test-example.equally-enabled-heron
Status: success

             STEP            | STATUS  |            ID            | ERROR  
+----------------------------+---------+--------------------------+-------+
  run script: testkube/test1 | success | 61e1142465e59a318346512d |        
  delay 2000ms               | success |                          |        
  run script: testkube/test1 | success | 61e1142a65e59a318346512f |        



Use the following command to get the test execution details:
$ kubectl testkube tests execution 61e1142465e59a318346512b

```
