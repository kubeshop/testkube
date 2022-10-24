class MainPage {
    visitMainPage() {
        cy.visit('http://localhost:8080/apiEndpoint?apiEndpoint=localhost:8088/v1')
        this.handleApiEndpointDialog()
        this.handleCookiesDialog()
    }

    handleApiEndpointDialog(customUri) {
        //TODO: check if displayed

        if (customUri === undefined) {
            cy.get('div[role="dialog"] button[class="ant-modal-close"]').click() //TODO: data-test attribute needed
        }
    }

    handleCookiesDialog() {
        //TODO: temporary click. Where the cookies consent is stored? Nothing in cookies or localStorage
        //TODO: check if displayed

        cy.get('div[class*="ant-space-vertical"] div[class="ant-space-item"] div[class*="ant-space-horizontal"] button').first().click()
    }

    openCreateTestDialog() {
        cy.get('button span').first().click() //TODO: data-test
    }
}
export default MainPage
