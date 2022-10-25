import superagent from 'superagent'

class ApiHelpers {
    async getTests() {
        const response = await superagent.get(`${Cypress.env('API_URL')}/tests`) //200

        return response.body
    }

    async createTest(testData) {
        //TODO
    }
    
    async removeTest(testName) {
        await superagent.delete(`${Cypress.env('API_URL')}/tests/${testName}`) //204
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

    async assureTestCreated(testName) {
        const alreadyCreated = await this.isTestCreated(testName)

        if(!alreadyCreated) {
            await this.createTest(testName)
        }
    }

    async getTestData(testName) {
        const response = await superagent.get(`${Cypress.env('API_URL')}/tests/${testName}`) //200

        return response.body
    }
}
export default ApiHelpers