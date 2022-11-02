import {execSync} from 'node:child_process'
import { expect } from 'chai';
import {setTimeout} from "timers/promises";

import ApiHelpers from '../helpers/api-helpers';
const apiHelpers=new ApiHelpers();
import TestDataHandler from '../helpers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import OutputValidators from '../helpers/output-validators';
const outputValidators=new OutputValidators();

describe('Get test results with CLI', function () {
    // it('Get cypress test results', async function () {
    //     this.timeout(30000);

    //     const testName = 'cypress-results-ran'
    //     const testData = testDataHandler.getTest(testName)

    //     //prerequisites
    //     await apiHelpers.assureTestCreated(testData)
    //     const executionName = await apiHelpers.runTest(testData.name)
    //     console.log('executionName: ')
    //     console.log(executionName)

    //     await apiHelpers.waitForExecutionFinished(executionName)

    // });
    it('Get K6 test results', async function () {
        const testName = 'k6-results-ran'
        this.timeout(30000);
        const waitForExecutionTimeout = 20000
        
        
        
        const testData = testDataHandler.getTest(testName)

        //prerequisites
        await apiHelpers.assureTestCreated(testData)
        const executionName = await apiHelpers.runTest(testData.name)

        await apiHelpers.waitForExecutionFinished(executionName, waitForExecutionTimeout)

    });
    // it('Get Postman test results', async function () {
    //     const testName = 'postman-results-ran'
        
    //     const testData = testDataHandler.getTest(testName)

    //     //prerequisites
    //     await apiHelpers.assureTestCreated(testData)
    //     const executionName = await apiHelpers.runTest(testData.name)
    //     console.log('executionName: ')
    //     console.log(executionName)

    //     await apiHelpers.waitForExecutionFinished(executionName)
    // });
});

describe('Get test results with CLI - Negative cases', function () {
    // it('Get test results - test failure', async function () {
    //     const testName = 'postman-results-ran-negative-test'
        
    //     // await createTestFlow(testName)
    // });
    // it('Get test results - test failure', async function () {
    //     const testName = 'postman-results-ran-negative-init'
        
    //     // await createTestFlow(testName)
    // });
});
