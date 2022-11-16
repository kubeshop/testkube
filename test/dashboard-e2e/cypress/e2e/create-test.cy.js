/// <reference types="cypress" />
/// <reference types="@cypress/xpath" />

import TestDataHandler from '../support/data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import ApiHelpers from '../support/api/api-helpers';
const apiHelpers=new ApiHelpers(Cypress.env('API_URL'));
import CommonHelpers from '../support/common-helpers';
const commonHelpers=new CommonHelpers();
import MainPage from '../support/pages/MainPage';
const mainPage=new MainPage();
import CreateTestPage from '../support/pages/CreateTestPage';
const createTestPage=new CreateTestPage();

function createTestFlow(testName) {
  const testData = testDataHandler.getTest(testName)

  //prerequisites
  const assureTestNotCreated = apiHelpers.assureTestNotCreated(testData.name) //API helpers using async/await must be wrapped
  cy.wrap(assureTestNotCreated)
  .then(() => {
    //actions
    mainPage.visitMainPage()
    mainPage.openCreateTestDialog()
    createTestPage.createTest(testName)
    cy.url().should('eq', `${Cypress.config('baseUrl')}/tests/executions/${testData.name}`)
  })
  .then(() => {
    //validation
    const getTestData = apiHelpers.getTestData(testData.name)
    cy.wrap(getTestData).then((createdTestData) => {
      commonHelpers.validateTest(testData, createdTestData)
    })
  })
}

describe('Create test with Dashboard', () => {
  it('Create Cypress test from git-dir', () => {
    createTestFlow('cypress-git-dir')
  })

  it('Create K6 test from git-file', () => {
    createTestFlow('k6-git-file')
  })

  it('Create Postman test from git-file', () => {
    createTestFlow('postman-git-file')
  })
})
