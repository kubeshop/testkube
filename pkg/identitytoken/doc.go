// Package identitytoken mints the execution identity tokens behind the
// identity_token(audience) expression (pkg/credentials) and publishes what a
// cloud provider needs to verify them. Both control planes use it: the
// standalone one in pkg/controlplane, and testkube-cloud-api, which imports
// this package rather than reimplementing it, so the two issue the same tokens.
//
// A workflow step calls identity_token("sts.amazonaws.com"); testworkflow-init
// sends GetIdentityToken(execution_id, audience) to the control plane, which
// answers with a token signed here. The workflow then exchanges it with the
// provider (AWS STS AssumeRoleWithWebIdentity, GCP Workload Identity
// Federation, Azure federated credentials, Vault JWT auth) for short-lived
// credentials, so it stores none.
//
// # Token
//
// An RS256 JWT, valid for TokenTTL, with the key ID of the signing key (its
// RFC 7638 thumbprint) in the "kid" header and these claims:
//
//   - iss: the issuer URL.
//   - sub: "organization:<org ID>:environment:<env ID>:workflow:<workflow name>".
//     Trust policies match on it, so it holds only what stays the same across
//     executions; the execution is in its own claim. A standalone installation
//     without TESTKUBE_PRO_ORG_ID/TESTKUBE_PRO_ENV_ID leaves those IDs empty,
//     and its issuer is what tells it apart.
//   - aud: the audience the workflow asked for, exactly as given.
//   - iat, nbf, exp, jti: issued now, expiring after TokenTTL, with a random ID.
//   - organization_id, environment_id, workflow, execution_id, runner_id: the
//     same identity as separate claims, for providers that map custom claims.
//
// A token is minted only for an execution that exists and has not finished.
//
// # Discovery
//
// The issuer is one URL for the whole installation, and the control plane's
// HTTP API serves, without authentication:
//
//   - /.well-known/openid-configuration (DiscoveryPath): the OIDC discovery document.
//   - /.well-known/jwks.json (JWKSPath): the public signing key as a JWK set.
//
// The issuer must therefore be the public URL of that API, reachable by the
// cloud provider; HandleDiscovery and HandleJWKS serve the two documents.
//
// # Execution tokens
//
// The standalone control plane has no other way to tell which execution is
// calling, so it hands each execution ExecutionToken(env, execution) as its
// execution token, and GetIdentityToken requires it back as the caller's
// api-key. The token is an HMAC keyed from the signing key, so it needs no
// storage and is the same on every replica. testkube-cloud-api authenticates
// callers itself and does not use it.
//
// # Keys
//
// A PEM RSA private key (PKCS#1 or PKCS#8) of at least 2048 bits. The JWK set
// holds only the current key, so rotating it invalidates tokens already
// issued, which TokenTTL keeps short-lived, and the execution tokens of
// running executions.
package identitytoken
