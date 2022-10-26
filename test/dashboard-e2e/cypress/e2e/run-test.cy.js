/// <reference types="cypress" />
/// <reference types="@cypress/xpath" />

import TestDataHandler from '../support/data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import ApiHelpers from '../support/api/api-helpers';
const apiHelpers=new ApiHelpers();
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
      testExecutionsPage.openExecutionDetails(executionName)
      //TODO check execution details (after data-test)
    })
  })
  .then(() => {
    //validation
    const getLastExecutionNumber = apiHelpers.getLastExecutionNumber(realTestName)
    cy.wrap(getLastExecutionNumber)
    .then((lastExecutionNumber) => {
      cy.expect(currentExecutionNumber).to.equal(lastExecutionNumber)
    })
  })
}

describe('Run test with Dashboard', () => {
  it('Run Cypress test from git-dir', () => {
    runTestFlow('cypress-git-dir-created')
  })
  it('Run K6 test from git-file', () => {
    runTestFlow('k6-git-file-created')
  })
  it('Run Postman test from git-file', () => {
    runTestFlow('postman-git-file-created')
  })
})
