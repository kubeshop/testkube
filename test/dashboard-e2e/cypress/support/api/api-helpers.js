import superagent from 'superagent'

class ApiHelpers {
    // TODO: update URLs
    getApiUrl() {
        return Cypress.env('API_URL')
    }

    async getTests() {
        const response = await superagent.get(`${this.getApiUrl()}/tests`) //200

        return response.body
    }
    
    async removeTest(testName) {
        await superagent.delete(`${this.getApiUrl()}/tests/${testName}`) //204
    }

    async isTestCreated(testName) {
        const currentTests = await this.getTests()
        const test = currentTests.find(singleTest => singleTest.name == testName)

        if(test != undefined) {
            return true
        }


        return false
    }

    async assureTestNotCreated(testName) {
        const alreadyCreated = await this.isTestCreated(testName)
        if(alreadyCreated) {
            await this.removeTest(testName)
        }

        return true
    }

    async getTestData(testName) {
        const response = await superagent.get(`${this.getApiUrl()}/tests/${testName}`) //200

        return response.body
    }
}
export default ApiHelpers