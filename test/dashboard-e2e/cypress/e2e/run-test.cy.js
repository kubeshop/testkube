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


describe('Run test with Dashboard', () => {
  it('Run K6 test from git-file', () => {

    const testName = 'k6-git-file-created'
    const testData = testDataHandler.getTest(testName)
    const realTestName = testData.name

    //prerequisites
    const assureTestCreated = apiHelpers.assureTestCreated(testData, false) //API helpers using async/await must be wrapped
    cy.wrap(assureTestCreated)
    .then(() => {
      //actions
      const getLastExecutionNumber = apiHelpers.getLastExecutionNumber(realTestName)
      cy.wrap(getLastExecutionNumber)
      .then((lastExecutionNumber) => {
        window.alert('lastExecutionNumber: ')
        window.alert(lastExecutionNumber)
        mainPage.visitMainPage()
        mainPage.openTestExecutionDetails(realTestName)
        testExecutionsPage.runTest()
        const executionName = `${realTestName}-${lastExecutionNumber+1}`
        testExecutionsPage.openExecutionDetails(executionName)
        //TODO check execution details (after data-test)
      })
    })
    .then(() => {
      //validation
      //TODO
    })
  })
})
