package drift

import (
	"fmt"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"github.com/shankarn-monkci/driftlens/backend/internal/normalization"
	"reflect"
	"sort"
	"strings"
)

type Difference struct {
	Path           string `json:"path"`
	Desired        any    `json:"desired"`
	Actual         any    `json:"actual"`
	Classification string `json:"classification"`
}
type Result struct {
	ID             string              `json:"id"`
	MappingID      string              `json:"mappingId"`
	RepositoryID   string              `json:"repositoryId"`
	Environment    string              `json:"environment"`
	ClusterID      string              `json:"clusterId"`
	Kind           string              `json:"kind"`
	Namespace      string              `json:"namespace"`
	Name           string              `json:"name"`
	Classification string              `json:"classification"`
	Ownership      string              `json:"ownership"`
	Reason         string              `json:"reason,omitempty"`
	Desired        *inventory.Resource `json:"desired,omitempty"`
	Actual         *inventory.Resource `json:"actual,omitempty"`
	Differences    []Difference        `json:"differences"`
}

func Compare(desired, actual []inventory.Resource, m inventory.Mapping, failed map[string]string) []Result {
	a := map[string]inventory.Resource{}
	for _, r := range actual {
		a[r.ID] = r
	}
	out := []Result{}
	for _, d := range desired {
		r := Result{ID: m.ID + "|" + d.ID, MappingID: m.ID, RepositoryID: m.RepositoryID, Environment: m.Name, ClusterID: m.ClusterID, Kind: d.Kind, Namespace: d.Namespace, Name: d.Name, Desired: &d, Classification: "MISSING", Ownership: "mapped", Differences: []Difference{}}
		if x, ok := a[d.ID]; ok {
			r.Actual = &x
			r.Classification = "IN_SYNC"
			walk("", normalization.Normalize(d.Object, m.IgnorePaths), normalization.Normalize(x.Object, m.IgnorePaths), &r.Differences)
			if len(r.Differences) > 0 {
				r.Classification = "MODIFIED"
			}
			delete(a, d.ID)
		} else if reason, ok := failed[d.Group+"/"+d.Kind]; ok {
			r.Classification = "UNKNOWN"
			r.Reason = reason
		}
		out = append(out, r)
	}
	for _, x := range a {
		if x.Kind == "Secret" && strings.HasPrefix(x.Name, "sh.helm.release.v1.") && x.Labels["owner"] == "helm" {
			continue
		}
		if x.Namespace != m.Namespace {
			continue
		}
		if owners, ok := inventory.At(x.Object, "metadata", "ownerReferences").([]any); ok && len(owners) > 0 {
			continue
		}
		if x.Kind == "Event" || x.Kind == "Endpoints" || x.Kind == "EndpointSlice" || x.Kind == "ServiceAccount" && x.Name == "default" || x.Kind == "ConfigMap" && x.Name == "kube-root-ca.crt" {
			continue
		}
		ownership := "unmanaged"
		if x.Release == m.Release {
			ownership = "release"
		} else if x.Release != "" {
			continue
		}
		out = append(out, Result{ID: m.ID + "|" + x.ID, MappingID: m.ID, RepositoryID: m.RepositoryID, Environment: m.Name, ClusterID: m.ClusterID, Kind: x.Kind, Namespace: x.Namespace, Name: x.Name, Actual: &x, Classification: "EXTRA", Ownership: ownership, Differences: []Difference{}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func escape(k string) string { return strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1") }
func walk(path string, d, a any, out *[]Difference) {
	if reflect.DeepEqual(d, a) {
		return
	}
	dm, dok := d.(map[string]any)
	am, aok := a.(map[string]any)
	if dok && aok {
		keys := map[string]bool{}
		for k := range dm {
			keys[k] = true
		}
		for k := range am {
			keys[k] = true
		}
		ordered := []string{}
		for k := range keys {
			ordered = append(ordered, k)
		}
		sort.Strings(ordered)
		for _, k := range ordered {
			p := path + "/" + escape(k)
			dv, exists := dm[k]
			av := am[k]
			if !exists && defaultTree(p, av) {
				continue
			}
			walk(p, dv, av, out)
		}
		return
	}
	dl, dok := d.([]any)
	al, aok := a.([]any)
	if dok && aok {
		// Kubernetes associative lists: merge by name, mountPath, or port/protocol. Scalar arrays stay ordered.
		key := listKey(path, dl, al)
		if key != "" {
			di := index(dl, key)
			ai := index(al, key)
			keys := map[string]bool{}
			for k := range di {
				keys[k] = true
			}
			for k := range ai {
				keys[k] = true
			}
			ordered := []string{}
			for k := range keys {
				ordered = append(ordered, k)
			}
			sort.Strings(ordered)
			for _, k := range ordered {
				walk(path+"["+key+"="+k+"]", di[k], ai[k], out)
			}
			return
		}
		n := len(dl)
		if len(al) > n {
			n = len(al)
		}
		for i := 0; i < n; i++ {
			var x, y any
			if i < len(dl) {
				x = dl[i]
			}
			if i < len(al) {
				y = al[i]
			}
			walk(fmt.Sprintf("%s/%d", path, i), x, y, out)
		}
		return
	}
	*out = append(*out, Difference{path, d, a, "MODIFIED"})
}
func defaultTree(p string, v any) bool {
	if normalization.Default(p, v) {
		return true
	}
	if m, ok := v.(map[string]any); ok {
		for k, x := range m {
			if !defaultTree(p+"/"+escape(k), x) {
				return false
			}
		}
		return true
	}
	return false
}
func listKey(path string, a, b []any) string {
	candidates := []string{"name"}
	if strings.HasSuffix(path, "/volumeMounts") {
		candidates = []string{"mountPath"}
	}
	if strings.HasSuffix(path, "/ports") {
		candidates = []string{"containerPort", "port", "name"}
	}
	for _, key := range candidates {
		ok := true
		for _, list := range [][]any{a, b} {
			seen := map[string]bool{}
			for _, v := range list {
				m, isMap := v.(map[string]any)
				if !isMap || m[key] == nil {
					ok = false
					break
				}
				s := fmt.Sprint(m[key])
				if key == "port" || key == "containerPort" {
					s += "/" + protocol(m)
				}
				if seen[s] {
					ok = false
					break
				}
				seen[s] = true
			}
		}
		if ok {
			return key
		}
	}
	return ""
}
func protocol(m map[string]any) string {
	p := inventory.Str(m["protocol"])
	if p == "" {
		return "TCP"
	}
	return p
}
func index(a []any, key string) map[string]any {
	out := map[string]any{}
	for _, v := range a {
		m := inventory.Map(v)
		s := fmt.Sprint(m[key])
		if key == "port" || key == "containerPort" {
			s += "/" + protocol(m)
		}
		out[s] = v
	}
	return out
}
