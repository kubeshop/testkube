import { expect } from 'chai';

class OutputValidators {
    removeAnsiCodes(rawOutput) {
        const output = rawOutput.replace(/[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g, '');

        return output
    }

    normalizeSpaces(output) {
        return output.replace(/\s+/g, ' ').trim()
    }

    validateTestCreated(testName, output) {
        const testCreatedText = `Test created testkube / ${testName}`

        expect(output).to.include(testCreatedText)
    }

    validateTestRunStarted(testData, output) {
        const normalizedOutput = this.normalizeSpaces(output)

        const typeText = `Type: ${testData.type}`
        const nameText = `Name: ${testData.name}`
        const statusText = `Status: running`
        const testExecutionStartedText = 'Test execution started'
        
        expect(normalizedOutput).to.include(typeText)
        expect(normalizedOutput).to.include(nameText)
        expect(normalizedOutput).to.include(statusText)
        expect(normalizedOutput).to.include(testExecutionStartedText)
    }

    validateTestExecutionSummary(executionData, output) {
        const normalizedOutput = this.normalizeSpaces(output)

        for (let key in executionData) {
            var value = executionData[key];

            if(key == 'Name') { //special case because of this bug: https://github.com/kubeshop/testkube/issues/2655
                expect(normalizedOutput).to.include(`${key} ${value}`)
            } else {
                expect(normalizedOutput).to.include(`${key}: ${value}`)
            }
        }
    }

    getExecutionId(output) {
        const normalizedOutput = this.normalizeSpaces(output)

        const executionIdRegex = /Execution ID:\s(?<id>\w+)/gm;
        const executionId = executionIdRegex.exec(normalizedOutput).groups.id

        return executionId
    }
}
export default OutputValidators