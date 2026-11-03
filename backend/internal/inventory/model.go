package inventory

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"time"
)

type Object = map[string]any
type Resource struct {
	ID              string         `json:"id"`
	Group           string         `json:"group"`
	APIVersion      string         `json:"apiVersion"`
	Kind            string         `json:"kind"`
	Namespace       string         `json:"namespace"`
	Name            string         `json:"name"`
	Labels          map[string]any `json:"labels"`
	Annotations     map[string]any `json:"annotations"`
	RepositoryID    string         `json:"repositoryId,omitempty"`
	SourceFile      string         `json:"sourceFile,omitempty"`
	Release         string         `json:"release,omitempty"`
	ClusterID       string         `json:"clusterId,omitempty"`
	ObservedAt      time.Time      `json:"observedAt"`
	Health          string         `json:"health"`
	DesiredReplicas int64          `json:"desiredReplicas"`
	ReadyReplicas   int64          `json:"readyReplicas"`
	Object          Object         `json:"object"`
}
type Repository struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}
type Cluster struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kubeconfig string `json:"kubeconfig"`
	Context    string `json:"context,omitempty"`
}
type Mapping struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RepositoryID string   `json:"repositoryId"`
	ClusterID    string   `json:"clusterId"`
	ChartPath    string   `json:"chartPath"`
	ManifestPath string   `json:"manifestPath,omitempty"`
	Release      string   `json:"release"`
	Namespace    string   `json:"namespace"`
	ValuesFiles  []string `json:"valuesFiles"`
	KubeVersion  string   `json:"kubeVersion,omitempty"`
	APIVersions  []string `json:"apiVersions,omitempty"`
	IgnorePaths  []string `json:"ignorePaths"`
}
type Settings struct {
	ScanInterval int `json:"scanInterval"`
}

func Map(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
func Str(v any) string { s, _ := v.(string); return s }
func Num(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	case json.Number:
		x, _ := n.Int64()
		return x
	}
	return 0
}
func At(o Object, keys ...string) any {
	var v any = o
	for _, k := range keys {
		v = Map(v)[k]
	}
	return v
}
func Clone(o Object) Object {
	b, _ := json.Marshal(o)
	var m Object
	_ = json.Unmarshal(b, &m)
	return m
}
func New(o Object, namespace string) Resource {
	o = Clone(o)
	meta := Map(o["metadata"])
	kind := Str(o["kind"])
	// A Secret is never retained beyond its identity and safe Helm ownership metadata.
	if kind == "Secret" {
		a := Map(meta["annotations"])
		safe := map[string]any{}
		for _, k := range []string{"meta.helm.sh/release-name", "meta.helm.sh/release-namespace"} {
			if v, ok := a[k]; ok {
				safe[k] = v
			}
		}
		meta = map[string]any{"name": meta["name"], "namespace": meta["namespace"], "labels": meta["labels"], "annotations": safe}
		o = Object{"apiVersion": o["apiVersion"], "kind": kind, "metadata": meta}
	}
	// Last-applied can embed Secret data even on objects used by external tooling.
	delete(Map(meta["annotations"]), "kubectl.kubernetes.io/last-applied-configuration")
	o["metadata"] = meta
	ns := Str(meta["namespace"])
	if ns == "" && !ClusterScoped(kind) {
		ns = namespace
		meta["namespace"] = ns
	}
	api := Str(o["apiVersion"])
	group := ""
	if strings.Contains(api, "/") {
		group = strings.SplitN(api, "/", 2)[0]
	}
	r := Resource{ID: group + "/" + kind + "/" + ns + "/" + Str(meta["name"]), Group: group, APIVersion: api, Kind: kind, Namespace: ns, Name: Str(meta["name"]), Labels: Map(meta["labels"]), Annotations: Map(meta["annotations"]), ObservedAt: time.Now().UTC(), Object: o, Health: "Unknown"}
	r.Release = Str(r.Annotations["meta.helm.sh/release-name"])
	switch kind {
	case "Deployment", "StatefulSet", "ReplicaSet":
		r.DesiredReplicas = Num(At(o, "spec", "replicas"))
		r.ReadyReplicas = Num(At(o, "status", "readyReplicas"))
		r.Health = "Healthy"
		if r.ReadyReplicas < r.DesiredReplicas {
			r.Health = "Degraded"
		}
		if kind == "Deployment" || kind == "StatefulSet" {
			if updated := At(o, "status", "updatedReplicas"); updated != nil && Num(updated) < r.DesiredReplicas {
				r.Health = "Degraded"
			}
		}
	case "DaemonSet":
		r.DesiredReplicas = Num(At(o, "status", "desiredNumberScheduled"))
		r.ReadyReplicas = Num(At(o, "status", "numberReady"))
		r.Health = "Healthy"
		if r.ReadyReplicas < r.DesiredReplicas {
			r.Health = "Degraded"
		}
	case "Pod":
		r.Health = Str(At(o, "status", "phase"))
		if r.Health == "Running" {
			for _, c := range slice(At(o, "status", "containerStatuses")) {
				m := Map(c)
				if m["ready"] != true {
					r.Health = "Degraded"
				}
			}
		}
	}
	return r
}
func slice(v any) []any { a, _ := v.([]any); return a }
func ClusterScoped(kind string) bool {
	switch kind {
	case "Namespace", "Node", "ClusterRole", "ClusterRoleBinding", "CustomResourceDefinition", "PersistentVolume", "StorageClass", "IngressClass", "APIService", "MutatingWebhookConfiguration", "ValidatingWebhookConfiguration", "PriorityClass":
		return true
	}
	return false
}
func Parse(data []byte, namespace string) ([]Resource, error) {
	d := yaml.NewDecoder(strings.NewReader(string(data)))
	out := []Resource{}
	for {
		var o Object
		err := d.Decode(&o)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid manifest YAML: %w", err)
		}
		if len(o) == 0 {
			continue
		}
		if Str(o["kind"]) == "List" {
			for _, v := range slice(o["items"]) {
				r := New(Map(v), namespace)
				if r.Name != "" {
					out = append(out, r)
				}
			}
			continue
		}
		if Str(o["kind"]) == "" {
			continue
		}
		r := New(o, namespace)
		if r.Name == "" {
			return nil, fmt.Errorf("%s has no metadata.name", r.Kind)
		}
		out = append(out, r)
	}
	return out, nil
}
