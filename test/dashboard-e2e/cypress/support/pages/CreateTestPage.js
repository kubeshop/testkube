import TestDataHandler from '../data-handlers/test-data-handlers';
const testDataHandler=new TestDataHandler();

class CreateTestPage {
    fillInTestDetails(testName) {
        cy.log('CreateTestPage fillInTestDetails')
        let test = testDataHandler.getTest(testName)
        cy.log(test)
        this.setBasicInput(test.name, 'name')
        this.selectTestType(test.testType)
        this.selectTestSource(test.testSource)
        this._clickCreateTestButton()
    }

    selectTestType(testType) {
        cy.log(`selectTestType testType: ${testType}`)
        this.setSelectionSearch(testType, "testType")
    }

    selectTestSource(testSource) {
        this.setSelectionSearch(testSource.type, "testSource")
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
        cy.get(`input[id="test-suite-creation_${inputName}"]`).type(value) //TODO: move selectors
    }

    setSelectionSearch(value, inputName) {
        cy.log(`setSelectionSearch value: ${value}, inputName: ${inputName}`)
        let firstWord = value.split(' ')[0] //workaround - otherwise search won't find it
        cy.get(`input[id="test-suite-creation_${inputName}"]`).type(firstWord)
        cy.get(`div[class*="list-holder"] div[title="${value}"]`).click()
    }

    _clickCreateTestButton() {
        cy.get('form[id="test-suite-creation"] button[type="submit"]').click()
    }
}
export default CreateTestPage



//test-suite-creation
//test-suite-creation_name

//TODO: label, testSource data-test