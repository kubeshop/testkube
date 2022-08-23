# Cypress Executors

Cypress is a next generation front end testing tool built for the modern web. 

Please visit the Cypress documentation for more detailed information - <https://docs.cypress.io/>.

TestKube makes running Cypress tests simple. As Cypress is organised in projects, Testkube allows tests to be defined in a Github repository.

To create a new Cypress test, you will need a Git repository with an example Cypress project - <https://docs.cypress.io/guides/dashboard/projects>.

## **Creating a New Test**

Let's assume we've created a Cypress project in <https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples>,
which contains a really simple test that checks for the existence of a particular string on a web site.  We'll also check
if the Cypress **env** parameter exists to show how to pass additional parameters into the test. 
particular executor.

<https://github.com/kubeshop/testkube-executor-cypress/blob/main/examples/cypress/integration/simple-test.js>

```js
describe('The Home Page', () => {
  it('successfully loads', () => {
    cy.visit('https://testkube.io') 

    expect(Cypress.env('testvariable')).to.equal('testvalue')

    cy.contains('Efficient testing of k8s applications')
  })
})
```

## **Creating the Testkube Test Script**

Create the Testkube test script from this example. The parameters passed are **repository**, **branch** and **the path where the project exists**. In the case of a mono repository, the parameters are **name** and **type**.

```sh
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project
```

Check that script is created:

```sh
kubectl get tests 
```

Output:

```sh
NAME                  AGE
kubeshop-cypress      51s
```

## **Starting the Test**

Start the test:

```sh
kubectl testkube run test kubeshop-cypress
```

Output:

```sh
Type          : cypress/project
Name          : kubeshop-cypress
Execution ID  : 615d5265b046f8fbd3d955d0
Execution name: wildly-popular-worm

Test queued for execution
Use the following command to get test execution details:
$ kubectl testkube get execution 615d5265b046f8fbd3d955d0

or watch test execution until complete:
$ kubectl testkube watch execution 615d5265b046f8fbd3d955d0
```

## **Getting Execution Results**

Let's watch our test execution:

```sh
kubectl testkube watch execution 615d43d3b046f8fbd3d955ca
```

Output:

```sh
Type          : cypress/project
Name          : cypress-example
Execution ID  : 615d43d3b046f8fbd3d955ca
Execution name: early-vast-turtle

Watching for changes
Status: error, Duration: 1m16s

Getting results
Name: early-vast-turtle, Status: error, Duration: 1m16s
process error: exit status 1
output:
====================================================================================================

  (Run Starting)

  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Cypress:    8.3.0                                                                              │
  │ Browser:    Electron 91 (headless)                                                             │
  │ Specs:      1 found (simple-test.js)                                                           │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘


────────────────────────────────────────────────────────────────────────────────────────────────────

  Running:  simple-test.js                                                                  (1 of 1)


  The Home Page
    1) successfully loads


  0 passing (2s)
  1 failing

  1) The Home Page
       successfully loads:
     AssertionError: expected undefined to equal 'testvalue'
      at Context.eval (http://localhost:34845/__cypress/tests?p=cypress/integration/simple-test.js:102:41)




  (Results)

  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Tests:        1                                                                                │
  │ Passing:      0                                                                                │
  │ Failing:      1                                                                                │
  │ Pending:      0                                                                                │
  │ Skipped:      0                                                                                │
  │ Screenshots:  1                                                                                │
  │ Video:        true                                                                             │
  │ Duration:     2 seconds                                                                        │
  │ Spec Ran:     simple-test.js                                                                   │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘


  (Screenshots)

  -  /tmp/testkube-scripts1127226423/repo/examples/cypress/screenshots/simple-test.js/     (1280x720)
     The Home Page -- successfully loads (failed).png


  (Video)

  -  Started processing:  Compressing to 32 CRF
    Compression progress:  35%
  -  Finished processing: /tmp/testkube-scripts1127226423/repo/examples/cypress/videos   (19 seconds)
                          /simple-test.js.mp4


====================================================================================================

  (Run Finished)


       Spec                                              Tests  Passing  Failing  Pending  Skipped
  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ ✖  simple-test.js                           00:02        1        -        1        -        - │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘
    ✖  1 of 1 failed (100%)                     00:02        1        -        1        -        -



Test execution completed in 1m17s

```

## **Adding Parameters**

The test failed because of `AssertionError: expected undefined to equal 'testvalue'`.

The test variable was not passed into the test script. In this test, the parameter will have the name `testvariable` and its value will be `testvalue`.   

Add the `-f` flag to follow the execution and watch for changes. Currently, we're only looking for test completion, but in the future, we'll pipe test output in real time.

```sh
kubectl testkube run test kubeshop-cypress -v testvariable=testvalue -f
```

Output:

```sh
Type          : cypress/project
Name          : kubeshop-cypress
Execution ID  : 615d5372b046f8fbd3d955d2
Execution name: nominally-able-glider

Test queued for execution
Use the following command to get test execution details:
$ kubectl testkube get execution 615d5372b046f8fbd3d955d2

or watch test execution until complete:
$ kubectl testkube watch execution 615d5372b046f8fbd3d955d2


Watching for changes
Status: queued, Duration: 0s
Status: pending, Duration: 383.064ms
....
Status: pending, Duration: 1m45.405939s
Status: success, Duration: 1m45.405939s

Getting results
Name: nominally-able-glider, Status: success, Duration: 2562047h47m16.854775807s

====================================================================================================

  (Run Starting)

  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Cypress:    8.5.0                                                                              │
  │ Browser:    Electron 91 (headless)                                                             │
  │ Specs:      1 found (simple-test.js)                                                           │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘


────────────────────────────────────────────────────────────────────────────────────────────────────

  Running:  simple-test.js                                                                  (1 of 1)

  (Results)

  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Tests:        1                                                                                │
  │ Passing:      1                                                                                │
  │ Failing:      0                                                                                │
  │ Pending:      0                                                                                │
  │ Skipped:      0                                                                                │
  │ Screenshots:  0                                                                                │
  │ Video:        true                                                                             │
  │ Duration:     19 seconds                                                                       │
  │ Spec Ran:     simple-test.js                                                                   │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘


  (Video)

  -  Started processing:  Compressing to 32 CRF
    Compression progress:  39%
    Compression progress:  81%
  -  Finished processing: /tmp/testkube-scripts531364188/repo/examples/cypress/videos/   (30 seconds)
                          simple-test.js.mp4

    Compression progress:  100%

====================================================================================================

  (Run Finished)


       Spec                                              Tests  Passing  Failing  Pending  Skipped
  ┌────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ ✔  simple-test.js                           00:19        1        1        -        -        - │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘
    ✔  All specs passed!                        00:19        1        1        -        -        -

Use the following command to get test execution details:
$ kubectl testkube get execution 615d5372b046f8fbd3d955d2

Test execution completed in 1m45.405939s
```


## Passing secret variables to tests

You can pass secret variables to test run, just use `-s` flag for that

```sh
kubectl testkube run test kubeshop-cypress -s secretvariable=S3cretv4lue -f
```

Testkube will create Kubernetes `Secret` resource corresponding to that value based on `executionId` 

```sh

kubectl testkube run test vartest2 -cdirect -f -s sec1=aaaa -s sec2=cnjksnkjsdncjksd

kubectl get secret -ntestkube 6284bc254ce130c4bd2fdc53-vars -oyaml

apiVersion: v1
data:
  sec1: YWFhYQ==
  sec2: Y25qa3Nua2pzZG5jamtzZA==
kind: Secret
metadata:
  creationTimestamp: "2022-05-19T06:25:41Z"
  labels:
    executionID: 6285e2e54403fb203d7e3ac0
    testName: vartest2
    testkube: tests-secrets
  name: 6285e2e54403fb203d7e3ac0-vars
  namespace: testkube
  resourceVersion: "64289430"
  uid: ef22268b-e1a9-4046-a1d1-3fb8c37dfb9e
type: Opaque
```

Only reference to this secret will be stored in Testkube internal storage.


## **Summary**

Our first test completed successfully! As we've seen above, it's really easy to run Cypress tests with Testkube!
