package git

import (
	"context"
	"fmt"
	"github.com/shankarn-monkci/driftlens/backend/internal/config"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"gopkg.in/yaml.v3"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Chart struct {
	Path        string   `json:"path"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	ValuesFiles []string `json:"valuesFiles"`
}
type Discovery struct {
	Commit string   `json:"commit"`
	Files  []string `json:"files"`
	Charts []Chart  `json:"charts"`
}

var branchPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

func Validate(r inventory.Repository) error {
	if strings.TrimSpace(r.Name) == "" || r.URL == "" {
		return fmt.Errorf("name and URL/path are required")
	}
	if r.Branch != "" && (!branchPattern.MatchString(r.Branch) || strings.Contains(r.Branch, "..")) {
		return fmt.Errorf("invalid branch")
	}
	if strings.Contains(r.URL, "://") {
		u, e := url.Parse(r.URL)
		if e != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "ssh") {
			return fmt.Errorf("use an HTTPS, ssh:// URL, or an allowed local directory")
		}
		if u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("repository URL must not contain a query or fragment")
		}
		if u.User != nil {
			if _, ok := u.User.Password(); ok || u.Scheme != "ssh" {
				return fmt.Errorf("credentials in URLs are prohibited; use a server-side Git credential helper or SSH agent")
			}
		}
	}
	return nil
}
func run(ctx context.Context, dir string, args ...string) (string, error) {
	args = append([]string{"-c", "core.hooksPath=/dev/null", "-c", "protocol.file.allow=never", "-c", "protocol.ext.allow=never"}, args...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=https:ssh", "GIT_CONFIG_NOSYSTEM=1")
	b, e := cmd.Output()
	if e != nil {
		return "", fmt.Errorf("git operation failed (check URL, branch, network and server-side credentials)")
	}
	return strings.TrimSpace(string(b)), nil
}
func Checkout(ctx context.Context, c config.Config, r inventory.Repository) (string, string, error) {
	if e := Validate(r); e != nil {
		return "", "", e
	}
	if !strings.Contains(r.URL, "://") {
		p, e := config.SafePath(c.RepositoryRoot, r.URL)
		if e != nil {
			return "", "", e
		}
		sha, _ := run(ctx, p, "rev-parse", "HEAD")
		if sha == "" {
			sha = "local-working-tree"
		}
		return p, sha, nil
	}
	root := filepath.Join(c.DataDir, "repos")
	if e := os.MkdirAll(root, 0700); e != nil {
		return "", "", e
	}
	p := filepath.Join(root, r.ID)
	if _, e := os.Stat(p); os.IsNotExist(e) {
		args := []string{"clone", "--depth", "1", "--no-recurse-submodules"}
		if r.Branch != "" {
			args = append(args, "--branch", r.Branch)
		}
		args = append(args, "--", r.URL, p)
		if _, e = run(ctx, "", args...); e != nil {
			return "", "", e
		}
	} else {
		branch := r.Branch
		if branch == "" {
			branch = "HEAD"
		}
		if _, e = run(ctx, p, "fetch", "--depth", "1", "origin", branch); e != nil {
			return "", "", e
		}
		if _, e = run(ctx, p, "reset", "--hard", "FETCH_HEAD"); e != nil {
			return "", "", e
		}
	}
	sha, e := run(ctx, p, "rev-parse", "HEAD")
	return p, sha, e
}
func Discover(root, commit string) (Discovery, error) {
	d := Discovery{Commit: commit, Files: []string{}, Charts: []Chart{}}
	err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			if e.Name() == ".git" || e.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if e.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		d.Files = append(d.Files, filepath.ToSlash(rel))
		if len(d.Files) > 10000 {
			return fmt.Errorf("repository exceeds 10000 files")
		}
		if e.Name() == "Chart.yaml" {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			var m map[string]any
			if err = yaml.Unmarshal(b, &m); err != nil {
				return err
			}
			dir := filepath.Dir(p)
			cr, _ := filepath.Rel(root, dir)
			chart := Chart{Path: filepath.ToSlash(cr), Name: inventory.Str(m["name"]), Version: inventory.Str(m["version"]), Description: inventory.Str(m["description"]), ValuesFiles: []string{}}
			_ = filepath.WalkDir(dir, func(v string, f fs.DirEntry, e error) error {
				if e != nil {
					return e
				}
				if f.IsDir() && (f.Name() == "templates" || f.Name() == "charts") {
					return filepath.SkipDir
				}
				if !f.IsDir() && f.Type()&os.ModeSymlink == 0 && strings.Contains(f.Name(), "values") && (strings.HasSuffix(v, ".yaml") || strings.HasSuffix(v, ".yml")) {
					vr, _ := filepath.Rel(root, v)
					chart.ValuesFiles = append(chart.ValuesFiles, filepath.ToSlash(vr))
				}
				return nil
			})
			d.Charts = append(d.Charts, chart)
		}
		return nil
	})
	return d, err
}
