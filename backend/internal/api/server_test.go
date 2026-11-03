package api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shankarn-monkci/driftlens/backend/internal/config"
	"github.com/shankarn-monkci/driftlens/backend/internal/storage"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	st, e := storage.Open(filepath.Join(dir, "test.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { st.DB.Close() })
	s, e := New(config.Config{Token: "test-token", DataDir: dir, RepositoryRoot: dir, CredentialRoot: dir}, st)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func request(s *Server, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}
func TestAuthAndValidation(t *testing.T) {
	s := newTestServer(t)
	if w := request(s, "GET", "/api/state", "", ""); w.Code != 401 {
		t.Fatalf("expected auth rejection: %d", w.Code)
	}
	if w := request(s, "GET", "/api/health", "", ""); w.Code != 200 {
		t.Fatal("health failed")
	}
	for _, body := range []string{`{"name":"bad","url":"https://token:secret@example.com/repo"}`, `{"name":"bad","url":"../../outside"}`, `{"name":"bad","url":"https://example.com/x","branch":"--upload-pack=bad"}`} {
		if w := request(s, "POST", "/api/repositories", body, "test-token"); w.Code != 400 {
			t.Fatalf("unsafe input accepted: %s (%d)", body, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/api/scans", nil)
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin request accepted")
	}
}
func TestPersistenceAndEmptyScan(t *testing.T) {
	s := newTestServer(t)
	w := request(s, "POST", "/api/repositories", `{"name":"public","url":"https://github.com/example/infra.git","branch":"main"}`, "test-token")
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	var saved map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &saved)
	if saved["id"] == "" {
		t.Fatal("missing ID")
	}
	w = request(s, "GET", "/api/repositories", "", "test-token")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	_ = s.Store.Delete("repositories", saved["id"].(string))
	if e := s.Scan(context.Background()); e != nil {
		t.Fatal(e)
	}
	restarted, e := New(s.Config, s.Store)
	if e != nil {
		t.Fatal(e)
	}
	if restarted.current().ID == "" {
		t.Fatal("snapshot did not survive restart")
	}
	if w = request(restarted, http.MethodGet, "/api/history", "", "test-token"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
}
