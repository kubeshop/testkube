class MainPage {
    visitMainPage(manualInitialDalogHandling) {
        if(manualInitialDalogHandling) {
            cy.visit(`/apiEndpoint?apiEndpoint=${Cypress.env('API_URL')}`)
            this.handleApiEndpointDialog()
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
            cy.get('div[role="dialog"] button[class="ant-modal-close"]').click() //TODO: data-test attribute needed - replace when it will be available
        }
    }

    handleCookiesDialog() {
        //TODO: check if displayed

        cy.get('div[class*="ant-space-vertical"] div[class="ant-space-item"] div[class*="ant-space-horizontal"] button').first().click() //TODO: data-test attribute needed - replace when it will be available
    }

    openCreateTestDialog() {
        cy.get('button span').first().click() //TODO: data-test attribute needed - replace when it will be available
    }
}
export default MainPage
