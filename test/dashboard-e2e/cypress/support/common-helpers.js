class CommonHelpers {
    validateTest(testData, createdTestData) {
        cy.expect(testData.name).to.equal(createdTestData.name)
        //TODO: label
        cy.expect(testData.type).to.equal(createdTestData.type)
        cy.expect(testData.testSource.type).to.equal(createdTestData.content.type)


        //testSources
        if (testData.testSource.type == "git-file" || testData.testSource.type == "gir-dir") {
            for (let key in testData.testSource) {
                cy.expect(testData.testSource[key]).to.equal(createdTestData.content.repository[key])
            }
        }
        
    }
}
export default CommonHelpers