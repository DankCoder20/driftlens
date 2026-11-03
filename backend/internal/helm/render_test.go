package helm

import (
	"context"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRealHelmRender(t *testing.T) {
	if _, e := exec.LookPath("helm"); e != nil {
		t.Skip("Helm is not installed")
	}
	root, e := filepath.Abs("../../../demo/infra-repo")
	if e != nil {
		t.Fatal(e)
	}
	m := inventory.Mapping{Name: "dev", RepositoryID: "repo", ClusterID: "cluster", ChartPath: "charts/platform", Namespace: "demo", Release: "platform", ValuesFiles: []string{"charts/platform/values-dev.yaml"}}
	rs, e := Render(context.Background(), root, m)
	if e != nil {
		t.Fatal(e)
	}
	if len(rs) != 8 {
		t.Fatalf("expected eight chart objects, got %d", len(rs))
	}
	found := false
	for _, r := range rs {
		if r.Name == "api" && r.Kind == "Deployment" {
			found = true
			if inventory.Num(inventory.At(r.Object, "spec", "replicas")) != 3 {
				t.Fatal("replicas lost")
			}
		}
		if r.Kind == "ConfigMap" && inventory.At(r.Object, "data", "LOG_LEVEL") != "debug" {
			t.Fatal("values file was not merged")
		}
	}
	if !found {
		t.Fatal("missing API deployment")
	}
	m.ValuesFiles = []string{"../../outside.yaml"}
	if _, e = Render(context.Background(), root, m); e == nil {
		t.Fatal("accepted escaping values path")
	}
}
