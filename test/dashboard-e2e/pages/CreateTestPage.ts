import type { Page } from  '@playwright/test';
import { TestDataHandler } from '../data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();

export class CreateTestPage{
    readonly page: Page
    constructor(page:Page){
        this.page=page
    }
    
    async createTest(testName) {
        await this._fillInTestDetails(testName)
        await this._clickCreateTestButton()
    }

    async selectTestType(testType) {
        await this.setSelectionSearch(testType, "testType")
    }

    async selectTestSource(contentData) {
        if(contentData.type == "git") {

            let repositoryData = contentData.repository

            await this.setSelectionSearch("Git", "testSource")
            for (let key in repositoryData) {
                var value = repositoryData[key];
                // cy.log(`${key}: ${value}`)
    
                if(key == 'type') {
                    continue
                }
    
                await this.setBasicInput(value, key)
            }

        }else {
            throw 'Type not supported by selectTestSource - extend CreateTestPage'
        }
    }

    async setBasicInput(value, inputName) {
        await this.page.locator(`input[id="test-creation_${inputName}"]`).fill(value)
    }

    async setSelectionSearch(value, inputName) {
        let firstWord = value.split(' ')[0] //workaround - otherwise search won't find it

        await this.page.locator(`input[id="test-creation_${inputName}"]`).fill(firstWord)
        await this.page.click(`div[class*="list-holder"] div[title="${value}"]`)
    }

    async _fillInTestDetails(testName) {
        const testData = testDataHandler.getTest(testName)
        await this.setBasicInput(testData.name, 'name')
        await this.selectTestType(testData.type)
        await this.selectTestSource(testData.content)
    }

    async _clickCreateTestButton() {
        await this.page.click('button[data-test="add-a-new-test-create-button"]')
    }
}