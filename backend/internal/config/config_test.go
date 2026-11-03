package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafePath(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "inside")
	if e := os.WriteFile(inside, []byte("ok"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := SafePath(root, "inside"); e != nil {
		t.Fatal(e)
	}
	outside := t.TempDir()
	if _, e := SafePath(root, outside); e == nil {
		t.Fatal("accepted path outside root")
	}
	if e := os.Symlink(outside, filepath.Join(root, "link")); e == nil {
		if _, e = SafePath(root, "link"); e == nil {
			t.Fatal("accepted escaping symlink")
		}
	}
}
