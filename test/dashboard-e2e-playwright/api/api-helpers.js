import superagent from 'superagent'

export class ApiHelpers {
    constructor(apiUrl) {
        this.API_URL = apiUrl;
    }

    async getTests() {
        const request = `${this.API_URL}/tests`

        try {
            const response = await superagent.get(request)

            return response.body
        } catch (e) {
            throw Error(`getTests failed on "${request}" with: "${e}"`)
        }
    }

    async createTest(testData) {
        const request = `${this.API_URL}/tests`
        
        try {
            const response = await superagent.post(request)
            .set('Content-Type', 'application/json')
            .send(testData)
    
            return response.body
        } catch (e) {
            throw Error(`createTest failed on "${request}" with: "${e}"`)
        }
    }

    async abortTest(testName, executionId) {
        const request = `${this.API_URL}/tests/${testName}/executions/${executionId}`

        try {
            const response = await superagent.patch(request)

            return response
        } catch (e) {
            throw Error(`abortTest failed on "${request}" with: "${e}"`)
        }
    }
    
    async removeTest(testName) {
        const request = `${this.API_URL}/tests/${testName}`

        try {
            await superagent.delete(request)
        } catch (e) {
            throw Error(`removeTest failed on "${request}" with: "${e}"`)
        }
    }

    async updateTest(testData) {
        const request = `${this.API_URL}/tests/${testData.name}`
        
        try {
            const response = await superagent.patch(request)
            .set('Content-Type', 'application/json')
            .send(testData)
    
            return response.body
        } catch (e) {
            throw Error(`updateTest failed on "${request}" with: "${e}"`)
        }
    }

    async isTestCreated(testName) {
        try {
            const currentTests = await this.getTests()
            const test = currentTests.find(singleTest => singleTest.name == testName)
    
            if(test != undefined) {
                return true
            }
    
            return false
        } catch (e) {
            throw Error(`isTestCreated failed for "${testName}" with: "${e}"`)
        }
    }

    async assureTestNotCreated(testName) {
        try {
            const alreadyCreated = await this.isTestCreated(testName)
            if(alreadyCreated) {
                await this.removeTest(testName)
            }
    
            return true
        } catch (e) {
            throw Error(`assureTestNotCreated failed for "${testName}" with: "${e}"`)
        }
    }

    async assureTestCreated(testData, fullCleanup=false) {
        try {
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
        } catch (e) {
            throw Error(`assureTestCreated failed for "${testData.name}" with: "${e}"`)
        }
    }

    async getTestData(testName) {
        const request = `${this.API_URL}/tests/${testName}`

        try {
            const response = await superagent.get(request)

            return response.body
        } catch (e) {
            throw Error(`getTestData failed on "${request}" with: "${e}"`)
        }
    }

    async getLastExecutionNumber(testName) {
        const request = `${this.API_URL}/tests/${testName}/executions`

        try {
            const response = await superagent.get(request)
            const totalsResults = response.body.totals.results
    
            if(totalsResults == 0) {
                return totalsResults
            } else {
                const lastExecutionResults = response.body.results[0]
    
                return lastExecutionResults.number
            }
        } catch (e) {
            throw Error(`getLastExecutionNumber failed on "${request}" with: "${e}"`)
        }
    }
}