package inventory

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMultidocumentAndSecretRedaction(t *testing.T) {
	b := []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: password\n  annotations:\n    leak: TOPSECRET\ndata:\n  password: TOPSECRET\nstringData:\n  token: TOPSECRET\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: settings\n")
	r, e := Parse(b, "dev")
	if e != nil || len(r) != 2 {
		t.Fatalf("parse: %v %#v", e, r)
	}
	j, _ := json.Marshal(r)
	if strings.Contains(string(j), "TOPSECRET") {
		t.Fatal("secret leaked")
	}
	if r[0].Namespace != "dev" {
		t.Fatal("namespace default failed")
	}
}
func TestHealthFromStatus(t *testing.T) {
	r := New(Object{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]any{"name": "api"}, "spec": map[string]any{"replicas": 3}, "status": map[string]any{"readyReplicas": 1}}, "dev")
	if r.Health != "Degraded" || r.ReadyReplicas != 1 {
		t.Fatal("incorrect deployment health")
	}
}
