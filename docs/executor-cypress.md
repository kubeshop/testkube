# Cypress Tests

TestKube is able to make simple runs of Cypress tests. As Cypress is organised in projects we allow to define your tests in Github repository (for now only public one is implemented). 

To create new cypress test you need some Git repository with example cypress project (please follow Cypress docs for details - https://docs.cypress.io/guides/dashboard/projects)


## Creating new test 

Let's assume we've created one in https://github.com/kubeshop/testkube-executor-cypress/tree/main/examples 
which contains really simple test which checks if some string exists on web site, we'll also check 
if env parameter exists - to show how to pass additional parameters into test.

https://github.com/kubeshop/testkube-executor-cypress/blob/main/examples/cypress/integration/simple-test.js
```js
describe('The Home Page', () => {
  it('successfully loads', () => {
    cy.visit('https://testkube.io') 

    expect(Cypress.env('testparam')).to.equal('testvalue')

    cy.contains('Efficient testing of k8s applications')
  })
})
```

## Creating `testkube` test script

Now we need to create TestKube test script from this example (we need to pass repo, branch, path where project exists - in case of mono repo, name and type)

```sh 
kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name kubeshop-cypress --type cypress/project
```

We can check that script is created with:
```
kubectl get scripts 
NAME                  AGE
kubeshop-cypress      51s
```

## Starting test 

Now we can start our test

```
kubectl testkube scripts start kubeshop-cypress

██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████
██  ██  ██    ██ ██   ██    ██    ██      ██         ██
█████   ██    ██ ██████     ██    █████   ███████    ██
██  ██  ██    ██ ██   ██    ██    ██           ██    ██
██   ██  ██████  ██████     ██    ███████ ███████    ██
                                 /kʌb tɛst/ by Kubeshop


Type          : cypress/project
Name          : kubeshop-cypress
Execution ID  : 615d5265b046f8fbd3d955d0
Execution name: wildly-popular-worm

Script queued for execution
Use following command to get script execution details:
$ kubectl testkube scripts execution 615d5265b046f8fbd3d955d0

or watch script execution until complete:
$ kubectl testkube scripts watch 615d5265b046f8fbd3d955d0
```

## Getting execution results

Let's watch our script execution 

```
kubectl testkube scripts watch 615d43d3b046f8fbd3d955ca

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



Script execution completed in 1m17s

```

## Adding parameters

We can see that our test was failed because of  `AssertionError: expected undefined to equal 'testvalue'` 

We forgot to add test param - let's fix that! In our test we assume that param will have name `testparam` and its value will be `testvalue` - so let's add that.  

We can also add `-f` flag to follow execution (watch for changes) - for now we're only looking for test completion but in future we'll pipe test output in real time (ongoing features)

```
kubectl testkube scripts start kubeshop-cypress -p testparam=testvalue -f

██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████
██  ██  ██    ██ ██   ██    ██    ██      ██         ██
█████   ██    ██ ██████     ██    █████   ███████    ██
██  ██  ██    ██ ██   ██    ██    ██           ██    ██
██   ██  ██████  ██████     ██    ███████ ███████    ██
                                 /kʌb tɛst/ by Kubeshop


Type          : cypress/project
Name          : kubeshop-cypress
Execution ID  : 615d5372b046f8fbd3d955d2
Execution name: nominally-able-glider

Script queued for execution
Use following command to get script execution details:
$ kubectl testkube scripts execution 615d5372b046f8fbd3d955d2

or watch script execution until complete:
$ kubectl testkube scripts watch 615d5372b046f8fbd3d955d2


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

Use following command to get script execution details:
$ kubectl testkube scripts execution 615d5372b046f8fbd3d955d2

Script execution completed in 1m45.405939s
```

## Summary

Our first test completed successfully! As we've seen above it's really easy to run Cypress tests with TestKube!