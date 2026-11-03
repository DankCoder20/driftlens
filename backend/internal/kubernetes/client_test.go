package kubernetes

import (
	"context"
	"encoding/json"
	"k8s.io/client-go/rest"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadOnlyAdapterAndSecretMetadata(t *testing.T) {
	metadataRequested := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("unexpected mutation %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			_, _ = w.Write([]byte(`{"major":"1","minor":"34","gitVersion":"v1.34.0"}`))
		case "/api":
			_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"]}`))
		case "/apis":
			_, _ = w.Write([]byte(`{"kind":"APIGroupList","groups":[]}`))
		case "/api/v1":
			_, _ = w.Write([]byte(`{"kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"pods","kind":"Pod","namespaced":true,"verbs":["list"]},{"name":"secrets","kind":"Secret","namespaced":true,"verbs":["list"]}]}`))
		case "/api/v1/pods":
			_, _ = w.Write([]byte(`{"kind":"PodList","apiVersion":"v1","items":[{"metadata":{"name":"api","namespace":"dev"},"status":{"phase":"Running"}}]}`))
		case "/api/v1/secrets":
			metadataRequested = strings.Contains(r.Header.Get("Accept"), "PartialObjectMetadataList")
			_, _ = w.Write([]byte(`{"kind":"PartialObjectMetadataList","apiVersion":"meta.k8s.io/v1","items":[{"metadata":{"name":"token","namespace":"dev","annotations":{"private":"DO_NOT_LEAK"}}}]}`))
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c, e := FromConfig(&rest.Config{Host: srv.URL})
	if e != nil {
		t.Fatal(e)
	}
	s, e := c.Collect(context.Background(), "dev")
	if e != nil {
		t.Fatal(e)
	}
	if !metadataRequested {
		t.Fatal("Secret request must request metadata only")
	}
	b, _ := json.Marshal(s)
	if strings.Contains(string(b), "DO_NOT_LEAK") {
		t.Fatal("Secret annotation leaked")
	}
	if len(s.Resources) != 2 || s.Version != "v1.34.0" {
		t.Fatalf("unexpected inventory: %+v", s)
	}
}
func TestUnreachableCluster(t *testing.T) {
	c, e := FromConfig(&rest.Config{Host: "http://127.0.0.1:1"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Test(); e == nil {
		t.Fatal("unreachable cluster passed")
	}
}
