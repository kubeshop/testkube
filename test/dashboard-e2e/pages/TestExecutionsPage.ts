import type { Page } from  '@playwright/test';

export class TestExecutionsPage{
    readonly page: Page
    constructor(page:Page){
        this.page=page
    }
    
    async runTest() {
        await this.page.click('div[class="ant-page-header-heading"] button')
    }
    
    async openExecutionDetails(executionName) {
        await this.page.click(`xpath=//tr[.//span[text()="${executionName}"]]`)
    }
}