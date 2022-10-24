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

// {
//     "k6-git-file": {
//         "name": "internal-dashboard-e2e-k6-git-file",
//         "label": "TODO",
//         "type": "k6/script",
//         "testSource": {
//             "type": "git-file",
//             "uri": "https://github.com/kubeshop/testkube.git",
//             "branch": "cypress-e2e",
//             "path": "test/k6/executor-tests/k6-smoke-test-without-envs.js"
//         }
//     }
// }


// {
//     "name": "internal-dashboard-e2e-k6-git-file",
//     "namespace": "testkube",
//     "type": "k6/script",
//     "content": {
//         "type": "git-file",
//         "repository": {
//             "type": "git-file",
//             "uri": "https://github.com/kubeshop/testkube.git",
//             "branch": "cypress-e2e",
//             "path": "test/k6/executor-tests/k6-smoke-test-without-envs.js"
//         }
//     },
//     "created": "2022-10-24T11:16:18Z"
// }