describe('Log in', () => {
  it(`Log in with valid credentials`, () => {
    const userData = {
      username: "AdminUser",
      password: "SomeVeryLongPassword123456",
    }

    cy.visit("https://testkube-test-page-login.pages.dev/")

    cy.get('[data-testid="username"]').type(userData.username, { force: true })
    cy.get('[data-testid="password"]').type(userData.password, { force: true })
    cy.get('[data-testid="login-button"]').click()
    cy.url().should('contain', '/demo')
    cy.get('[data-testid="lorem-ipsum"]').should("contain.text", "Lorem ipsum");
    cy.get('[data-testid="login-message"]').should('not.be.visible')
  })
})
