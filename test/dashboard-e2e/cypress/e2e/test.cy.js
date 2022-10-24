/// <reference types="cypress" />

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

    apiHelpers.assureTestNotCreated(testName)//.then(() => {
      // mainPage.visitMainPage()
      // mainPage.openCreateTestDialog()
      // createTestPage.fillInTestDetails("k6-git-file")
    //})
  })
})
