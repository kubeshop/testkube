import type { Page } from  '@playwright/test';
export class MainPage{
    readonly page: Page
    constructor(page:Page){
        this.page=page
    }

    async visitMainPage(){
      await this.page.goto('http://localhost:8080'); //TODO: temporary hardcoded
    }

    async openCreateTestDialog() {
        await this.page.click('button[data-test="add-a-new-test-btn"]')
    }

    async openTestExecutionDetails(realTestName) {
      await this.page.click(`xpath=//div[@data-test="tests-list-item" and .//span[text()="${realTestName}"]]`)
    }
}