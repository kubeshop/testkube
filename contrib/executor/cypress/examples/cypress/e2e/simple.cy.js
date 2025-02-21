describe("The Home Page", () => {
  it("successfully loads", () => {
    cy.visit("https://testkube-test-page-lipsum.pages.dev/");

    cy.contains("Testkube");
  });
});
