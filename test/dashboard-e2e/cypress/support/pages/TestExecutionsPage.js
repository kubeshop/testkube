class TestExecutionsPage {
    runTest() {
        cy.get('div[class="ant-page-header-heading"] button').click() //TODO: replace with data-test
    }
    openExecutionDetails(executionName) {
        cy.xpath(`//tr[.//span[text()="${executionName}"]]`).click()
    }

    validateLogOutputContents(expectedText, customTimeout=null) {
        cy.log(`validateLogOutputContents: ${expectedText}`)
        const logOutpusContainerSelector = 'code span' //TODO: data-test

        if (customTimeout) {
            cy.contains(logOutpusContainerSelector, expectedText, { timeout: customTimeout })
        } else {
            cy.contains(logOutpusContainerSelector, expectedText)
        }
    }
}
export default TestExecutionsPage