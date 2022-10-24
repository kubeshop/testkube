/// <reference types="cypress" />

import TestDataHandler from '../support/data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import ApiHelpers from '../support/api/api-helpers';
const apiHelpers=new ApiHelpers();
import CommonHelpers from '../support/common-helpers';
const commonHelpers=new CommonHelpers();
import MainPage from '../support/pages/MainPage';
const mainPage=new MainPage();
import CreateTestPage from '../support/pages/CreateTestPage';
const createTestPage=new CreateTestPage();

describe('Create test with Dashboard', () => {
  // beforeEach(() => {

  // })
  
  it('Create K6 test from git-file', () => {
    const testName = "k6-git-file"
    const testData = testDataHandler.getTest(testName)

    //prerequisites
    const assureTestNotCreated = apiHelpers.assureTestNotCreated(testData.name)
    cy.wrap(assureTestNotCreated)
    .then(() => {
      //actions
      mainPage.visitMainPage()
      mainPage.openCreateTestDialog()
      createTestPage.createTest("k6-git-file")
      cy.url().should('eq', `http://localhost:8080/tests/executions/${testData.name}`) //TODO baseurl
    })
    .then(() => {
      //validation
      const getTestData = apiHelpers.getTestData(testData.name)
      cy.wrap(getTestData).then((createdTestData) => {
        commonHelpers.validateTest(testData, createdTestData)
      })
    })
  })
})
