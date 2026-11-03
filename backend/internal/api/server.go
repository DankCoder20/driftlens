package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/shankarn-monkci/driftlens/backend/internal/analytics"
	"github.com/shankarn-monkci/driftlens/backend/internal/config"
	"github.com/shankarn-monkci/driftlens/backend/internal/drift"
	gitrepo "github.com/shankarn-monkci/driftlens/backend/internal/git"
	"github.com/shankarn-monkci/driftlens/backend/internal/helm"
	"github.com/shankarn-monkci/driftlens/backend/internal/history"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	kube "github.com/shankarn-monkci/driftlens/backend/internal/kubernetes"
	"github.com/shankarn-monkci/driftlens/backend/internal/storage"
	"github.com/shankarn-monkci/driftlens/backend/internal/topology"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	Mappings     []inventory.Mapping          `json:"mappings"`
	ID           string                       `json:"id"`
	CreatedAt    time.Time                    `json:"createdAt"`
	Status       string                       `json:"status"`
	Resources    []inventory.Resource         `json:"resources"`
	Drift        []drift.Result               `json:"drift"`
	Summary      analytics.Summary            `json:"summary"`
	Clusters     map[string]kube.Scan         `json:"clusters"`
	Repositories map[string]gitrepo.Discovery `json:"repositories"`
	Changes      []history.Change             `json:"changes"`
	Errors       map[string]string            `json:"errors"`
}
type Server struct {
	Config   config.Config
	Store    *storage.Store
	mu       sync.RWMutex
	configMu sync.Mutex
	latest   Snapshot
	running  bool
	events   map[chan string]bool
}

func ID() string { b := make([]byte, 12); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func New(c config.Config, st *storage.Store) (*Server, error) {
	s := &Server{Config: c, Store: st, events: map[chan string]bool{}, latest: empty()}
	var snapshots []Snapshot
	if e := st.Snapshots(1, &snapshots); e != nil {
		return nil, e
	}
	if len(snapshots) > 0 {
		s.latest = snapshots[0]
	}
	return s, nil
}
func empty() Snapshot {
	return Snapshot{Resources: []inventory.Resource{}, Drift: []drift.Result{}, Summary: analytics.Summarize(nil, nil), Clusters: map[string]kube.Scan{}, Repositories: map[string]gitrepo.Discovery{}, Changes: []history.Change{}, Errors: map[string]string{}}
}
func (s *Server) current() Snapshot { s.mu.RLock(); defer s.mu.RUnlock(); return s.latest }
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, msg string) {
	write(w, status, map[string]any{"error": map[string]string{"message": msg}})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		problem(w, 400, "Invalid JSON body or unsupported fields")
		return false
	}
	if e := d.Decode(&struct{}{}); e != io.EOF {
		problem(w, 400, "Expected a single JSON object")
		return false
	}
	return true
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		write(w, 200, map[string]string{"status": "ok", "service": "driftlens"})
	})
	m.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		running := s.running
		s.mu.RUnlock()
		snapshot := s.current()
		if r.URL.Query().Get("compact") == "true" {
			snapshot.Resources = []inventory.Resource{}
		}
		write(w, 200, map[string]any{"snapshot": snapshot, "scanning": running})
	})
	for _, kind := range []string{"repositories", "clusters", "mappings"} {
		m.HandleFunc("GET /api/"+kind, func(w http.ResponseWriter, r *http.Request) {
			var v []json.RawMessage
			if e := s.Store.List(kind, &v); e != nil {
				problem(w, 500, "Unable to read connections")
				return
			}
			write(w, 200, v)
		})
		m.HandleFunc("POST /api/"+kind, func(w http.ResponseWriter, r *http.Request) { s.saveConnection(w, r, kind) })
		m.HandleFunc("DELETE /api/"+kind+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.configMu.Lock()
			defer s.configMu.Unlock()
			if kind != "mappings" {
				var maps []inventory.Mapping
				if e := s.Store.List("mappings", &maps); e != nil {
					problem(w, 500, "Cannot read mappings")
					return
				}
				for _, m := range maps {
					if m.RepositoryID == r.PathValue("id") || m.ClusterID == r.PathValue("id") {
						problem(w, 409, "Remove dependent mappings first")
						return
					}
				}
			}
			if e := s.Store.Delete(kind, r.PathValue("id")); e != nil {
				problem(w, 500, "Unable to remove connection")
				return
			}
			write(w, 200, map[string]bool{"deleted": true})
		})
	}
	m.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) { write(w, 200, s.settings()) })
	m.HandleFunc("PUT /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var v inventory.Settings
		if !decode(w, r, &v) {
			return
		}
		if v.ScanInterval != 0 && (v.ScanInterval < 30 || v.ScanInterval > 86400) {
			problem(w, 400, "Scan interval must be 0 (manual), or 30–86400 seconds")
			return
		}
		if e := s.Store.Put("settings", "global", v); e != nil {
			problem(w, 500, "Unable to save settings")
			return
		}
		write(w, 200, v)
	})
	m.HandleFunc("POST /api/clusters/{id}/test", s.testCluster)
	m.HandleFunc("POST /api/repositories/{id}/test", s.testRepository)
	m.HandleFunc("GET /api/inventory", func(w http.ResponseWriter, r *http.Request) {
		out := []inventory.Resource{}
		for _, x := range s.current().Resources {
			q := r.URL.Query()
			if q.Get("cluster") != "" && x.ClusterID != q.Get("cluster") || q.Get("namespace") != "" && x.Namespace != q.Get("namespace") || q.Get("kind") != "" && x.Kind != q.Get("kind") {
				continue
			}
			out = append(out, x)
		}
		write(w, 200, out)
	})
	m.HandleFunc("GET /api/drift", func(w http.ResponseWriter, r *http.Request) {
		out := []drift.Result{}
		q := r.URL.Query()
		for _, d := range s.current().Drift {
			if q.Get("cluster") != "" && d.ClusterID != q.Get("cluster") || q.Get("repository") != "" && d.RepositoryID != q.Get("repository") || q.Get("environment") != "" && d.Environment != q.Get("environment") || q.Get("namespace") != "" && d.Namespace != q.Get("namespace") || q.Get("kind") != "" && d.Kind != q.Get("kind") || q.Get("classification") != "" && d.Classification != q.Get("classification") {
				continue
			}
			out = append(out, d)
		}
		write(w, 200, out)
	})
	m.HandleFunc("GET /api/drift/{id}", func(w http.ResponseWriter, r *http.Request) {
		for _, d := range s.current().Drift {
			if d.ID == r.PathValue("id") {
				write(w, 200, d)
				return
			}
		}
		problem(w, 404, "Resource comparison not found")
	})
	m.HandleFunc("GET /api/inventory/{id}", func(w http.ResponseWriter, r *http.Request) {
		for _, d := range s.current().Resources {
			if d.ID == r.PathValue("id") && d.ClusterID == r.URL.Query().Get("cluster") {
				write(w, 200, d)
				return
			}
		}
		problem(w, 404, "Resource not found")
	})
	m.HandleFunc("GET /api/topology/{id}", func(w http.ResponseWriter, r *http.Request) {
		out := []inventory.Resource{}
		resources := s.current().Resources
		if r.URL.Query().Get("source") == "desired" {
			resources = []inventory.Resource{}
			seen := map[string]bool{}
			for _, d := range s.current().Drift {
				if d.ClusterID == r.PathValue("id") && d.Desired != nil && !seen[d.Desired.ID] {
					x := *d.Desired
					x.ClusterID = d.ClusterID
					resources = append(resources, x)
					seen[x.ID] = true
				}
			}
		}
		for _, x := range resources {
			if x.ClusterID == r.PathValue("id") {
				out = append(out, x)
			}
		}
		write(w, 200, topology.Build(out))
	})
	m.HandleFunc("GET /api/analytics/overview", func(w http.ResponseWriter, r *http.Request) { write(w, 200, s.current().Summary) })
	for _, path := range []string{"history", "scans", "analytics/drift", "analytics/resources"} {
		m.HandleFunc("GET /api/"+path, func(w http.ResponseWriter, r *http.Request) {
			var all []Snapshot
			if e := s.Store.Snapshots(200, &all); e != nil {
				problem(w, 500, "Unable to read scan history")
				return
			}
			out := []map[string]any{}
			for _, x := range all {
				out = append(out, map[string]any{"id": x.ID, "createdAt": x.CreatedAt, "status": x.Status, "summary": x.Summary, "changes": x.Changes, "errors": x.Errors, "repositories": x.Repositories})
			}
			write(w, 200, out)
		})
	}
	m.HandleFunc("POST /api/scans", func(w http.ResponseWriter, r *http.Request) {
		if !s.StartScan() {
			problem(w, 409, "A scan is already running")
			return
		}
		write(w, 202, map[string]string{"status": "running"})
	})
	m.HandleFunc("GET /api/events", s.stream)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			problem(w, 404, "API endpoint not found")
			return
		}
		path := filepath.Join(s.Config.FrontendDir, filepath.Clean("/"+r.URL.Path))
		if info, e := os.Stat(path); e == nil && !info.IsDir() {
			http.ServeFile(w, r, path)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.Config.FrontendDir, "index.html"))
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		defer func() {
			if e := recover(); e != nil {
				slog.Error("request panicked", "path", r.URL.Path)
				problem(w, 500, "Internal server error")
			}
		}()
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/health" {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, e := url.Parse(origin)
				if e != nil || u.Host != r.Host {
					problem(w, 403, "Cross-origin API access is not allowed")
					return
				}
			}
			if s.Config.Token != "" {
				token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				if subtle.ConstantTimeCompare([]byte(token), []byte(s.Config.Token)) != 1 {
					problem(w, 401, "Enter the server API token to connect")
					return
				}
			}
		}
		m.ServeHTTP(w, r)
	})
}
func (s *Server) settings() inventory.Settings {
	var v []inventory.Settings
	if s.Store.List("settings", &v) == nil && len(v) > 0 {
		return v[0]
	}
	return inventory.Settings{ScanInterval: 300}
}
func (s *Server) saveConnection(w http.ResponseWriter, r *http.Request, kind string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	id := ID()
	var value any
	switch kind {
	case "repositories":
		var v inventory.Repository
		if !decode(w, r, &v) {
			return
		}
		v.ID = id
		if e := gitrepo.Validate(v); e != nil {
			problem(w, 400, e.Error())
			return
		}
		if !strings.Contains(v.URL, "://") {
			if _, e := config.SafePath(s.Config.RepositoryRoot, v.URL); e != nil {
				problem(w, 400, "Local repository must exist inside DRIFTLENS_REPOSITORY_ROOT")
				return
			}
		}
		value = v
	case "clusters":
		var v inventory.Cluster
		if !decode(w, r, &v) {
			return
		}
		v.ID = id
		if v.Name == "" || v.Kubeconfig == "" {
			problem(w, 400, "Cluster name and kubeconfig filename are required")
			return
		}
		if _, e := kube.New(s.Config, v); e != nil {
			problem(w, 400, e.Error())
			return
		}
		value = v
	case "mappings":
		var v inventory.Mapping
		if !decode(w, r, &v) {
			return
		}
		v.ID = id
		if e := helm.Validate(v); e != nil {
			problem(w, 400, e.Error())
			return
		}
		var repos []inventory.Repository
		var clusters []inventory.Cluster
		var mappings []inventory.Mapping
		if s.Store.List("repositories", &repos) != nil || s.Store.List("clusters", &clusters) != nil || s.Store.List("mappings", &mappings) != nil {
			problem(w, 500, "Unable to read connections")
			return
		}
		repoOK, clusterOK := false, false
		for _, x := range repos {
			repoOK = repoOK || x.ID == v.RepositoryID
		}
		for _, x := range clusters {
			clusterOK = clusterOK || x.ID == v.ClusterID
		}
		if !repoOK || !clusterOK {
			problem(w, 400, "Select an existing repository and cluster")
			return
		}
		for _, x := range mappings {
			if x.ClusterID == v.ClusterID && x.Namespace == v.Namespace {
				problem(w, 409, "The MVP supports one mapping per cluster namespace to prevent ambiguous extra-resource ownership")
				return
			}
		}
		value = v
	}
	if e := s.Store.Put(kind, id, value); e != nil {
		problem(w, 500, "Unable to save connection")
		return
	}
	write(w, 201, value)
}
func (s *Server) testCluster(w http.ResponseWriter, r *http.Request) {
	var all []inventory.Cluster
	if e := s.Store.List("clusters", &all); e != nil {
		problem(w, 500, "Unable to read clusters")
		return
	}
	for _, v := range all {
		if v.ID == r.PathValue("id") {
			c, e := kube.New(s.Config, v)
			if e != nil {
				problem(w, 400, e.Error())
				return
			}
			version, e := c.Test()
			if e != nil {
				problem(w, 502, e.Error())
				return
			}
			write(w, 200, map[string]string{"status": "connected", "version": version})
			return
		}
	}
	problem(w, 404, "Cluster not found")
}
func (s *Server) testRepository(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		problem(w, 409, "Wait for the active scan before testing a repository")
		return
	}
	s.running = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.running = false; s.mu.Unlock() }()
	var all []inventory.Repository
	if e := s.Store.List("repositories", &all); e != nil {
		problem(w, 500, "Unable to read repositories")
		return
	}
	for _, v := range all {
		if v.ID == r.PathValue("id") {
			ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
			defer cancel()
			p, sha, e := gitrepo.Checkout(ctx, s.Config, v)
			if e != nil {
				problem(w, 502, e.Error())
				return
			}
			d, e := gitrepo.Discover(p, sha)
			if e != nil {
				problem(w, 422, "Unable to discover repository contents")
				return
			}
			write(w, 200, d)
			return
		}
	}
	problem(w, 404, "Repository not found")
}
func (s *Server) broadcast(event string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.events {
		select {
		case ch <- event:
		default:
		}
	}
}
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		problem(w, 500, "Streaming unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	ch := make(chan string, 8)
	s.mu.Lock()
	s.events[ch] = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.events, ch); s.mu.Unlock() }()
	fmt.Fprint(w, "event: connected\ndata: {}\n\n")
	f.Flush()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case e := <-ch:
			fmt.Fprintf(w, "event: %s\ndata: {}\n\n", e)
			f.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			f.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
func (s *Server) StartScan() bool {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return false
	}
	s.running = true
	s.mu.Unlock()
	go func() {
		defer func() { s.mu.Lock(); s.running = false; s.mu.Unlock(); s.broadcast("scan-complete") }()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		s.broadcast("scan-started")
		if e := s.Scan(ctx); e != nil {
			slog.Error("scan persistence failed", "error", e.Error())
			s.broadcast("scan-failed")
		}
	}()
	return true
}
func (s *Server) Poll(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			interval := s.settings().ScanInterval
			if interval > 0 && time.Since(last) >= time.Duration(interval)*time.Second {
				if s.StartScan() {
					last = time.Now()
				}
			}
		}
	}
}
func (s *Server) Scan(ctx context.Context) error {
	old := s.current()
	n := empty()
	n.ID = ID()
	n.CreatedAt = time.Now().UTC()
	n.Status = "complete"
	var repos []inventory.Repository
	var clusters []inventory.Cluster
	var mappings []inventory.Mapping
	if e := s.Store.List("repositories", &repos); e != nil {
		return e
	}
	if e := s.Store.List("clusters", &clusters); e != nil {
		return e
	}
	if e := s.Store.List("mappings", &mappings); e != nil {
		return e
	}
	n.Mappings = mappings
	roots := map[string]string{}
	for _, r := range repos {
		p, sha, e := gitrepo.Checkout(ctx, s.Config, r)
		if e != nil {
			n.Errors[r.ID] = e.Error()
			continue
		}
		d, e := gitrepo.Discover(p, sha)
		if e != nil {
			n.Errors[r.ID] = "Repository discovery failed"
			continue
		}
		roots[r.ID] = p
		n.Repositories[r.ID] = d
	}
	for _, c := range clusters {
		client, e := kube.New(s.Config, c)
		if e != nil {
			n.Errors[c.ID] = e.Error()
			continue
		}
		data, e := client.Collect(ctx, c.ID)
		if e != nil {
			n.Errors[c.ID] = e.Error()
			continue
		}
		n.Resources = append(n.Resources, data.Resources...)
		data.Resources = nil
		n.Clusters[c.ID] = data
		for key, msg := range data.Failures {
			n.Errors[c.ID+":"+key] = msg
		}
	}
	for _, m := range mappings {
		unavailable := func(reason string) {
			for _, previous := range old.Drift {
				if previous.MappingID == m.ID {
					previous.Classification = "UNKNOWN"
					previous.Reason = reason + "; last-known desired state shown"
					previous.Differences = []drift.Difference{}
					previous.Actual = nil
					n.Drift = append(n.Drift, previous)
				}
			}
		}
		root, ok := roots[m.RepositoryID]
		if !ok {
			n.Errors[m.ID] = "Desired state unavailable: repository scan failed"
			unavailable(n.Errors[m.ID])
			continue
		}
		desired, e := helm.Render(ctx, root, m)
		if e != nil {
			n.Errors[m.ID] = e.Error()
			unavailable(n.Errors[m.ID])
			continue
		}
		actual := []inventory.Resource{}
		for _, r := range n.Resources {
			if r.ClusterID == m.ClusterID {
				actual = append(actual, r)
			}
		}
		fail := map[string]string{}
		if data, ok := n.Clusters[m.ClusterID]; ok {
			for k, v := range data.Failures {
				fail[k] = v
			}
			for _, d := range desired {
				found := false
				for _, a := range actual {
					if a.Group == d.Group && a.Kind == d.Kind {
						found = true
						break
					}
				}
				if !found && data.Failures["discovery"] != "" {
					fail[d.Group+"/"+d.Kind] = "Discovery incomplete; absence cannot be confirmed"
				}
			}
		} else {
			for _, d := range desired {
				fail[d.Group+"/"+d.Kind] = "Cluster unavailable; absence cannot be confirmed"
			}
		}
		n.Drift = append(n.Drift, drift.Compare(desired, actual, m, fail)...)
	}
	n.Summary = analytics.Summarize(n.Resources, n.Drift)
	// Failed or incomplete list scopes cannot prove that resources were added or removed.
	comparable := func(rs []inventory.Resource) []inventory.Resource {
		out := []inventory.Resource{}
		for _, r := range rs {
			cluster, ok := n.Clusters[r.ClusterID]
			if ok && cluster.Failures["discovery"] == "" && cluster.Failures[r.Group+"/"+r.Kind] == "" {
				out = append(out, r)
			}
		}
		return out
	}
	n.Changes = history.Changes(old.Drift, n.Drift, comparable(old.Resources), comparable(n.Resources))
	if len(n.Errors) > 0 {
		n.Status = "partial"
	}
	if e := s.Store.Snapshot(n.ID, n.CreatedAt.Format(time.RFC3339Nano), n); e != nil {
		return e
	}
	s.mu.Lock()
	s.latest = n
	s.mu.Unlock()
	return nil
}
