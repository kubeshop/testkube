# Unreleased: standalone MCP HTTP authentication

Standalone Streamable HTTP now requires a separate `TK_MCP_SHTTP_AUTH_TOKEN`.
Every client request must carry `Authorization: Bearer <inbound-token>`; the
outbound Testkube API token is not accepted as the inbound secret. Missing or invalid
configuration fails startup with a nonzero exit, including without verbose mode.

**Migration:** provision a random inbound secret with at least 32 bytes of entropy,
configure it on server and clients, and explicitly set `--shttp-host=0.0.0.0` if external
binding is needed. The default is now loopback even in Docker. Remote connections
must use HTTPS via the repaired built-in TLS flags or a TLS proxy with a private
upstream listener. CLI transport/host/port are selected with flags, not environment variables.

All clients share the server's outbound permissions. Rotate the inbound secret by
restarting the server; existing streams close. Stdio remains the default and requires
no new secret; the hosted Control Plane MCP endpoint is unchanged. If migration fails,
disable standalone HTTP or use stdio instead of restoring unauthenticated access.

Release version and advisory fixed-version metadata remain pending release verification.
