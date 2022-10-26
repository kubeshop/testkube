import superagent from 'superagent'

class ApiHelpers {
    async getTests() {
        const response = await superagent.get(`${Cypress.env('API_URL')}/tests`) //200

        return response.body
    }
<<<<<<< HEAD

    async createTest(testData) {
        const response = await superagent.post(`${Cypress.env('API_URL')}/tests`) //201
        .set('Content-Type', 'application/json')
        .send(testData)

        return response.body
    }
=======
>>>>>>> origin/main
    
    async removeTest(testName) {
        await superagent.delete(`${Cypress.env('API_URL')}/tests/${testName}`) //204
    }

<<<<<<< HEAD
    // async updateTest(testData) { //TODO

    // }

=======
>>>>>>> origin/main
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

<<<<<<< HEAD
    async assureTestCreated(testData, fullCleanup=true) {
        const alreadyCreated = await this.isTestCreated(testData.name)

        // if(alreadyCreated) {
        //     if(fullCleanup) {
        //         await removeTest(testData.name)
        //         await createTest(testData)
        //     }// else { //TODO
        //         //update
        //         // await updateTest(testData)
        //    // }
        // } else {
        //     await createTest(testData)
        // }
    }

=======
>>>>>>> origin/main
    async getTestData(testName) {
        const response = await superagent.get(`${Cypress.env('API_URL')}/tests/${testName}`) //200

        return response.body
    }
}
export default ApiHelpers