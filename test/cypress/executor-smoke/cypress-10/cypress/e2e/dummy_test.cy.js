describe('Testkube website', () => {
  it('Open Testkube website', () => {
    cy.visit('/')
  })
  it(`Print CYPRESS_CUSTOM_ENV ENV: ${Cypress.env('CUSTOM_ENV')}`, () => {
  })
  it(`Print NON_CYPRESS_ENV ENV: ${Cypress.env('NON_CYPRESS_ENV')}`, () => {
  })
  it('Validate ENVs', () => {
    expect('CYPRESS_CUSTOM_ENV_value').to.equal(Cypress.env('CUSTOM_ENV')) //CYPRESS_CUSTOM_ENV - "cypress" prefix - auto-loaded from global ENVs
    expect('NON_CYPRESS_ENV_value').to.equal(Cypress.env('NON_CYPRESS_ENV')) //NON_CYPRESS_ENV - need to be loaded with --env parameter
  })
})
