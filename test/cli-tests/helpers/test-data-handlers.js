const testsData = require('../test-data/tests.json')

class TestDataHandler {
    getTest(testName) {
        return testsData[testName]
    }
}
export default TestDataHandler