package topology

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"testing"
)

func TestRelationshipsRequireEvidence(t *testing.T) {
	s := inventory.New(inventory.Object{"apiVersion": "v1", "kind": "Service", "metadata": map[string]any{"name": "api", "namespace": "dev"}, "spec": map[string]any{"selector": map[string]any{"app": "api"}}}, "")
	p := inventory.New(inventory.Object{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]any{"name": "pod", "namespace": "dev", "labels": map[string]any{"app": "api"}}}, "")
	other := inventory.New(inventory.Object{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]any{"name": "other", "namespace": "dev"}}, "")
	g := Build([]inventory.Resource{s, p, other})
	if len(g.Edges) != 1 || g.Edges[0].Target != p.ID {
		t.Fatalf("invented relationship: %+v", g.Edges)
	}
}
