package history

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/drift"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
)

type Change struct {
	Type     string `json:"type"`
	Resource string `json:"resource"`
	Before   string `json:"before,omitempty"`
	After    string `json:"after,omitempty"`
}

func Changes(before, after []drift.Result, oldResources, newResources []inventory.Resource) []Change {
	out := []Change{}
	old := map[string]drift.Result{}
	for _, r := range before {
		old[r.ID] = r
	}
	for _, r := range after {
		if p, ok := old[r.ID]; ok {
			if p.Classification != r.Classification {
				kind := "drift_changed"
				if r.Classification == "IN_SYNC" {
					kind = "drift_resolved"
				} else if p.Classification == "IN_SYNC" {
					kind = "drift_introduced"
				}
				if r.Classification == "UNKNOWN" {
					kind = "observation_unavailable"
				} else if p.Classification == "UNKNOWN" {
					kind = "observation_restored"
				}
				out = append(out, Change{kind, r.ID, p.Classification, r.Classification})
			}
			delete(old, r.ID)
		} else {
			out = append(out, Change{"comparison_added", r.ID, "", r.Classification})
		}
	}
	for _, r := range old {
		out = append(out, Change{"comparison_removed", r.ID, r.Classification, ""})
	}
	health := map[string]string{}
	for _, r := range oldResources {
		health[r.ClusterID+"|"+r.ID] = r.Health
	}
	for _, r := range newResources {
		key := r.ClusterID + "|" + r.ID
		if h, ok := health[key]; ok {
			if h != r.Health {
				out = append(out, Change{"health_changed", key, h, r.Health})
			}
			delete(health, key)
		} else {
			out = append(out, Change{"resource_added", key, "", r.Health})
		}
	}
	for key, h := range health {
		out = append(out, Change{"resource_removed", key, h, ""})
	}
	return out
}
