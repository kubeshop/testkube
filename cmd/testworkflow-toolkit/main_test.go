package main

import "testing"

func TestIsLocalArtifactReceiver(t *testing.T) {
	if !isLocalArtifactReceiver([]string{"/toolkit", "local-artifact-receiver"}) {
		t.Fatal("expected local artifact receiver command to bypass workflow configuration")
	}
	if isLocalArtifactReceiver([]string{"/toolkit", "artifacts"}) {
		t.Fatal("ordinary toolkit action must retain workflow configuration requirements")
	}
}
