describe('Testkube website', () => { //TODO: disabled for now - reenable after: https://github.com/kubeshop/testkube/issues/2540
  it('Open Testkube website', () => {
    cy.visit('/')
  })
  it.skip(`Validate CYPRESS_CUSTOM_ENV ENV (${Cypress.env('CUSTOM_ENV')})`, () => {
    expect('CYPRESS_CUSTOM_ENV_value').to.equal(Cypress.env('CUSTOM_ENV')) //CYPRESS_CUSTOM_ENV - "cypress" prefix - auto-loaded from global ENVs
  })
  it.skip(`Validate NON_CYPRESS_ENV ENV (${Cypress.env('NON_CYPRESS_ENV')})`, () => {
    expect('NON_CYPRESS_ENV_value').to.equal(Cypress.env('NON_CYPRESS_ENV')) //NON_CYPRESS_ENV - need to be loaded with --env parameter
  })
})
