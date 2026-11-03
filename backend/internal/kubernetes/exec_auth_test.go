package kubernetes

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/config"
	"testing"
)

func TestExecAuthUsesExactOptIn(t *testing.T) {
	if execAllowed(config.Config{}, "gke-gcloud-auth-plugin.exe") {
		t.Fatal("exec auth enabled by default")
	}
	cfg := config.Config{ExecAuthCommands: "gke-gcloud-auth-plugin.exe"}
	if !execAllowed(cfg, "gke-gcloud-auth-plugin.exe") {
		t.Fatal("approved plugin rejected")
	}
	for _, command := range []string{"powershell.exe", "cmd.exe", "C:/untrusted/gke-gcloud-auth-plugin.exe"} {
		if execAllowed(cfg, command) {
			t.Fatalf("unexpected command allowed: %s", command)
		}
	}
}
