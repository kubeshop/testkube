---
sidebar_position: 1
sidebar_label: Testkube UI
---
# The Testkube UI

The Testkube UI provides a simple web-based user interface for monitoring Testkube test results via a web browser.

![img.png](../img/dashboard-1.6.png)

The URL to access the Testkube UI is [https://demo.testkube.io](https://demo.testkube.io), which, when first loaded, will prompt for the results endpoint of your Testkube installation. Click the **Settings** icon at the bottom left of the screen to return to change the Testkube API endpoint.

![dashboard-endpoint-prompt.png](../img/dashboard-endpoint-prompt-1.6.png)

See the [UI Results Endpoint section](#ui-results-endpoint)  below to find the endpoint to open the dashboard. Once you have the results endpoint, you can append it to the above URL (as an apiEndpoint parameter) for a direct link to the dashboard with your results:

`https://demo.testkube.io/?apiEndpoint=...`

Alternatively, the UI can be opened on your local machine using command ```sh kubectl testkube dashboard``` which uses port forwarding for accessing your local results endpoint (see more [here](../5-cli-reference/testkube_dashboard.md)).

## **Explore the UI**

The Testkube UI displays the current status of Tests and Test Suites executed in your environment.

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

### **General Settings**

Clicking the **General** box under the **Settings** tab displays the **Test name & description** and **Labels** for the Test or Test Suite:

![Settings General](../img/settings-general-1.6.png)

It is also the place to delete a Test or Test Suite:

![Settings General Delete](../img/settings-general-delete-1.6.png)

### **Variables & Secrets**

![Variable Tab](../img/variable-tab-1.6.png)

Visit [Using Tests Variables](tests/tests-variables.md) for a description of adding Variables and Secrets.

### **Definition**

Clicking the **Definition** box under the **Settings** tab allows the validation and export the configuration for the Test or Test Suite:

![Settings Definition](../img/settings-definition-1.6.png)


## **UI Results Endpoint**

To expose the results endpoint to open the UI there are a few options:

* Expose the results endpoint using an Ingress controller and use it in the UI at [https://demo.testkube.io](https://demo.testkube.io).
* Install the dashboard together with Testkube.

* If you are using minikube use [http://localhost:8088/v1](http://localhost:8088/v1) as a TestKube API endpoint.

This is achieved by installing Testkube using the Helm Charts located at [https://github.com/kubeshop/helm-charts](https://github.com/kubeshop/helm-charts).

## **Prerequisites**

Add the repo to Helm:

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts && helm repo update
```

An Ingress controller is needed to expose externally. By default Testkube is using `ingress-nginx`. Any other Ingress controller can be used but will require advanced configuration. A values file for guidance can be found [here](https://github.com/kubeshop/helm-charts/blob/39f73098630b333ba66db137e7fc016c39d92876/testkube/charts/testkube/values-demo.yaml).

## **Configure Ingress for Results Endpoint**

```sh
helm install testkube kubeshop/testkube --set testkube-api.ingress.enabled="true"
```

By default, the results are using the path ```/results```, so the results will be accessible at ```ingress_host/results/```

The Ingress configuration used is available in the [Testkube Helm Repo](https://github.com/kubeshop/helm-charts).

## **Installing the Testkube UI**

Ingress can be installed for the UI together with api-server Ingress by using a Helm chart:

```sh
helm install testkube kubeshop/testkube --set testkube-dashboard.enabled="true" --set testkube-dashboard.ingress.enabled="true" --set testkube-api.ingress.enabled="true"
```

> The Testkube UI talks to an api-server via the endpoint. Hence, the api-server will need to be exposed as well.

To get the address of Ingress use:

```sh
kubectl get ing
```

## **HTTPS/TLS Configuration**

To have secure access to the UI and the results endpoint, a certificate should be provided. The Helm charts can be configured from the Ingress section of the values file:

```yaml
ingress:
    enabled: "true"
    annotations: 
      kubernetes.io/ingress.class: nginx
      nginx.ingress.kubernetes.io/force-ssl-redirect: "false"
      nginx.ingress.kubernetes.io/ssl-redirect: "false"
      nginx.ingress.kubernetes.io/enable-cors: "true"
      nginx.ingress.kubernetes.io/cors-allow-methods: "GET"
      nginx.ingress.kubernetes.io/cors-allow-credentials: "false"
      # add an annotation indicating the issuer to use.
      cert-manager.io/cluster-issuer: letsencrypt-prod
      # controls whether the ingress is modified ‘in-place’,
      # or a new one is created specifically for the HTTP01 challenge.
      acme.cert-manager.io/http01-edit-in-place: "true"
    path: /
    hosts:
      - demo.testkube.io
    tlsenabled: "true"
    tls: # < placing a host in the TLS config will indicate a certificate should be created
    - hosts:
      - demo.testkube.io
      secretName: testkube-demo-cert-secret
```
Certificates are automatically generated using encrypt and cert-manager, but can be configured for any particular case. A full values file example can be found [here](https://github.com/kubeshop/helm-charts/blob/39f73098630b333ba66db137e7fc016c39d92876/testkube/charts/testkube/values-demo.yaml).

If there is no need for TLS (Transport Layer Security) to be enabled, omit the TLS configuration.

> We highly discourage working in a non-safe environment which is exposed without the use of a TLS-based connection. Please do so in a private internal environment for testing or development purposes only.

To pass specific values to the Ingress annotations, the Helm "--set" option can be used: 

```sh
helm install testkube kubeshop/testkube --set testkube-dashboard.enabled="true" --set testkube-dashboard.ingress.enabled="true" --set testkube-api.ingress.enabled="true" --set testkube-api.ingress.annotations.kubernetes\\.io/ingress\\.class="anything_needed" 
```

A better approach is to configure and call a values file with the Ingress custom values:

```sh
helm install testkube kubeshop/testkube --values https://github.com/kubeshop/helm-charts/blob/39f73098630b333ba66db137e7fc016c39d92876/testkube/charts/testkube/values-demo.yaml
```
