# Kubtest test scripts

Test scripts are single executor oriented tests. Script can have different types, which depends what executors are installed in your cluster. 

Kubtests includes `postman/collection`, `cypress/project` and `curl/test` script types which are auto registered during kubtest install by default. 

As Kubtest was designed with flexibility in mind - you can add your own executor which will handle additional script types. 


## Script test source

Scripts can be currently created from two sources: 
- First one is simple `file` with test content e.g. for postman collections we're exporting collection as json file, or for curl executor we're passing json file with configured curl command.
- Second source handled by kubtest is `git` - we can pass `repository`, `path` and `branch` where our tests is. This one is used in Cypress executor - as Cypress tests are more like npm-based projects which can have a lot of files. We're handling here sparse checkouts which are fast even in case of huge mono-repos 


## Create script

### Create your first script from file (Postman Collection test)

