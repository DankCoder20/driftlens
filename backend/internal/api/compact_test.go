package api

import (
	"encoding/json"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"testing"
)

func TestCompactStateKeepsCountsWithoutCopyingInventory(t *testing.T) {
	s := newTestServer(t)
	s.latest.Resources = []inventory.Resource{{ID: "pod/test", Kind: "Pod", Name: "test"}}
	s.latest.Summary.Total = 1
	w := request(s, "GET", "/api/state?compact=true", "", "test-token")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var response struct {
		Snapshot Snapshot `json:"snapshot"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Snapshot.Resources) != 0 || response.Snapshot.Summary.Total != 1 {
		t.Fatal("compact state lost counts or included inventory")
	}
	if len(s.current().Resources) != 1 {
		t.Fatal("compact response mutated the stored snapshot")
	}
	w = request(s, "GET", "/api/state", "", "test-token")
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Snapshot.Resources) != 1 {
		t.Fatal("full state no longer includes inventory")
	}
}
