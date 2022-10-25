class CommonHelpers {
    validateTest(testData, createdTestData) {
        cy.expect(testData.name).to.equal(createdTestData.name)
        //TODO: label
        cy.expect(testData.type).to.equal(createdTestData.type)
        cy.expect(testData.content.type).to.equal(createdTestData.content.type)

        //testSources
        const contentType = testData.content.type
        if (contentType == "git-file" || contentType == "gir-dir") {
            for (let key in testData.content.repository) {
                cy.expect(testData.content.repository[key]).to.equal(createdTestData.content.repository[key])
            }
        }
    }
}
export default CommonHelpers