//TODO: common module for both cli-tests and Dashboard E2E tests?

import superagent from 'superagent'
import {setTimeout} from "timers/promises";


class ApiHelpers {
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

    async runTest(testName) {
        const request = `${this.API_URL}/tests/${testName}/executions`

        try {
            const response = await superagent.post(request)
            .set('Content-Type', 'application/json')
            .send({"namespace":"testkube"})
    
            const executionName = response.body.name
    
            return executionName
        } catch (e) {
            throw Error(`runTest failed on "${request}" with: "${e}"`)
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

    async getExecution(executionName) {
        const request = `${this.API_URL}/executions/${executionName}`

        try {
            const response = await superagent.get(request)
        
            return response.body
        } catch(e) {
            throw Error(`getExecution failed on "${request}" with: "${e}"`)
        }
    }

    async getExecutionStatus(executionName) {
        try {
            const execution = await this.getExecution(executionName)
            const executionStatus = execution.executionResult.status
    
            return executionStatus
        } catch (e) {
            throw Error(`getExecutionStatus failed for "${executionName}" with: "${e}"`)
        }
    }

    async getExecutionArtifacts(executionId) {
        const request = `${this.API_URL}/executions/${executionId}/artifacts`

        try {
            const response = await superagent.get(request)

            return response.body
        } catch (e) {
            throw Error(`getExecutionArtifacts failed for "${request}" with: "${e}"`)
        }
    }

    async downloadArtifact(executionId, artifactFileName) {
        const request = `${this.API_URL}/executions/${executionId}/artifacts/${artifactFileName}`

        try {
            const response = await superagent.get(request)
            const artifactContents = response.text
        
            return artifactContents
        } catch(e) {
            throw Error(`downloadArtifact failed on "${request}" with: "${e}"`)
        }
    }

    async waitForExecutionFinished(executionName, timeout) {
        const startTime = Date.now();
        while (Date.now() - startTime < timeout) {
            let status = await this.getExecutionStatus(executionName)

            if(status == 'passed' || status == 'failed') {
                return status
            }

            await setTimeout(1000);
        }

        throw Error(`waitForExecutionFinished timed out for "${executionName}" execution`)
    }
}
export default ApiHelpers