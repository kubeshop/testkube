package cliruntime

import (
	"os"
	"testing"
)

// aiToolEnvVars lists every environment variable DetectAITool inspects. Each
// test case unsets all of them and then sets only the ones under test, so cases
// don't leak into one another and the ambient environment can't affect results.
var aiToolEnvVars = []string{
	"CLAUDECODE",
	"CODEX_SANDBOX",
	"CODEX_SANDBOX_NETWORK_DISABLED",
	"CURSOR_TRACE_ID",
	"CURSOR_AGENT",
	"GEMINI_CLI",
}

// clearAIToolEnv unsets every AI-tool env var and registers cleanup to restore
// each to its original state after the test.
func clearAIToolEnv(t *testing.T) {
	t.Helper()
	for _, key := range aiToolEnvVars {
		if orig, ok := os.LookupEnv(key); ok {
			t.Cleanup(func() { os.Setenv(key, orig) })
		} else {
			t.Cleanup(func() { os.Unsetenv(key) })
		}
		os.Unsetenv(key)
	}
}

func TestDetectAITool(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "no AI tool",
			env:  nil,
			want: AIToolNone,
		},
		{
			name: "claude code",
			env:  map[string]string{"CLAUDECODE": "1"},
			want: "claude-code",
		},
		{
			name: "claude code empty value is ignored",
			env:  map[string]string{"CLAUDECODE": ""},
			want: AIToolNone,
		},
		{
			name: "codex sandbox",
			env:  map[string]string{"CODEX_SANDBOX": "seatbelt"},
			want: "codex",
		},
		{
			name: "codex sandbox network disabled",
			env:  map[string]string{"CODEX_SANDBOX_NETWORK_DISABLED": "1"},
			want: "codex",
		},
		{
			name: "codex sandbox present but empty",
			env:  map[string]string{"CODEX_SANDBOX": ""},
			want: "codex",
		},
		{
			name: "cursor trace id",
			env:  map[string]string{"CURSOR_TRACE_ID": "abc123"},
			want: "cursor",
		},
		{
			name: "cursor trace id empty falls through",
			env:  map[string]string{"CURSOR_TRACE_ID": ""},
			want: AIToolNone,
		},
		{
			name: "cursor agent",
			env:  map[string]string{"CURSOR_AGENT": "1"},
			want: "cursor",
		},
		{
			name: "gemini cli",
			env:  map[string]string{"GEMINI_CLI": "1"},
			want: "gemini-cli",
		},
		{
			name: "claude code takes precedence over others",
			env: map[string]string{
				"CLAUDECODE":   "1",
				"GEMINI_CLI":   "1",
				"CURSOR_AGENT": "1",
			},
			want: "claude-code",
		},
		{
			name: "codex takes precedence over cursor and gemini",
			env: map[string]string{
				"CODEX_SANDBOX": "seatbelt",
				"CURSOR_AGENT":  "1",
				"GEMINI_CLI":    "1",
			},
			want: "codex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAIToolEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			if got := DetectAITool(); got != tt.want {
				t.Errorf("DetectAITool() = %q, want %q", got, tt.want)
			}
		})
	}
}
