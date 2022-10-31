import { expect } from 'chai';

class OutputValidators {
    removeAnsiCodes(rawOutput) {
        const output = rawOutput.replace(/[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g, '');

        return output
    }

    validateTestCreated(testName, output) {
        const testCreatedText = `Test created testkube / ${testName}`
        const cleanOutput = this.removeAnsiCodes(output)

        expect(cleanOutput).to.include(testCreatedText)
    }
}
export default OutputValidators