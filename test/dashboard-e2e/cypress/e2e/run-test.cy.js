/// <reference types="cypress" />

import TestDataHandler from '../support/data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import ApiHelpers from '../support/api/api-helpers';
const apiHelpers=new ApiHelpers();
import MainPage from '../support/pages/MainPage';
const mainPage=new MainPage();


describe('Create test with Dashboard', () => {
  it('Run Cypress test from git-dir', () => {
    const testName = 'cypress-git-dir-created'
    const testData = testDataHandler.getTest(testName)
    const realTestName = testData.name

    mainPage.visitMainPage()
    mainPage.openTestExecutionDetails(realTestName)
  })
})
