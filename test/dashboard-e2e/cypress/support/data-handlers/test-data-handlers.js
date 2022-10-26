const testsData = require('../../fixtures/tests.json')

class TestDataHandler {
    getTest(testName) {
        return testsData[testName]
    }
}
export default TestDataHandler