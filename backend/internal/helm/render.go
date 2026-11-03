package helm

import (
	"context"
	"fmt"
	"github.com/shankarn-monkci/driftlens/backend/internal/config"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var name = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$`)

func Validate(m inventory.Mapping) error {
	if m.Name == "" || m.RepositoryID == "" || m.ClusterID == "" || !name.MatchString(m.Release) || !name.MatchString(m.Namespace) || len(m.Release) > 53 || len(m.Namespace) > 63 {
		return fmt.Errorf("environment, repository, cluster, valid release and namespace are required")
	}
	if m.ChartPath == "" && m.ManifestPath == "" {
		return fmt.Errorf("chartPath or manifestPath is required")
	}
	for _, p := range m.IgnorePaths {
		if !strings.HasPrefix(p, "/") {
			return fmt.Errorf("ignore paths must be JSON pointers beginning with /")
		}
	}
	return nil
}
func Render(ctx context.Context, root string, m inventory.Mapping) ([]inventory.Resource, error) {
	if err := Validate(m); err != nil {
		return nil, err
	}
	out := []inventory.Resource{}
	if m.ChartPath != "" {
		chart, e := config.SafePath(root, m.ChartPath)
		if e != nil {
			return nil, e
		}
		// Helm can follow symlinks while reading charts: reject them before invoking the process.
		if e = filepath.WalkDir(chart, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("chart symlinks are not allowed")
			}
			return nil
		}); e != nil {
			return nil, e
		}
		args := []string{"template", m.Release, chart, "--namespace", m.Namespace, "--include-crds", "--skip-tests"}
		for _, v := range m.ValuesFiles {
			p, e := config.SafePath(root, v)
			if e != nil {
				return nil, e
			}
			args = append(args, "-f", p)
		}
		if m.KubeVersion != "" {
			args = append(args, "--kube-version", m.KubeVersion)
		}
		for _, a := range m.APIVersions {
			args = append(args, "--api-versions", a)
		}
		cmd := exec.CommandContext(ctx, "helm", args...)
		b, e := cmd.Output()
		if e != nil {
			return nil, fmt.Errorf("Helm rendering failed: verify chart dependencies, selected values, Kubernetes version and Helm installation (renderer output withheld to protect values)")
		}
		rs, e := inventory.Parse(b, m.Namespace)
		if e != nil {
			return nil, fmt.Errorf("Helm returned invalid manifests")
		}
		for i := range rs {
			rs[i].SourceFile = m.ChartPath + "/templates"
		}
		out = append(out, rs...)
	}
	if m.ManifestPath != "" {
		p, e := config.SafePath(root, m.ManifestPath)
		if e != nil {
			return nil, e
		}
		e = filepath.WalkDir(p, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("manifest symlinks are not allowed")
			}
			if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
				return nil
			}
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			rs, e := inventory.Parse(b, m.Namespace)
			if e != nil {
				return fmt.Errorf("invalid manifest YAML")
			}
			rel, _ := filepath.Rel(root, path)
			for i := range rs {
				rs[i].SourceFile = filepath.ToSlash(rel)
			}
			out = append(out, rs...)
			return nil
		})
		if e != nil {
			return nil, e
		}
	}
	seen := map[string]bool{}
	for i := range out {
		r := &out[i]
		if seen[r.ID] {
			return nil, fmt.Errorf("duplicate resource identity %s", r.ID)
		}
		seen[r.ID] = true
		r.RepositoryID = m.RepositoryID
		r.Release = m.Release
		r.Health = "Not observed"
	}
	return out, nil
}
