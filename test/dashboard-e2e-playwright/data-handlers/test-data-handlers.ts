const testsData = require('../fixtures/tests.json')

export class TestDataHandler {
    getTest(testName) {
        return testsData[testName]
    }
}