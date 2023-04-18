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

    selectTestSource(contentData) {
        if(contentData.type == "git") {

            let repositoryData = contentData.repository

            this.setSelectionSearch("Git", "testSource")
            for (let key in repositoryData) {
                var value = repositoryData[key];
                cy.log(`${key}: ${value}`)
    
                if(key == 'type') {
                    continue
                }
    
                this.setBasicInput(value, key)
            }

        }else {
            throw 'Type not supported by selectTestSource - extend CreateTestPage'
        }


    }

    setBasicInput(value, inputName) {
        cy.get(`input[id="test-creation_${inputName}"]`).type(value)
    }

    setSelectionSearch(value, inputName) {
        let firstWord = value.split(' ')[0] //workaround - otherwise search won't find it
        cy.get(`input[id="test-creation_${inputName}"]`).type(firstWord)
        cy.get(`div[class*="list-holder"] div[title="${value}"]`).click()//TODO: data-test attribute needed - replace when it will be available
    }

    _fillInTestDetails(testName) {
        const testData = testDataHandler.getTest(testName)
        this.setBasicInput(testData.name, 'name')
        this.selectTestType(testData.type)
        this.selectTestSource(testData.content)
    }

    _clickCreateTestButton() {
        cy.get('button[data-test="add-a-new-test-create-button"]').click()
    }
}
export default CreateTestPage