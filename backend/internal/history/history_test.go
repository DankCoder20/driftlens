package history

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/drift"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"testing"
)

func TestRuntimeResourceAndHealthChanges(t *testing.T) {
	old := []inventory.Resource{{ID: "pod/a", ClusterID: "c", Health: "Running"}, {ID: "pod/b", ClusterID: "c", Health: "Running"}}
	now := []inventory.Resource{{ID: "pod/a", ClusterID: "c", Health: "Degraded"}, {ID: "pod/c", ClusterID: "c", Health: "Pending"}}
	changes := Changes(nil, nil, old, now)
	counts := map[string]int{}
	for _, c := range changes {
		counts[c.Type]++
	}
	if counts["health_changed"] != 1 || counts["resource_added"] != 1 || counts["resource_removed"] != 1 {
		t.Fatalf("wrong changes: %+v", changes)
	}
}
func TestUnavailableObservationIsNotDrift(t *testing.T) {
	before := []drift.Result{{ID: "x", Classification: "IN_SYNC"}}
	after := []drift.Result{{ID: "x", Classification: "UNKNOWN"}}
	changes := Changes(before, after, nil, nil)
	if len(changes) != 1 || changes[0].Type != "observation_unavailable" {
		t.Fatalf("wrong transition: %+v", changes)
	}
}
