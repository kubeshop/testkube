---
sidebar_position: 2
sidebar_label: OAuth for CLI
---
# Testkube CLI Authentication

Testkube doesn't provide a separate user/role management system to protect access to its CLI.
Users can configure OAuth-based authentication modules using Testkube Helm chart parameters and
the CLI config command.
Testkube can automatically configure the Kubernetes Nginx Ingress Controller and create required 
ingresses.

## **Provide Parameters for API Ingress**
Pass values to Testkube Helm chart during installation or upgrade (they are empty by default).
Pay attention to the usage of the scheme (http or https) in URIs.

```bash
--set testkube-api.cliIngress.enabled=true \
--set testkube-api.cliIngress.oauth.provider="github"
--set testkube-api.cliIngress.oauth.clientID="XXXXXXXXXX" \
--set testkube-api.cliIngress.oauth.clientSecret="XXXXXXXXXX" \
--set testkube-api.cliIngress.oauth.scopes=""
```
## **Create Github OAuth Application**

Register a new Github OAuth application for your personal or organization account.

![Register new App](../img/github_app_request_cli.png)

Pay attention to the usage of the scheme (http or https) in URIs.
The homepage URL
should be the UI home page http://127.0.0.1:13254.

The authorization callback URL
should be a prebuilt page at the UI website http://127.0.0.1:13254/oauth/callback.

![View created App](../img/github_app_response_cli.png)

Remember the generated Client ID and Client Secret.

## **Provide Parameters for CLI**

Run the command below to configure oauth parameters (we support github OAuth provider):

```bash
kubectl testkube config oauth https://demo.testkube.io/api --client-id XXXXXXXXXX --client-secret XXXXXXXXXX
```

Output:

```bash
You will be redirected to your browser for authentication or you can open the url below manually
https://github.com/login/oauth/authorize?access_type=offline&client_id=XXXXXXXXXX&redirect_uri=http%3A%2F%2F127.0.0.1%3A13254%2Foauth%2Fcallback&response_type=code&state=iRQkcwXV
Authentication will be cancelled in 60 seconds
```

Authorization for the Github application will be requested and access will need to be confirmed. 
![Confirm App authorization](../img/github_app_authorize_cli.png)

If authorization is successful, you will see the success page.
![Success Page](../img/github_app_success_cli.png)

Output:

```bash
Shutting down server...
Server gracefully stopped 🥇
New api uri set to https://demo.testkube.io/api 🥇
New oauth token gho_XXXXXXXXXX 🥇
```

## **Run CLI Commands with OAuth**

Now all of your requests with direct client will submit an OAuth token, for example:

```bash
kubectl testkube get executors -c direct
```

Output:

```bash
  NAME               | URI | LABELS  
+--------------------+-----+--------+
  artillery-executor |     |         
  curl-executor      |     |         
  cypress-executor   |     |         
  k6-executor        |     |         
  postman-executor   |     |         
  soapui-executor    |     |      
```

## **Environment Variables**

You can use 2 environment variables to override CLI config values:

TESTKUBE_API_URI - for API uri.

TESTKUBE_OAUTH_ACCESS_TOKEN - for OAuth access token.
