package kubernetes

import (
	"context"
	"fmt"
	"github.com/shankarn-monkci/driftlens/backend/internal/config"
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"os"
	"strings"
	"time"
)

type Client struct {
	Dynamic   dynamic.Interface
	Discovery discovery.DiscoveryInterface
	Metadata  metadata.Interface
}
type Scan struct {
	Resources  []inventory.Resource `json:"resources"`
	Failures   map[string]string    `json:"failures"`
	Version    string               `json:"version"`
	NodeCount  int                  `json:"nodeCount"`
	Namespaces []string             `json:"namespaces"`
	Status     string               `json:"status"`
}

func New(c config.Config, cluster inventory.Cluster) (*Client, error) {
	p, e := config.SafePath(c.CredentialRoot, cluster.Kubeconfig)
	if e != nil {
		return nil, fmt.Errorf("kubeconfig must be a readable file inside the credential directory")
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return nil, fmt.Errorf("cannot read kubeconfig")
	}
	raw, e := clientcmd.Load(b)
	if e != nil {
		return nil, fmt.Errorf("invalid kubeconfig")
	}
	if cluster.Context != "" {
		raw.CurrentContext = cluster.Context
	}
	for _, a := range raw.AuthInfos {
		if a.Exec != nil && !c.AllowExec {
			return nil, fmt.Errorf("exec credential plugins are disabled; use a read-only token/certificate kubeconfig or explicitly enable trusted exec auth")
		}
		if a.AuthProvider != nil {
			return nil, fmt.Errorf("legacy auth providers are not supported")
		}
		for _, path := range []*string{&a.ClientCertificate, &a.ClientKey, &a.TokenFile} {
			if *path != "" {
				var resolved string
				if resolved, e = config.SafePath(c.CredentialRoot, *path); e != nil {
					return nil, fmt.Errorf("credential file is outside allowed directory; embed credentials in kubeconfig")
				}
				*path = resolved
			}
		}
	}
	for _, cl := range raw.Clusters {
		if cl.CertificateAuthority != "" {
			if cl.CertificateAuthority, e = config.SafePath(c.CredentialRoot, cl.CertificateAuthority); e != nil {
				return nil, fmt.Errorf("CA file is outside allowed directory; embed CA data")
			}
		}
	}
	cfg, e := clientcmd.NewDefaultClientConfig(*raw, &clientcmd.ConfigOverrides{}).ClientConfig()
	if e != nil {
		return nil, fmt.Errorf("invalid kubeconfig context or credentials")
	}
	return FromConfig(cfg)
}
func FromConfig(cfg *rest.Config) (*Client, error) {
	cfg = rest.CopyConfig(cfg)
	cfg.Timeout = 15 * time.Second
	cfg.QPS = 5
	cfg.Burst = 10
	cfg.UserAgent = "driftlens/read-only"
	d, e := dynamic.NewForConfig(cfg)
	if e != nil {
		return nil, fmt.Errorf("invalid Kubernetes client configuration")
	}
	disc, e := discovery.NewDiscoveryClientForConfig(cfg)
	if e != nil {
		return nil, fmt.Errorf("invalid discovery client configuration")
	}
	m, e := metadata.NewForConfig(cfg)
	if e != nil {
		return nil, fmt.Errorf("invalid metadata client configuration")
	}
	return &Client{d, disc, m}, nil
}
func Friendly(err error) string {
	if apierrors.IsUnauthorized(err) {
		return "Unauthorized: credentials may be expired or invalid"
	}
	if apierrors.IsForbidden(err) {
		return "Insufficient RBAC permissions: read/list access is required"
	}
	if apierrors.IsNotFound(err) {
		return "Resource API is unavailable"
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return "Kubernetes API server timed out"
	}
	s := err.Error()
	if strings.Contains(s, "certificate") {
		return "TLS certificate validation failed; check the cluster CA and certificate expiry"
	}
	return "Kubernetes API request failed; verify endpoint connectivity and credentials"
}
func (c *Client) Test() (string, error) {
	v, e := c.Discovery.ServerVersion()
	if e != nil {
		return "", fmt.Errorf("%s", Friendly(e))
	}
	return v.GitVersion, nil
}
func (c *Client) Collect(ctx context.Context, id string) (Scan, error) {
	s := Scan{Resources: []inventory.Resource{}, Failures: map[string]string{}, Namespaces: []string{}, Status: "connected"}
	v, e := c.Test()
	if e != nil {
		return s, e
	}
	s.Version = v
	lists, e := c.Discovery.ServerPreferredResources()
	if e != nil {
		s.Status = "partial"
		s.Failures["discovery"] = Friendly(e)
	}
	for _, list := range lists {
		gv, e := schema.ParseGroupVersion(list.GroupVersion)
		if e != nil {
			continue
		}
		for _, r := range list.APIResources {
			if strings.Contains(r.Name, "/") || r.Kind == "Event" && gv.Group != "" {
				continue
			}
			listable := false
			for _, v := range r.Verbs {
				if v == "list" {
					listable = true
				}
			}
			if !listable {
				continue
			}
			gvr := gv.WithResource(r.Name)
			key := gv.Group + "/" + r.Kind
			if r.Kind == "Secret" {
				var next string
				for {
					items, e := c.Metadata.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{Limit: 500, Continue: next})
					if e != nil {
						s.Failures[key] = Friendly(e)
						s.Status = "partial"
						break
					}
					for _, m := range items.Items {
						o := inventory.Object{"apiVersion": list.GroupVersion, "kind": "Secret", "metadata": map[string]any{"name": m.Name, "namespace": m.Namespace, "labels": toAny(m.Labels), "annotations": toAny(m.Annotations)}}
						x := inventory.New(o, "")
						x.ClusterID = id
						s.Resources = append(s.Resources, x)
					}
					next = items.Continue
					if next == "" {
						break
					}
				}
				continue
			}
			var ri dynamic.ResourceInterface = c.Dynamic.Resource(gvr)
			if r.Namespaced {
				ri = c.Dynamic.Resource(gvr).Namespace("")
			}
			var next string
			for {
				items, e := ri.List(ctx, metav1.ListOptions{Limit: 500, Continue: next})
				if e != nil {
					s.Failures[key] = Friendly(e)
					s.Status = "partial"
					break
				}
				for _, u := range items.Items {
					u.SetAPIVersion(list.GroupVersion)
					u.SetKind(r.Kind)
					x := inventory.New(u.Object, "")
					x.ClusterID = id
					if r.Kind == "Namespace" {
						s.Namespaces = append(s.Namespaces, x.Name)
					}
					if r.Kind == "Node" {
						s.NodeCount++
					}
					s.Resources = append(s.Resources, x)
				}
				next = items.GetContinue()
				if next == "" {
					break
				}
				if len(s.Resources) > 50000 {
					return s, fmt.Errorf("cluster exceeds the 50000 resource MVP safety limit")
				}
			}
		}
	}
	return s, nil
}
func toAny(m map[string]string) map[string]any {
	o := map[string]any{}
	for k, v := range m {
		o[k] = v
	}
	return o
}
