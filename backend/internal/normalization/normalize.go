package normalization

import (
	"github.com/shankarn-monkci/driftlens/backend/internal/inventory"
	"reflect"
	"strings"
)

func Normalize(o inventory.Object, ignores []string) inventory.Object {
	m := inventory.Clone(o)
	delete(m, "status")
	meta := inventory.Map(m["metadata"])
	for _, k := range []string{"resourceVersion", "uid", "managedFields", "creationTimestamp", "generation", "selfLink", "deletionTimestamp", "deletionGracePeriodSeconds"} {
		delete(meta, k)
	}
	a := inventory.Map(meta["annotations"])
	for _, k := range []string{"kubectl.kubernetes.io/last-applied-configuration", "deployment.kubernetes.io/revision", "meta.helm.sh/release-name", "meta.helm.sh/release-namespace"} {
		delete(a, k)
	}
	if len(a) == 0 {
		delete(meta, "annotations")
	}
	labels := inventory.Map(meta["labels"])
	if labels["app.kubernetes.io/managed-by"] == "Helm" {
		delete(labels, "app.kubernetes.io/managed-by")
	}
	if len(labels) == 0 {
		delete(meta, "labels")
	}
	// Ignore rules are JSON pointers, so dots and slashes in annotation names remain unambiguous.
	for _, p := range ignores {
		parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
		var curr any = m
		for i, k := range parts {
			k = strings.ReplaceAll(strings.ReplaceAll(k, "~1", "/"), "~0", "~")
			mm, ok := curr.(map[string]any)
			if !ok {
				break
			}
			if i == len(parts)-1 {
				delete(mm, k)
			} else {
				curr = mm[k]
			}
		}
	}
	return m
}

// These defaults only apply when Git omits the field. Explicit Git values always compare.
func Default(path string, v any) bool {
	if v == nil {
		return true
	}
	if m, ok := v.(map[string]any); ok && len(m) == 0 {
		return true
	}
	if a, ok := v.([]any); ok && len(a) == 0 {
		return true
	}
	defaults := map[string]any{"/spec/replicas": float64(1), "/spec/revisionHistoryLimit": float64(10), "/spec/progressDeadlineSeconds": float64(600), "/spec/type": "ClusterIP", "/spec/sessionAffinity": "None", "/spec/internalTrafficPolicy": "Cluster", "/spec/externalTrafficPolicy": "Cluster", "/spec/ipFamilyPolicy": "SingleStack", "/spec/podManagementPolicy": "OrderedReady", "/spec/persistentVolumeClaimRetentionPolicy/whenDeleted": "Retain", "/spec/persistentVolumeClaimRetentionPolicy/whenScaled": "Retain", "/spec/template/spec/restartPolicy": "Always", "/spec/template/spec/dnsPolicy": "ClusterFirst", "/spec/template/spec/schedulerName": "default-scheduler", "/spec/template/spec/terminationGracePeriodSeconds": float64(30), "/spec/restartPolicy": "Always", "/spec/dnsPolicy": "ClusterFirst", "/spec/schedulerName": "default-scheduler", "/spec/terminationGracePeriodSeconds": float64(30)}
	if x, ok := defaults[path]; ok {
		return reflect.DeepEqual(x, v)
	}
	for _, suffix := range []string{"/terminationMessagePath", "/terminationMessagePolicy", "/imagePullPolicy", "/protocol"} {
		if strings.HasSuffix(path, suffix) {
			switch suffix {
			case "/terminationMessagePath":
				return v == "/dev/termination-log"
			case "/terminationMessagePolicy":
				return v == "File"
			case "/imagePullPolicy":
				return v == "IfNotPresent" || v == "Always"
			case "/protocol":
				return v == "TCP"
			}
		}
	}
	switch path {
	case "/spec/clusterIP", "/spec/clusterIPs", "/spec/ipFamilies", "/spec/healthCheckNodePort":
		return true
	}
	return false
}
