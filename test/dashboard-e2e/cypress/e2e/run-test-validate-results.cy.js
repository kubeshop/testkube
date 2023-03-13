/// <reference types="cypress" />
/// <reference types="@cypress/xpath" />

import TestDataHandler from '../support/data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import ApiHelpers from '../support/api/api-helpers';
const apiHelpers=new ApiHelpers(Cypress.env('API_URL'));
import MainPage from '../support/pages/MainPage';
const mainPage=new MainPage();
import TestExecutionsPage from '../support/pages/TestExecutionsPage';
const testExecutionsPage=new TestExecutionsPage();


function runTestFlow(testName) {
  const testData = testDataHandler.getTest(testName)
  const realTestName = testData.name
  let currentExecutionNumber;

  //prerequisites
  const assureTestCreated = apiHelpers.assureTestCreated(testData, false) //API helpers using async/await must be wrapped
  cy.wrap(assureTestCreated)
  .then(() => {
    //actions
    const getLastExecutionNumber = apiHelpers.getLastExecutionNumber(realTestName)
    cy.wrap(getLastExecutionNumber)
    .then((lastExecutionNumber) => {
      mainPage.visitMainPage()
      mainPage.openTestExecutionDetails(realTestName)
      testExecutionsPage.runTest()
      currentExecutionNumber = lastExecutionNumber+1
      const executionName = `${realTestName}-${currentExecutionNumber}`
      testExecutionsPage.openExecutionDetails(executionName) //openLatestExecutionDetails?
    })
  })
}

describe('Run test with Dashboard', () => {
  it('Run Cypress test from git-dir', () => {
    runTestFlow('cypress-git-dir-created')

    testExecutionsPage.validateLogOutputContents('smoke-without-envs.cy.js', 120000)
    testExecutionsPage.validateLogOutputContents('All specs passed', 20000)

    //TODO: validate passed/failed icon - data-test needed
  })
  it('Run K6 test from git-file', () => {
    runTestFlow('k6-git-file-created')

    testExecutionsPage.validateLogOutputContents('script: test/k6/executor-tests/k6-smoke-test-without-envs.js', 60000)
    testExecutionsPage.validateLogOutputContents('1 complete and 0 interrupted iterations', 20000)

    //TODO: validate passed/failed icon - data-test needed
  })
  it('Run Postman test from git-file', () => {
    runTestFlow('postman-git-file-created')
    
    testExecutionsPage.validateLogOutputContents('postman-executor-smoke', 60000)
    testExecutionsPage.validateLogOutputContents('GET https://testkube.kubeshop.io/ [200 OK', 20000)

    //TODO: validate passed/failed icon - data-test needed
  })
})


describe('Run test with Dashboard - Negative cases', () => {
  it('Test results - test failure', () => {
    runTestFlow('postman-negative-test')

    testExecutionsPage.validateLogOutputContents('postman-executor-smoke', 60000)
    testExecutionsPage.validateLogOutputContents(`'TESTKUBE_POSTMAN_PARAM' should be set correctly to 'TESTKUBE_POSTMAN_PARAM_value' value`, 20000)
    testExecutionsPage.validateLogOutputContents('AssertionError', 20000)

    //TODO: validate passed/failed icon - data-test needed
  })
  it.skip('Test results - test init failure', () => { // temporary disabled because of https://github.com/kubeshop/testkube/issues/2669 issue
    runTestFlow('postman-negative-init')

    testExecutionsPage.validateLogOutputContents('warning: Could not find remote branch some-non-existing-branch to clone', 60000)
    testExecutionsPage.validateLogOutputContents('Remote branch some-non-existing-branch not found', 20000)
    
    //TODO: validate passed/failed icon - data-test needed
  })
})