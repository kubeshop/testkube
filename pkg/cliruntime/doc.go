// Package cliruntime exposes lightweight runtime-context detection helpers
// (Docker, CI, interactive local, and the AI tool that invoked the CLI):
// IsRunningInDocker, DockerContext, CliRunContext and DetectAITool. It is
// intentionally dependency-free so it can be imported by both the telemetry
// package and the CLI commands packages without creating an import cycle.
//
// DetectAITool reports the AI coding agent that invoked the CLI (claude-code,
// codex, cursor, gemini-cli, or "") purely from env vars the agents set in their
// subprocesses.
//
// Telemetry and the CLI update check both read this package, so a detection
// added here reaches both. A new CI or runtime detection goes in context.go. A
// new AI tool needs its env-var check in DetectAITool and a TestDetectAITool
// case in context_test.go; no telemetry wiring changes are needed, since the
// payloads already read the AITool field.
package cliruntime
