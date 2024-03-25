# Configure Identity Providers

You can configure Testkube Pro On-Prem to authenticate users using different identity providers such as Azure AD, Google, Okta, and OIDC. To do this, you need to update the `additionalConfig` field in the Helm chart values with the appropriate configuration for each identity provider.

For a list of all supported identity providers, see [Connectors](https://dexidp.io/docs/connectors/).

The examples below show how to configure Testkube Pro On-Prem with each identity provider by editing the `dex.configTemplate.additionalConfig` field in the Helm chart values.

### Quickstart

For a quickstart, or if you do not have an identity provider, you can configure Testkube Pro On-Prem to use static users.
See [Static Users](#static-users).

### Static Users

To configure Testkube Pro On-Prem with static users, add the following configuration to the `additionalConfig` field:

```yaml
additionalConfig: |
  enablePasswordDB: true
  staticPasswords:
    - email: <user email>
      hash: <bcrypt hash of user password>
      username: <username>
```

Replace `<user email>`, `<bcrypt hash of user password>`, and `<username>` with your actual values.

### Azure AD

To configure Testkube Pro On-Prem with Azure AD, add the following configuration to the `additionalConfig` field:

```yaml
additionalConfig: |
  connectors:
    - type: azuread
      id: azuread
      name: Azure AD
      config:
        clientID: <Azure AD client ID>
        clientSecret: <Azure AD client secret>
        redirectURI: <Testkube Pro On-Prem redirect URI>
```

Replace `Azure AD client ID`, `Azure AD client secret`, and `Testkube Pro On-Prem redirect URI` with your actual Azure AD configuration values.

### Google

To configure Testkube Pro On-Prem with Google, add the following configuration to the 'additionalConfig' field:

```yaml
additionalConfig: |
  connectors:
    - type: google
      id: google
      name: Google
      config:
        clientID: <Google client ID>
        clientSecret: <Google client secret>
        redirectURI: <Testkube Pro On-Prem redirect URI>
```

Replace `Google client ID`, `Google client secret`, and `Testkube Pro On-Prem redirect URI` with your actual Google configuration values.

### Okta

To configure Testkube Pro On-Prem with Okta, add the following configuration to the `additionalConfig` field:

```yaml
additionalConfig: |
  connectors:
    - type: okta
      id: okta
      name: Okta
      config:
        issuerURL: <Okta issuer URL>
        clientID: <Okta client ID>
        clientSecret: <Okta client secret>
        redirectURI: <Testkube Pro On-Prem redirect URI>
```

Replace `Okta issuer URL`, `Okta client ID`, `Okta client secret`, and `Testkube Pro On-Prem redirect URI` with your actual Okta configuration values.

### OIDC

To configure Testkube Pro On-Prem with an OIDC provider, add the following configuration to the `additionalConfig` field:

```yaml
additionalConfig: |
  connectors:
    - type: oidc
      id: oidc
      name: OIDC
      config:
        issuerURL: <OIDC issuer URL>
        clientID: <OIDC client ID>
        clientSecret: <OIDC client secret>
        redirectURI: <Testkube Pro On-Prem redirect URI>
```
