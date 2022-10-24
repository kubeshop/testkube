const testsData = require('../../data/tests.json')

class TestDataHandler {
    getTest(testName) {
        cy.log('TestDataHandler getTest')
        return testsData[testName]
    }
}
export default TestDataHandler