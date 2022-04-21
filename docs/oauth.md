# Testkube UI authentication

Testkube doesn't provide a separate user/role management system to protect access to its UI.
Users can configure OAuth based authenication module using Testkube Helm chart parameters.
In order to provide it Testkube can automatically create OAuth2-Proxy service and deployment integrated 
with GitHub, as well as properly configure Kubernetes Nginx Ingress Controller and create required 
ingresses.

# Provide parameters for UI ans API ingresses

## API Ingress

Pass values to helm chart (they are empty by default)

```sh
--set testkube-api.ingress.annotations."nginx\.ingress\.kubernetes\.io/auth-url"="http://\$host/oauth2/auth" (pay attention to http or https)
--set testkube-api.ingress.annotations."nginx\.ingress\.kubernetes\.io/auth-signin"="http://\$host/oauth2/start?rd=\$escaped_request_uri" (pay attention to http or https)
--set testkube-api.ingress.annotations."nginx\.ingress\.kubernetes\.io/access-control-allow-origin" = "*"
```

## UI Ingress

Pass values to helm chart (they are empty by default)

```sh
--set testkube-dashboard.ingress.annotations.nginx."ingress\.kubernetes\.io/auth-url"="http://\$host/oauth2/auth" (pay attention to http or https)
--set testkube-dashboard.ingress.annotations.nginx."ingress\.kubernetes\.io/auth-signin"="http://\$host/oauth2/start?rd=\$escaped_request_uri" (pay attention to http or https)
```

# Create Github application

Homepage URL
should be UI home page http://testdash.testkube.io (pay attention to http or https)

Authorization callback URL
should be prebuilt page at UI website http://testdash.testkube.io/oauth2/callback (pay attention to http or https)

Remember generated Client ID and Client Secret

# OAuth service, deployment and ingresses parameters

Pass values to helm chart (they are empty by default)

```sh
--set testkube-dashboard.oauth2.enabled=false
--set testkube-dashboard.oauth2.env.clientId=<client id from Github app>
--set testkube-dashboard.oauth2.env.clientSecret=<client secret from Github app>
--set testkube-dashboard.oauth2.env.githubOrg=<your organization if you need it to provide access only to members of your organization>
--set testkube-dashboard.oauth2.env.cookieSecret=<16 or 32 byte encoded base 64>
--set testkube-dashboard.oauth2.env.cookieSecure="false" (false - for http connection, true - for https connections, can be skipped for https)
--set testkube-dashboard.oauth2.env.redirectUrl="http://demo.testkube.io/oauth2/callback" (pay attention to http or https, can be skipped for https)
```
