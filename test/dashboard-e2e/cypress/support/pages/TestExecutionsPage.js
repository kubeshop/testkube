class TestExecutionsPage {
    runTest() {
        cy.get('div[class="ant-page-header-heading"] button').click() //TODO: replace with data-test
    }
    openExecutionDetails(executionName) {
        cy.xpath(`//tr[.//span[text()="${executionName}"]]`).click()
    }
}
export default TestExecutionsPage