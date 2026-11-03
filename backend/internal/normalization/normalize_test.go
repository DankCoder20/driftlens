package normalization

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"testing"
)

func TestGeneratedMetadataAndPointer(t *testing.T) {
	o := inventory.Object{"kind": "Deployment", "metadata": map[string]any{"uid": "private", "labels": map[string]any{"team": "platform"}, "annotations": map[string]any{"a.b/c": "x", "keep": "yes"}}, "status": map[string]any{"readyReplicas": 1}}
	m := Normalize(o, []string{"/metadata/annotations/a.b~1c"})
	if m["status"] != nil || inventory.At(m, "metadata", "uid") != nil || inventory.At(m, "metadata", "annotations", "a.b/c") != nil {
		t.Fatal("normalization failed")
	}
	if inventory.At(m, "metadata", "labels", "team") != "platform" || inventory.At(m, "metadata", "annotations", "keep") != "yes" {
		t.Fatal("meaningful metadata removed")
	}
	if o["status"] == nil {
		t.Fatal("input mutated")
	}
}
