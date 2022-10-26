class MainPage {
    visitMainPage(manualInitialDalogHandling) {
        if(manualInitialDalogHandling) {
<<<<<<< HEAD
            cy.visit(`/`)
            this.handleApiEndpointDialog(Cypress.env('API_URL'))
=======
            cy.visit(`/apiEndpoint?apiEndpoint=${Cypress.env('API_URL')}`) //TODO: move to variables
            this.handleApiEndpointDialog()
>>>>>>> origin/main
            this.handleCookiesDialog()
        } else {
            cy.visit('/', {
                onBeforeLoad: function (window) {
                    window.localStorage.setItem('isGADisabled', '1');
<<<<<<< HEAD
                    window.localStorage.setItem('apiEndpoint', Cypress.env('API_URL'))
=======
                    window.localStorage.setItem('apiEndpoint', Cypress.env('API_URL')) //TODO: move to variables
>>>>>>> origin/main
                }
            })
        }
    }

    handleApiEndpointDialog(customUri) {
        //TODO: check if displayed

        if (customUri === undefined) {
<<<<<<< HEAD
            cy.get('span[data-test="endpoint-modal-close-button"]').click()
        } else {
            cy.get('input[data-test="endpoint-modal-input"]').type(customUri)
            cy.get('button[data-test="endpoint-modal-get-button"]').click()
=======
            cy.get('div[role="dialog"] button[class="ant-modal-close"]').click() //TODO: data-test attribute needed - replace when it will be available
>>>>>>> origin/main
        }
    }

    handleCookiesDialog() {
        //TODO: check if displayed

<<<<<<< HEAD
        cy.get('button[data-test="cookies-banner-accept-button"]').click()
    }

    openCreateTestDialog() {
        cy.get('button[data-test="add-a-new-test-btn"]').click()
    }

    openTestExecutionDetails(realTestName) {
        cy.xpath(`//div[@data-test="tests-list-item" and .//span[text()="${realTestName}"]]`).click()
=======
        cy.get('div[class*="ant-space-vertical"] div[class="ant-space-item"] div[class*="ant-space-horizontal"] button').first().click() //TODO: data-test attribute needed - replace when it will be available
    }

    openCreateTestDialog() {
        cy.get('button span').first().click() //TODO: data-test attribute needed - replace when it will be available
>>>>>>> origin/main
    }
}
export default MainPage
