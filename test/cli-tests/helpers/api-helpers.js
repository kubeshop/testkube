//TODO: common module for both cli-tests and Dashboard E2E tests?

import superagent from 'superagent'

class ApiHelpers {
    API_URL = process.env.API_URL //TODO: constructor
    async getTests() {
        const response = await superagent.get(`${this.API_URL}/tests`) //200

        return response.body
    }

    async createTest(testData) {
        const response = await superagent.post(`${this.API_URL}/tests`) //201
        .set('Content-Type', 'application/json')
        .send(testData)

        return response.body
    }
    
    async removeTest(testName) {
        await superagent.delete(`${this.API_URL}/tests/${testName}`) //204
    }

    async updateTest(testData) {
        const response = await superagent.patch(`${this.API_URL}/tests/${testData.name}`) //200
        .set('Content-Type', 'application/json')
        .send(testData)

        return response.body
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

    async assureTestCreated(testData, fullCleanup=false) {
        const alreadyCreated = await this.isTestCreated(testData.name)

        if(alreadyCreated) {
            if(fullCleanup) {
                await this.removeTest(testData.name)
                await this.createTest(testData)
            } else {
                await this.updateTest(testData)
           }
        } else {
            await this.createTest(testData)
        }
    }

    async getTestData(testName) {
        const response = await superagent.get(`${this.API_URL}/tests/${testName}`) //200

        return response.body
    }

    async getLastExecutionNumber(testName) {
        const response = await superagent.get(`${this.API_URL}/tests/${testName}/executions`) //200
        const totalsResults = response.body.totals.results

        if(totalsResults == 0) {
            return totalsResults
        } else {
            const lastExecutionResults = response.body.results[0]

            return lastExecutionResults.number
        }
    }
}
export default ApiHelpers