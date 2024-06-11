# Configure Identity Providers

You can configure Testkube Pro On-Prem to authenticate users using different
identity providers such.

For a list of all supported identity providers and example Dex configurations,
see [Connectors](https://dexidp.io/docs/connectors/).

### Static Users

For a quickstart or if you do not have an identity provider, you can configure
Testkube to use static users.

```yaml
dex:
    configTemplate:
        additionalConfig: |
            enablePasswordDB: true
            staticPasswords:
              - email: <user email>
                hash: <bcrypt hash of user password>
                username: <username>
```

Replace `<user email>`, `<bcrypt hash of user password>`, and `<username>` with
the actual values for your user(s).

### OIDC

Examples of OIDC providers: Okta, Google, Salesforce, and Azure AD v2.

To configure using an OIDC provider, set the appropriate values in the
`testkube-enterprise` chart. Note, you will need to create a secret containing
credentials for the identity provider and replace the
`<oidc-credentials-secret-name>`, `<client-id-key`, and `<client-secret-key>`
placeholders with the right values.

```yaml
dex:
    envVars:
        - name: GOOGLE_CLIENT_ID
          valueFrom:
              secretKeyRef:
                  name: <oidc-credentials-secret-name>
                  key: <client-id-key>
        - name: GOOGLE_CLIENT_SECRET
          valueFrom:
              secretKeyRef:
                  name: <oidc-credentials-secret-name>
                  key: <client-secret-key>
    configTemplate:
        additionalConfig: |
            connectors:
            - type: oidc
              id: google
              name: Google
              config:
                # Canonical URL of the provider, also used for configuration discovery.
                # This value MUST match the value returned in the provider config discovery.
                #
                # See: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig
                issuer: https://accounts.google.com

                # Connector config values starting with a "$" will read from the environment.
                clientID: $GOOGLE_CLIENT_ID
                clientSecret: $GOOGLE_CLIENT_SECRET

                # Dex's issuer URL + "/callback"
                redirectURI: <dex endpoint>/callback
```
