class ApiHelpers {
    // TODO: update URLs
    getTests() {
        // cy.log('getTests')
        return cy.request('http://localhost:8088/v1/tests').then((response) => {
            // cy.log('getTests then')
            return response.body
        })
        // return response.body
    }
    
    removeTest(testName) {
        // cy.log('removeTest')
        return cy.request('DELETE', `http://localhost:8088/v1/tests/${testName}`).then((response) => {
            // cy.log('removeTest then')
            return response
        })
    }

    isTestCreated(testName) {
        this.getTests().then((currentTests) => {
            const test = currentTests.find(singleTest => singleTest.name == testName)

            if(test != undefined) {
                return true
            }

            return false
        })
    }

    assureTestNotCreated(testName) {
        // cy.log(`assureTestNotCreated testName: "${testName}"`)
        return this.isTestCreated(testName).then((created) => {
            // cy.log('assureTestNotCreated then')
            if(created) {
                return this.removeTest()
            }
        })
        // cy.log('after isTestCreated result')
        // cy.log(result)


    }
}
export default ApiHelpers