import type { Page } from  '@playwright/test';
export class MainPage{
    readonly page: Page
    constructor(page:Page){
        this.page=page
    }

    async visitMainPage(){
      await this.page.goto(`/apiEndpoint?apiEndpoint=${process.env.API_URL}`);
    
      await this.page.addInitScript(() => {
        window.localStorage.setItem('isGADisabled', '1');
      });
    }

    async openCreateTestDialog() {
      await this.page.click('button[data-test="add-a-new-test-btn"]')
    }

    async openTestExecutionDetails(realTestName) {
      await this.page.locator(`input[data-cy="search-filter"]`).fill(realTestName)
      await this.page.click(`xpath=//div[@data-test="tests-list-item" and .//span[text()="${realTestName}"]]`)
    }
}