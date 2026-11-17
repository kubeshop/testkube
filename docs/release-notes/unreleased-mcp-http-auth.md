# Unreleased: standalone MCP HTTP authentication

Standalone Streamable HTTP now requires a separate `TK_MCP_SHTTP_AUTH_TOKEN`.
Every client request must carry `Authorization: Bearer <inbound-token>`; the
outbound Testkube API token is not accepted as the inbound secret. Missing or invalid
configuration fails startup with a nonzero exit, including without verbose mode.

**Breaking change for standalone HTTP users:** existing configurations without the
new token will no longer start, and clients without its bearer header will receive
HTTP 401. The default bind address is now `localhost`, including Docker; deployments
that relied on Docker's external bind default must set an explicit host. Stdio and
the hosted Control Plane MCP endpoint require no migration.

The operator deploying standalone HTTP MCP owns token creation, secure storage, and
distribution to clients. The server does not provision it; reuse the stored token on
restarts. Standard agent/runner and Enterprise Helm charts need no changes because
they do not deploy this standalone server. For custom Kubernetes deployments, see
the [Secret injection example](../../pkg/mcp/README.md#streamable-http-shttp-transport).

Migrate an existing standalone HTTP deployment in this order:

1. Schedule a restart and client reconnection. Stop or isolate the existing HTTP
   endpoint while preparing the upgrade so it cannot continue accepting unauthenticated
   requests. Existing streams will end when the process stops.
2. Generate and securely store a separate inbound token with at least 32 bytes of
   entropy. Configure `TK_MCP_SHTTP_AUTH_TOKEN` in the server's deployment environment;
   keep the outbound Testkube credential separate and reuse the stored inbound token
   on subsequent restarts.
3. Configure every MCP client to send `Authorization: Bearer <inbound-token>` on
   every request using that stored value. Arrange this before restoring client traffic;
   existing session IDs do not bypass authentication.
4. Update the server launch arguments: keep `--transport=shttp`, and explicitly set
   `--shttp-host=0.0.0.0` if external binding is needed (including container access).
   CLI transport/host/port are selected with flags, not environment variables.
   For remote clients, configure HTTPS using `--shttp-tls`, `--shttp-cert-file`, and
   `--shttp-key-file`, or a TLS proxy with a private upstream listener. Prevent direct
   access around the proxy; a loopback proxy must set upstream `Host` to `localhost`.
5. Deploy the updated binary or image and restart with the prepared configuration.
   Missing or invalid authentication configuration, listener failures, and invalid TLS
   files must be corrected before restoring traffic; startup errors now exit nonzero.
6. Verify that an unauthenticated request to `/mcp` receives HTTP 401. With a configured
   MCP client, initialize a new session, list tools, and run a workflow intended for
   testing. Confirm it executes under the configured outbound Testkube identity before
   restoring normal traffic.

All clients share the server's outbound permissions. Rotate the inbound secret by
restarting the server; existing streams close. Stdio remains the default and requires
no new secret; the hosted Control Plane MCP endpoint is unchanged. If migration fails,
disable standalone HTTP or use stdio instead of restoring unauthenticated access.

Release version and advisory fixed-version metadata remain pending release verification.
