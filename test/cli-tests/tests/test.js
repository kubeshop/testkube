import {execSync} from 'node:child_process'
import { expect } from 'chai';

import ApiHelpers from '../helpers/api-helpers';
const apiHelpers=new ApiHelpers();
import TestDataHandler from '../helpers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import OutputValidators from '../helpers/output-validators';
const outputValidators=new OutputValidators();

async function createTestFlow(testName) {
 
    const testData = testDataHandler.getTest(testName)

    //prerequisites
    console.log('assureTestNotCreated')
    await apiHelpers.assureTestNotCreated(testData.name)

    //command
    console.log('execSync')
    let rawOutput = execSync(`testkube create test --name ${testData.name} --type ${testData.type} --test-content-type ${testData.content.type} --git-uri ${testData.content.repository.uri} --git-branch ${testData.content.repository.branch} --git-path ${testData.content.repository.path} --label core-tests=${testData.labels['core-tests']}`); //TODO: command builder
    let output = rawOutput.toString()

    console.log('output: ')
    console.log(output)

    //validate command output
    outputValidators.validateTestCreated(testData.name, output)

    //validate result
    console.log('isTestCreated')
    const isTestCreated = await apiHelpers.isTestCreated(testData.name)

    expect(isTestCreated).to.be.true;
}

describe('Create test with CLI', function () {
    it('Create Cypress test with git-dir', async function () {
        const testName = 'cypress-git-dir'
        
        await createTestFlow(testName)
    });
    it('Create K6 test with git-file', async function () {
        const testName = 'k6-git-file'
        
        await createTestFlow(testName)
    });
    it('Create Postman test with git-file', async function () {
        const testName = 'postman-git-file'
        
        await createTestFlow(testName)
    });
});
