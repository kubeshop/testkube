class MainPage {
    visitMainPage(manualInitialDalogHandling) {
        if(manualInitialDalogHandling) {

            cy.visit(`/`)
            this.handleApiEndpointDialog(Cypress.env('API_URL'))

            this.handleCookiesDialog()
        } else {
            cy.visit('/', {
                onBeforeLoad: function (window) {
                    window.localStorage.setItem('isGADisabled', '1');
                    window.localStorage.setItem('apiEndpoint', Cypress.env('API_URL'))
                }
            })
        }
    }

    handleApiEndpointDialog(customUri) {
        //TODO: check if displayed

        if (customUri === undefined) {
            cy.get('span[data-test="endpoint-modal-close-button"]').click()
        } else {
            cy.get('input[data-test="endpoint-modal-input"]').type(customUri)
            cy.get('button[data-test="endpoint-modal-get-button"]').click()
        }
    }

    handleCookiesDialog() {
        //TODO: check if displayed
        cy.get('button[data-test="cookies-banner-accept-button"]').click()
    }

    openCreateTestDialog() {
        cy.get('button[data-test="add-a-new-test-btn"]').click()
    }

    openTestExecutionDetails(realTestName) {
        cy.xpath(`//div[@data-test="tests-list-item" and .//span[text()="${realTestName}"]]`).click()
    }
}
export default MainPage
