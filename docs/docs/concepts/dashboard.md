# Testkube Dashboard

The Testkube Dashboard provides a simple web-based user interface for monitoring Testkube test results via a web browser.

![img.png](../img/dashboard-1.6.png)

The URL to access the Testkube Dashboard is [https://demo.testkube.io](https://demo.testkube.io), which, when first loaded, will prompt for the results endpoint of your Testkube installation. Click the **Settings** icon at the bottom left of the screen to return to change the Testkube API endpoint.

![dashboard-endpoint-prompt.png](../img/dashboard-endpoint-prompt-1.6.png)

See the [UI Results Endpoint section](#ui-results-endpoint) below to find the endpoint to open the dashboard. Once you have the results endpoint, you can append it to the above URL (as an apiEndpoint parameter) for a direct link to the dashboard with your results:

`https://demo.testkube.io/?apiEndpoint=...`

Alternatively, the Testkube Dashboard can be opened on your local machine using command `sh kubectl testkube dashboard` which uses port forwarding for accessing your local results endpoint (see more [here](../reference/cli/testkube_dashboard.md)).

## Explore the Testkube Dashboard

The Testkube Dashboard displays the current status of Tests and Test Suites executed in your environment.

![Test List](../img/test-list-1.6.png)

![Test Suites List](../img/test-suite-list-1.6.png)

After selecting Tests or Test Suites in the left bar, the list of recent runs is displayed. At the top of the list, a Search field and filters for Labels and Status make finding tests in a large list easier:

![Search & Filter](../img/search-filter-1.6.png)

Select any Test or Test Suite to see the recent executions and their statuses.

![Execution Status](../img/execution-status-1.6.png)

The execution statistics of the chosen Test or Test Suite are at the top of the screen, along with a graph of success or failure for the executions.

The **Recent executions** tab has the list of executions. A green checkmark denotes a successful execution, a red 'x' denotes a failed execution and circling dots denotes a current run of a Test or Test Suite.

![Recent executions](../img/recent-executions-1.6.png)

The **CLI Commands** tab shows the commands used to perform the selected test:

![CLI Commands](../img/CLI-commands-1.6.png)

The **Settings** tab contains 3 types of information about the Test or Test Suite.

![Setting](../img/settings-1.6.png)

### General Settings

Clicking the **General** box under the **Settings** tab displays the **Test name & description** and **Labels** for the Test or Test Suite:

![Settings General](../img/settings-general-1.6.png)

It is also the place to delete a Test or Test Suite:

![Settings General Delete](../img/settings-general-delete-1.6.png)

### Variables & Secrets

![Variable Tab](../img/variable-tab-1.6.png)

Visit [Using Tests Variables](./tests/tests-variables.md) for a description of adding Variables and Secrets.

### Definition

Clicking the **Definition** box under the **Settings** tab allows the validation and export the configuration for the Test or Test Suite:

![Settings Definition](../img/settings-definition-1.6.png)
