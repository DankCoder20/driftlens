package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Addr, DataDir, RepositoryRoot, CredentialRoot, Token, FrontendDir string
	AllowExec                                                         bool
}

func env(k, d string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return d
}
func Load() (Config, error) {
	c := Config{Addr: env("DRIFTLENS_ADDR", "127.0.0.1:8080"), DataDir: env("DRIFTLENS_DATA_DIR", ".data"), RepositoryRoot: env("DRIFTLENS_REPOSITORY_ROOT", "../demo"), CredentialRoot: env("DRIFTLENS_CREDENTIAL_ROOT", "../credentials"), Token: os.Getenv("DRIFTLENS_API_TOKEN"), FrontendDir: env("DRIFTLENS_FRONTEND_DIR", "../frontend/dist"), AllowExec: os.Getenv("DRIFTLENS_ALLOW_EXEC_AUTH") == "true"}
	if c.Token == "" && !strings.HasPrefix(c.Addr, "127.0.0.1:") && !strings.HasPrefix(c.Addr, "localhost:") {
		return c, fmt.Errorf("DRIFTLENS_API_TOKEN is required when listening beyond loopback")
	}
	for _, p := range []string{c.DataDir, c.CredentialRoot} {
		if err := os.MkdirAll(p, 0700); err != nil {
			return c, err
		}
	}
	return c, nil
}

// SafePath resolves symlinks as well as .. before checking containment.
func SafePath(root, path string) (string, error) {
	r, e := filepath.Abs(root)
	if e != nil {
		return "", e
	}
	r, e = filepath.EvalSymlinks(r)
	if e != nil {
		return "", e
	}
	p := path
	if !filepath.IsAbs(p) {
		p = filepath.Join(r, p)
	}
	p, e = filepath.EvalSymlinks(p)
	if e != nil {
		return "", e
	}
	rel, e := filepath.Rel(r, p)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path is outside the configured directory")
	}
	return p, nil
}
