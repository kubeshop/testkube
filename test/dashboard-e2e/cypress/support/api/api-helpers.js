import superagent from 'superagent'

class ApiHelpers {
    async getTests() {
        const response = await superagent.get(`${Cypress.env('API_URL')}/tests`) //200

        return response.body
    }

    async createTest(testData) {
        const response = await superagent.post(`${Cypress.env('API_URL')}/tests`) //201
        .set('Content-Type', 'application/json')
        .send(testData)

        return response.body
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

    async assureTestCreated(testData) {
        const alreadyCreated = await this.isTestCreated(testData.name)

        if(!alreadyCreated) {
            await this.createTest(testData)
        }
    }

    async getTestData(testName) {
        const response = await superagent.get(`${Cypress.env('API_URL')}/tests/${testName}`) //200

        return response.body
    }
}
export default ApiHelpers