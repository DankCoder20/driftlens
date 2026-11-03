package drift

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"testing"
)

func resource(spec map[string]any) inventory.Resource {
	return inventory.New(inventory.Object{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]any{"name": "api", "namespace": "dev"}, "spec": spec}, "dev")
}
func mapping() inventory.Mapping {
	return inventory.Mapping{ID: "m", RepositoryID: "r", ClusterID: "c", Name: "dev", Namespace: "dev", Release: "platform"}
}
func TestReplicaDrift(t *testing.T) {
	d := resource(map[string]any{"replicas": 3})
	a := resource(map[string]any{"replicas": 1})
	out := Compare([]inventory.Resource{d}, []inventory.Resource{a}, mapping(), nil)
	if len(out) != 1 || out[0].Classification != "MODIFIED" || len(out[0].Differences) != 1 || out[0].Differences[0].Path != "/spec/replicas" {
		t.Fatalf("unexpected diff: %+v", out)
	}
}
func TestClassificationAndScope(t *testing.T) {
	d := resource(map[string]any{"replicas": 3})
	extra := inventory.New(inventory.Object{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "manual", "namespace": "dev"}}, "")
	other := inventory.New(inventory.Object{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "unrelated", "namespace": "prod"}}, "")
	out := Compare([]inventory.Resource{d}, []inventory.Resource{extra, other}, mapping(), nil)
	counts := map[string]int{}
	for _, r := range out {
		counts[r.Classification]++
	}
	if counts["MISSING"] != 1 || counts["EXTRA"] != 1 || len(out) != 2 {
		t.Fatalf("wrong classifications: %+v", out)
	}
	out = Compare([]inventory.Resource{d}, nil, mapping(), map[string]string{"apps/Deployment": "forbidden"})
	if out[0].Classification != "UNKNOWN" {
		t.Fatal("RBAC failure was classified as missing")
	}
}
func TestNamedListsReorderAndNestedChanges(t *testing.T) {
	d := resource(map[string]any{"template": map[string]any{"spec": map[string]any{"containers": []any{map[string]any{"name": "a", "image": "a:1"}, map[string]any{"name": "b", "image": "b:1"}}}}})
	a := resource(map[string]any{"template": map[string]any{"spec": map[string]any{"containers": []any{map[string]any{"name": "b", "image": "b:1"}, map[string]any{"name": "a", "image": "a:1"}}}}})
	out := Compare([]inventory.Resource{d}, []inventory.Resource{a}, mapping(), nil)
	if out[0].Classification != "IN_SYNC" {
		t.Fatalf("list reorder drift: %+v", out[0].Differences)
	}
	containers := inventory.At(a.Object, "spec", "template", "spec", "containers").([]any)
	inventory.Map(containers[1])["image"] = "a:2"
	out = Compare([]inventory.Resource{d}, []inventory.Resource{a}, mapping(), nil)
	if len(out[0].Differences) != 1 || out[0].Differences[0].Path != "/spec/template/spec/containers[name=a]/image" {
		t.Fatalf("incorrect nested diff: %+v", out[0].Differences)
	}
}
func TestScalarListsRemainOrdered(t *testing.T) {
	var diffs []Difference
	walk("/spec/args", []any{"a", "b"}, []any{"b", "a"}, &diffs)
	if len(diffs) != 2 {
		t.Fatal("ordered arguments must not be sorted")
	}
}
func TestDefaultsAndAnnotations(t *testing.T) {
	d := resource(map[string]any{})
	a := resource(map[string]any{"replicas": 1, "revisionHistoryLimit": 10})
	a.Object["status"] = map[string]any{"readyReplicas": 1}
	inventory.Map(a.Object["metadata"])["resourceVersion"] = "100"
	out := Compare([]inventory.Resource{d}, []inventory.Resource{a}, mapping(), nil)
	if out[0].Classification != "IN_SYNC" {
		t.Fatalf("defaulted fields drift: %+v", out[0].Differences)
	}
	inventory.Map(a.Object["metadata"])["annotations"] = map[string]any{"operational.example/setting": "changed"}
	out = Compare([]inventory.Resource{d}, []inventory.Resource{a}, mapping(), nil)
	if out[0].Classification != "MODIFIED" {
		t.Fatal("operational annotations must compare")
	}
}
func TestSameNameDifferentAPIGroup(t *testing.T) {
	a := inventory.New(inventory.Object{"apiVersion": "a/v1", "kind": "Thing", "metadata": map[string]any{"name": "same", "namespace": "dev"}}, "")
	b := inventory.New(inventory.Object{"apiVersion": "b/v1", "kind": "Thing", "metadata": map[string]any{"name": "same", "namespace": "dev"}}, "")
	out := Compare([]inventory.Resource{a}, []inventory.Resource{b}, mapping(), nil)
	if len(out) != 2 {
		t.Fatal("resources from different API groups must not match")
	}
}
func TestControllerChildrenAreNotExtra(t *testing.T) {
	r := inventory.New(inventory.Object{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]any{"name": "api-xyz", "namespace": "dev", "ownerReferences": []any{map[string]any{"kind": "ReplicaSet", "name": "api"}}}}, "")
	if len(Compare(nil, []inventory.Resource{r}, mapping(), nil)) != 0 {
		t.Fatal("runtime child reported extra")
	}
}
