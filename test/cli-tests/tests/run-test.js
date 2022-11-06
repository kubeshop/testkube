import {execSync} from 'node:child_process'
import { expect } from 'chai';

import ApiHelpers from '../helpers/api-helpers';
const apiHelpers=new ApiHelpers();
import TestDataHandler from '../helpers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import OutputValidators from '../helpers/output-validators';
const outputValidators=new OutputValidators();

async function runTestFlow(testName) {
    const testData = testDataHandler.getTest(testName)

    //prerequisites
    await apiHelpers.assureTestCreated(testData)

    //command
    const rawOutput = execSync(`testkube run test ${testData.name}`); //TODO: command builder
    const output = rawOutput.toString()
    const cleanOutput = outputValidators.removeAnsiCodes(output)

    //validate command output
    outputValidators.validateTestRunStarted(testData, cleanOutput)

    //validate result
    const executionId = outputValidators.getExecutionId(cleanOutput)
    const executionStatus = await apiHelpers.getExecutionStatus(executionId)

    expect(executionStatus).to.be.equal('running')
}

describe('Run test with CLI', function () {
    it('Run Cypress test with git-dir', async function () {
        const testName = 'cypress-git-dir-created'
        
        await runTestFlow(testName)
        //TODO: abort the test not to waste resources
    });
    it('Run K6 test with git-file', async function () {
        const testName = 'k6-git-file-created'
        
        await runTestFlow(testName)
    });
    it('Run Postman test with git-file', async function () {
        const testName = 'postman-git-file-created'
        
        await runTestFlow(testName)
    });
});
