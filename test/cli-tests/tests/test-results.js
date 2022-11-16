import {execSync} from 'node:child_process'
import { expect } from 'chai';

import ApiHelpers from '../helpers/api-helpers';
const apiHelpers=new ApiHelpers(process.env.API_URL);
import TestDataHandler from '../helpers/test-data-handlers';
const testDataHandler=new TestDataHandler();
import OutputValidators from '../helpers/output-validators';
const outputValidators=new OutputValidators();

async function getResultsPositiveFlow(testName, customRunSummary, waitForExecutionTimeout) {
    const testData = testDataHandler.getTest(testName)

    //prerequisites
    await apiHelpers.assureTestCreated(testData)
    const executionName = await apiHelpers.runTest(testData.name)
    const execution = await apiHelpers.getExecution(executionName)
    const executionId = execution.id

    await apiHelpers.waitForExecutionFinished(executionName, waitForExecutionTimeout)

    //command
    const rawOutput = execSync(`testkube get execution ${executionName}`);
    const output = rawOutput.toString()
    const cleanOutput = outputValidators.removeAnsiCodes(output)

    const expectedExecutionData = {
        "Name": executionName,
        "Test name": testData.name,
        "Type": testData.type,
        "Status": 'passed'
    }

    outputValidators.validateTestExecutionSummary(expectedExecutionData, cleanOutput)

    const normalizedOutput = outputValidators.normalizeSpaces(cleanOutput)
    expect(normalizedOutput).to.include(customRunSummary)

    const executionData = {
        "executionName": executionName,
        "executionId": executionId
    }

    return executionData
}

function assertArtifactExists(executionArtifacts, artifactFileName, notEmpty) {
    try {
        if(notEmpty) {
            expect(executionArtifacts.filter(obj => obj.name === artifactFileName)[0]).to.exist.and.to.contain.keys('name').and.to.contain.keys('size')
        } else {
            expect(executionArtifacts.filter(obj => obj.name === artifactFileName)[0]).to.exist
        }
    } catch(e) {
        throw Error(`Missing artifact "${artifactFileName}"`)
    }
}

describe('Get test results with CLI', function () { //Execution times are unpredictable - these tests require high timeouts!
    it('Get cypress test results', async function () {
        const testName = 'cypress-results-ran'
        this.timeout(120000);
        const waitForExecutionTimeout = 100000

        const customRunSummary = 'Passing: 1'
        await getResultsPositiveFlow(testName, customRunSummary, waitForExecutionTimeout)
    });
    it('Get K6 test results', async function () {
        const testName = 'k6-results-ran'
        this.timeout(60000);
        const waitForExecutionTimeout = 50000

        const customRunSummary = '1 complete and 0 interrupted iterations'
        await getResultsPositiveFlow(testName, customRunSummary, waitForExecutionTimeout)
    });
    it('Get Postman test results', async function () {
        const testName = 'postman-results-ran'
        this.timeout(60000);
        const waitForExecutionTimeout = 50000

        const customRunSummary = 'GET https://testkube.kubeshop.io/ [200 OK'
        await getResultsPositiveFlow(testName, customRunSummary, waitForExecutionTimeout)
    });
    it('Get SoapUI test results (including artifacts)', async function () {
        const testName = 'soapui-results-ran'
        this.timeout(60000);
        const waitForExecutionTimeout = 50000

        const customRunSummary = 'Project [soapui-smoke-test] finished with status [FINISHED]'
        const executionData = await getResultsPositiveFlow(testName, customRunSummary, waitForExecutionTimeout)

        // Artifacts
        const executionArtifacts = await apiHelpers.getExecutionArtifacts(executionData.executionId)

        assertArtifactExists(executionArtifacts, 'global-groovy.log')
        assertArtifactExists(executionArtifacts, 'soapui-errors.log')
        assertArtifactExists(executionArtifacts, 'soapui.log', true)

        const soapUiLogContents = await apiHelpers.downloadArtifact(executionData.executionId, 'soapui.log')
        const soapUiErrorLogContents = await apiHelpers.downloadArtifact(executionData.executionId, 'soapui-errors.log')

        expect(soapUiLogContents).to.include(customRunSummary)
        expect(soapUiErrorLogContents).to.be.empty
    });
});

describe('Get test results with CLI - Negative cases', function () { //Execution times are unpredictable - these tests require high timeouts!
    it('Get test results - test failure', async function () {
        const testName = 'postman-results-ran-negative-test'
        this.timeout(60000);
        const waitForExecutionTimeout = 50000

        const testData = testDataHandler.getTest(testName)

        //prerequisites
        await apiHelpers.assureTestCreated(testData)
        const executionName = await apiHelpers.runTest(testData.name)
    
        await apiHelpers.waitForExecutionFinished(executionName, waitForExecutionTimeout)
    
        //command
        try {
            const result = execSync(`testkube get execution ${executionName}`, {stdio : 'pipe'})
        } 
        catch (error) {
            //validation
            expect(error.status).to.not.equal(0)

            const rawOutput = error.stdout.toString()
            const errStr = error.stderr.toString()
            const cleanOutput = outputValidators.removeAnsiCodes(rawOutput)

            const expectedExecutionData = {
                "Name": executionName,
                "Test name": testData.name,
                "Type": testData.type,
                "Status": 'failed'
            }

            outputValidators.validateTestExecutionSummary(expectedExecutionData, cleanOutput)

            expect(errStr).to.include('Test execution failed')
            expect(errStr).to.include("'TESTKUBE_POSTMAN_PARAM' should be set correctly to 'TESTKUBE_POSTMAN_PARAM_value' value")

            return
        }

        throw Error('Execution was expected to fail')
    });
    it('Get test results - test init failure', async function () {
        const testName = 'postman-results-ran-negative-init'
        this.timeout(60000);
        const waitForExecutionTimeout = 50000

        const testData = testDataHandler.getTest(testName)

        //prerequisites
        await apiHelpers.assureTestCreated(testData)
        const executionName = await apiHelpers.runTest(testData.name)
    
        await apiHelpers.waitForExecutionFinished(executionName, waitForExecutionTimeout)
    
        //command
        try {
            execSync(`testkube get execution ${executionName}`, {stdio : 'pipe'}) // Expected failure
        } 
        catch (error) {
            //validation
            expect(error.status).to.not.equal(0)

            const rawStdoutStr = error.stdout.toString()
            const errStr = error.stderr.toString()
            const cleanOutput = outputValidators.removeAnsiCodes(rawStdoutStr)

            const expectedExecutionData = {
                "Name": executionName,
                "Test name": testData.name,
                "Type": testData.type,
                "Status": 'failed'
            }

            outputValidators.validateTestExecutionSummary(expectedExecutionData, cleanOutput)

            expect(errStr).to.include('Test execution failed')
            
            expect(errStr).to.include('process error: exit status 128')
            expect(errStr).to.include('fatal: Remote branch some-non-existing-branch not found')

            return
        }

        throw Error('Execution was expected to fail')
    });
});
