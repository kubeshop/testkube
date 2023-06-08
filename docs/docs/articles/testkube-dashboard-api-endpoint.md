# API Endpoint

The API Endpoint connects the dashboard to your Testkube API endpoint that is part of your Testkube installation. 

In case you're starting the dashboard from the CLI using the `testkube dashboard` command it should be automatically set for you. 

But if you are [exposing Testkube Dashboard over an Ingress Controller](./exposing-testkube-with-ingress-nginx.md) you will have to also expose the API Endpoint and connect the Testkube Dashboard through the Testkube API Endpoint modal which you can find in *Settings* > *General* > *Testkube API endpoint*: 

![dashboard-endpoint-prompt.png](../img/dashboard-endpoint-prompt-1.6.png)

You can also append it to the above URL (as an apiEndpoint parameter) for a direct link to the dashboard with your results:

`https://demo.testkube.io/?apiEndpoint=...`