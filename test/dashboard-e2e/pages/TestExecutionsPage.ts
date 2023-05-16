import type { Page } from  '@playwright/test';

export class TestExecutionsPage{
    readonly page: Page
    constructor(page:Page){
        this.page=page
    }
    
    async runTest() {
        await this.page.click('//span[@class="ant-page-header-heading-extra"]//button[.//span]') //TODO: data-test needed
    }
    
    async openExecutionDetails(executionName) {
        await this.page.click(`xpath=//tr[.//span[text()="${executionName}"]]`)
    }
}