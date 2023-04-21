import { expect } from '@playwright/test';

export class CommonHelpers {
    validateTest(testData, createdTestData) {
        expect(testData.name).toEqual(createdTestData.name)
        //TODO: label
        expect(testData.type).toEqual(createdTestData.type)
        expect(testData.content.type).toEqual(createdTestData.content.type)

        //testSources
        const contentType = testData.content.type
        if (contentType == "git") {
            for (let key in testData.content.repository) {
                expect(testData.content.repository[key]).toEqual(createdTestData.content.repository[key])
            }
        }
    }
}