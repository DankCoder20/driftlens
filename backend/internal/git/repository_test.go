package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOperatorCredentialHelperIsPreserved(t *testing.T) {
	dir := t.TempDir()
	system := filepath.Join(dir, "system.gitconfig")
	global := filepath.Join(dir, "global.gitconfig")
	if err := os.WriteFile(system, []byte("[credential]\n\thelper = driftlens-test-helper\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(global, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_SYSTEM", system)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "")
	helper, err := run(context.Background(), dir, "config", "--get", "credential.helper")
	if err != nil || helper != "driftlens-test-helper" {
		t.Fatalf("operator credential helper unavailable: %q, %v", helper, err)
	}
	for _, key := range []string{"protocol.file.allow", "protocol.ext.allow"} {
		value, err := run(context.Background(), dir, "config", "--get", key)
		if err != nil || value != "never" {
			t.Fatalf("unsafe transport enabled: %s=%s", key, value)
		}
	}
}
