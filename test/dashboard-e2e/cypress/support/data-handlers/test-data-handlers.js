<<<<<<< HEAD
const testsData = require('../../fixtures/tests.json')
=======
const testsData = require('../../data/tests.json')
>>>>>>> origin/main

class TestDataHandler {
    getTest(testName) {
        return testsData[testName]
    }
}
export default TestDataHandler