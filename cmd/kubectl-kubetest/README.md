# Kubectl kubetest plugin

## Installation 

TODO 

## Usage 

1) Starting new script execution 

```
$ kubectl kubetest scripts start SOME_SCRIPT_ID_DEFINED_IN_CR

Script "SCRIPTNAME" started
Execution ID 02wi02-29329-2392930-93939
```

Possible todo items:
- [ ] watch for results immediately ? 
- [ ] show some output from run ?
- [ ] maybe allow to name/describe your execution (will be easier to check) for example we can run execution for different server config when debugging some issue so we would have several executions (testing_128M testing_200M testing_256M testing_512M)?


2) Aborting already started script execution 
```
$ kubectl kubetest scripts abort SOME_EXECUTION_ID

Script "SCRIPTNAME" Execution aborted

```


3) Getting available scripts
```
$ kubectl kubetest scripts  list


ID         NAME              Type
040-134   HomePage test      postman-collection   
123-246   Contact API test   postman-collection

```
 
4) Getting available executions

```sh
kubectl kubetest scripts executions

ID         NAME             Status     Complete   Start              End
1233-333   HomePage run     pending    75%        2021-07-30 12:33   
1233-332   HomePage run     pending    100%       2021-07-30 12:33   2021-07-30 13:10
```


## What output renderers ? 
- plain text (for the beggining) but prepare model for adding additional renderers 
- json 
- go template 
