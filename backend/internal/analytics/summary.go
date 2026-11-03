package analytics

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/drift"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
)

type Summary struct {
	Total           int            `json:"total"`
	Classifications map[string]int `json:"classifications"`
	Kinds           map[string]int `json:"kinds"`
	Namespaces      map[string]int `json:"namespaces"`
	Clusters        map[string]int `json:"clusters"`
	Health          map[string]int `json:"health"`
	Images          map[string]int `json:"images"`
	DesiredReplicas int64          `json:"desiredReplicas"`
	ReadyReplicas   int64          `json:"readyReplicas"`
	Unhealthy       int            `json:"unhealthy"`
}

func Summarize(rs []inventory.Resource, ds []drift.Result) Summary {
	s := Summary{Total: len(rs), Classifications: map[string]int{}, Kinds: map[string]int{}, Namespaces: map[string]int{}, Clusters: map[string]int{}, Health: map[string]int{}, Images: map[string]int{}}
	for _, d := range ds {
		s.Classifications[d.Classification]++
	}
	for _, r := range rs {
		s.Kinds[r.Kind]++
		ns := r.Namespace
		if ns == "" {
			ns = "cluster-scoped"
		}
		s.Namespaces[ns]++
		s.Clusters[r.ClusterID]++
		if r.Kind == "Pod" || r.Kind == "Deployment" || r.Kind == "StatefulSet" || r.Kind == "DaemonSet" {
			s.Health[r.Health]++
			if r.Health == "Degraded" || r.Health == "Failed" || r.Health == "Pending" {
				s.Unhealthy++
			}
		}
		if r.Kind == "Deployment" || r.Kind == "StatefulSet" {
			s.DesiredReplicas += r.DesiredReplicas
			s.ReadyReplicas += r.ReadyReplicas
		}
		if r.Kind == "Pod" {
			containers, _ := inventory.At(r.Object, "spec", "containers").([]any)
			for _, c := range containers {
				image := inventory.Str(inventory.Map(c)["image"])
				if image != "" {
					s.Images[image]++
				}
			}
		}
	}
	return s
}
