# Configure Identity Providers

You can configure Testkube Pro On-Prem to authenticate users using different identity providers such.

For a list of all supported identity providers and example Dex configurations, see [Connectors](https://dexidp.io/docs/connectors/).

### Static Users

For a quickstart, or if you do not have an identity provider, you can configure Testkube Pro On-Prem to use static users.

```yaml
additionalConfig: |
    enablePasswordDB: true
    staticPasswords:
      - email: <user email>
        hash: <bcrypt hash of user password>
        username: <username>
```

Replace `<user email>`, `<bcrypt hash of user password>`, and `<username>` with your actual values.

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
