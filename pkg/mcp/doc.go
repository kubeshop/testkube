// Package mcp implements the Model Context Protocol server that lets AI assistants work with
// Testkube.
//
// It exposes tools across workflows, executions, artifacts and metadata through three entry
// points: `testkube mcp serve` (the CLI), the testkube/mcp-server Docker image, and the Control
// Plane's /mcp endpoint for each environment.
//
// Tools are designed around interfaces. A new tool has to be registered twice: in server.go here
// and in the Control Plane's mcp_handler.go (testkube-cloud-api), or the hosted MCP will not offer
// it.
//
// README.md in this directory describes the architecture, the tool patterns and usage examples.
// The Insights board tools keep their rules in the boards package; the tools package documents how
// they write.
package mcp
