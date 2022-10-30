const { execSync } = require('node:child_process');

import ApiHelpers from '../helpers/api-helpers.js';
const apiHelpers=new ApiHelpers();
import TestDataHandler from '../helpers/test-data-handlers.js';
const testDataHandler=new TestDataHandler();


describe('Create test with CLI', function () {
    it('Create K6 test with git-file', async function () {
        const testName = 'k6-git-file'
        const testData = testDataHandler.getTest(testName)

        //prerequisites
        console.log('assureTestNotCreated')
        await apiHelpers.assureTestNotCreated(testData.name)

        //command
        console.log('execSync')
        let rawResult = execSync(`testkube create test --name ${testData.name} --type ${testData.type} --test-content-type ${testData.content.type} --git-uri ${testData.content.repository.uri} --git-branch ${testData.content.repository.branch} --git-path ${testData.content.repository.path} --label core-tests=${testData.labels['core-tests']}`); //TODO: command builder
        let result = rawResult.toString()

        console.log('result: ')
        console.log(result)

        //result
        console.log('isTestCreated')
        const isTestCreated = await apiHelpers.isTestCreated(testData.name)

        if(!isTestCreated) {
            throw Error("Test not created")
        }

        //TODO: validate test
    });
});
