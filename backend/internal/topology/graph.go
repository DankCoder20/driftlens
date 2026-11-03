package topology

import (
	"fmt"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"sort"
)

type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}
type Graph struct {
	Nodes []inventory.Resource `json:"nodes"`
	Edges []Edge               `json:"edges"`
}

func Build(rs []inventory.Resource) Graph {
	g := Graph{Nodes: rs, Edges: []Edge{}}
	lookup := map[string]string{}
	seen := map[string]bool{}
	for _, r := range rs {
		lookup[r.Kind+"/"+r.Namespace+"/"+r.Name] = r.ID
	}
	add := func(a, b, l string) {
		key := a + "->" + b + ":" + l
		if a != "" && b != "" && a != b && !seen[key] {
			seen[key] = true
			g.Edges = append(g.Edges, Edge{key, a, b, l})
		}
	}
	for _, r := range rs {
		if r.Namespace != "" {
			add(lookup["Namespace//"+r.Namespace], r.ID, "contains")
		}
		if owners, ok := inventory.At(r.Object, "metadata", "ownerReferences").([]any); ok {
			for _, o := range owners {
				m := inventory.Map(o)
				add(lookup[inventory.Str(m["kind"])+"/"+r.Namespace+"/"+inventory.Str(m["name"])], r.ID, "owns")
			}
		}
		if r.Kind == "Service" {
			selector := inventory.Map(inventory.At(r.Object, "spec", "selector"))
			if len(selector) > 0 {
				for _, t := range rs {
					if t.Namespace != r.Namespace {
						continue
					}
					labels := t.Labels
					if t.Kind == "Deployment" || t.Kind == "StatefulSet" || t.Kind == "DaemonSet" {
						labels = inventory.Map(inventory.At(t.Object, "spec", "template", "metadata", "labels"))
					} else if t.Kind != "Pod" {
						continue
					}
					match := true
					for k, v := range selector {
						if fmt.Sprint(labels[k]) != fmt.Sprint(v) {
							match = false
						}
					}
					if match {
						add(r.ID, t.ID, "selects")
					}
				}
			}
		}
		var refs func(any)
		refs = func(v any) {
			switch m := v.(type) {
			case map[string]any:
				for k, x := range m {
					q := inventory.Map(x)
					kind := ""
					field := "name"
					switch k {
					case "configMap", "configMapRef", "configMapKeyRef":
						kind = "ConfigMap"
					case "secret", "secretRef", "secretKeyRef":
						kind = "Secret"
						if k == "secret" {
							field = "secretName"
						}
					case "persistentVolumeClaim":
						kind = "PersistentVolumeClaim"
						field = "claimName"
					case "service":
						if r.Kind == "Ingress" {
							kind = "Service"
						}
					}
					if kind != "" {
						add(r.ID, lookup[kind+"/"+r.Namespace+"/"+inventory.Str(q[field])], "references")
					}
					refs(x)
				}
			case []any:
				for _, x := range m {
					refs(x)
				}
			}
		}
		refs(r.Object["spec"])
	}
	sort.Slice(g.Edges, func(i, j int) bool { return g.Edges[i].ID < g.Edges[j].ID })
	return g
}
