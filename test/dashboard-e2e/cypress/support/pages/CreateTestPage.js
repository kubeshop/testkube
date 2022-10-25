import TestDataHandler from '../data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();

class CreateTestPage {
    createTest(testName) {
        this._fillInTestDetails(testName)
        this._clickCreateTestButton()
    }

    selectTestType(testType) {
        this.setSelectionSearch(testType, "testType")
    }

    selectTestSource(testSource) {
        let type = testSource.type
        const gui_type = {"git-file": "Git file", "git-dir": "Git directory"}

        if(testSource.type == "git-file" || testSource.type == "git-dir") {
            type = gui_type[testSource.type]
        }

        this.setSelectionSearch(type, "testSource")
        for (let key in testSource) {
            var value = testSource[key];
            cy.log(`${key}: ${value}`)

            if(key == 'type') {
                continue
            }

            this.setBasicInput(value, key)
        }
    }

    setBasicInput(value, inputName) {
        cy.get(`input[id="test-suite-creation_${inputName}"]`).type(value)
    }

    setSelectionSearch(value, inputName) {
        let firstWord = value.split(' ')[0] //workaround - otherwise search won't find it
        cy.get(`input[id="test-suite-creation_${inputName}"]`).type(firstWord)
        cy.get(`div[class*="list-holder"] div[title="${value}"]`).click()//TODO: data-test attribute needed - replace when it will be available
    }

    _fillInTestDetails(testName) {
        const testData = testDataHandler.getTest(testName)
        this.setBasicInput(testData.name, 'name')
        this.selectTestType(testData.type)
        this.selectTestSource(testData.testSource)
    }

    _clickCreateTestButton() {
        cy.get('form[id="test-suite-creation"] button[type="submit"]').click()//TODO: data-test attribute needed - replace when it will be available
    }
}
export default CreateTestPage