// Package commons holds helpers shared by the API server entry points.
//
// Self-registration secret: after a runner registers itself with the Control Plane,
// PersistSelfRegistrationSecret stores the returned credentials in a Kubernetes secret that the
// next start reads through envFrom instead of registering again.
//   - It creates the secret and, on AlreadyExists, updates it with the new credentials.
//   - Transient failures are retried with a bounded backoff (DefaultSelfRegistrationSecretBackoff).
//     Each Create or Update attempt has its own timeout; a timed-out attempt is retried while the
//     parent context is alive, and the whole operation stops once that context is done.
//   - Errors a retry cannot fix are returned at once: forbidden, unauthorized, invalid, secret
//     management disabled, and an existing secret that is not managed by Testkube (it lacks the
//     label createdBy=testkube, so the operator must delete it or add the label).
//   - The caller exits on failure. The runner is already registered, so continuing on in-memory
//     credentials would only fail later with AlreadyExists on the next start.
package commons
