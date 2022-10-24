/// <reference types="cypress" />

import TestDataHandler from '../support/data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import ApiHelpers from '../support/api/api-helpers';
const apiHelpers=new ApiHelpers();
import MainPage from '../support/pages/MainPage';
const mainPage=new MainPage();
import CreateTestPage from '../support/pages/CreateTestPage';
const createTestPage=new CreateTestPage();

describe('Create test with Dashboard', () => {
  // beforeEach(() => {

  // })
  
  it('Create K6 test from git-file', () => {
    let testName = "k6-git-file"
    let test = testDataHandler.getTest(testName)

    
    cy.wrap(null).then(() => apiHelpers.assureTestNotCreated(test.name)); //workaround for async code



    mainPage.visitMainPage()
    mainPage.openCreateTestDialog()
    createTestPage.fillInTestDetails("k6-git-file")
  })
})
