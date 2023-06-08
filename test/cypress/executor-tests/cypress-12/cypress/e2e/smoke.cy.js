describe('Testkube website', () => {
  it('Open Testkube website', () => {
    cy.visit('/')
  })
  it(`Validate CYPRESS_CUSTOM_ENV ENV (${Cypress.env('CUSTOM_ENV')})`, () => {
    expect('CYPRESS_CUSTOM_ENV_value').to.equal(Cypress.env('CUSTOM_ENV')) //CYPRESS_CUSTOM_ENV - "cypress" prefix - auto-loaded from global ENVs
  })
  it(`Validate NON_CYPRESS_ENV ENV (${Cypress.env('NON_CYPRESS_ENV')})`, () => {
    expect('NON_CYPRESS_ENV_value').to.equal(Cypress.env('NON_CYPRESS_ENV')) //NON_CYPRESS_ENV - need to be loaded with --env parameter
  })
})
