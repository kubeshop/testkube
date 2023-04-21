import testsData from '../fixtures/tests.json'

export class TestDataHandler {
    getTest(testName) {
        return testsData[testName]
    }
}