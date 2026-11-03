# Architecture and semantics

## Scan lifecycle

The API persists connection definitions separately from observations. Each scan checks out configured repositories (or reads an allowed local working tree), discovers charts/files, inventories each configured cluster once, then renders and compares each explicit mapping. Unmapped clusters still contribute inventory and health.

A snapshot records the Git commit, chart metadata and values-file configuration (through the mapping), observed resource bodies, comparison results, cluster discovery status, aggregated counts, and changes relative to the previous snapshot. SQLite commits each snapshot atomically and retains 200 snapshots. Periodic scans and manual scans share a single-flight guard. SSE invalidates browser query caches; periodic query refresh is the fallback.

Credentials are outside the database: cluster records contain a kubeconfig filename/context only. Private Git credentials are owned by the server’s agent/helper configuration. Runtime observations contain infrastructure configuration, so protect the SQLite volume as operationally sensitive data.

## Matching and normalization

Identity is `(API group, kind, namespace, name)`. API version is compared as content rather than identity. A configured mapping selects one Git environment, release, cluster and namespace. Runtime controller children remain in inventory/topology but are not automatically classified as extra drift. Standalone unexpected resources in the mapped namespace are `EXTRA` with `unmanaged` ownership. Resources explicitly owned by another Helm release are excluded from that mapping’s extras. Helm release-storage Secrets and common namespace-generated objects are also excluded.

Normalization clones its input. It removes status and volatile top-level metadata, Helm bookkeeping annotations and kubectl last-applied content. Other annotations/labels remain meaningful. Known defaults and allocated Service addresses are ignored only when Git omits them; an explicitly declared field always compares. Ignore rules are exact JSON pointers (`/metadata/annotations/example.com~1key`), currently traversing object maps, not wildcard expressions.

Nested maps compare recursively. Container/env lists match by unique name, volume mounts by mount path, and ports by port/protocol. Duplicate or unknown merge keys fall back to positional comparison. Scalar arrays remain ordered. Differences include an inspectable path plus desired and actual JSON values. A missing resource API/failed list produces `UNKNOWN`, not a false missing result. Failed repository/render observations retain last-known comparisons as `UNKNOWN` rather than claiming removals.

`Secret` resources are metadata-only. This intentionally means DriftLens cannot detect Secret value drift. Operational labels and safe Helm ownership annotations remain visible; other Secret annotations and payloads are removed.

## Health and topology

Deployment/StatefulSet health considers ready replicas and incomplete updated replicas. Pod health uses phase and container readiness. These are observations, not Prometheus metrics. Service selectors, owner references, Ingress backends, ConfigMap/Secret/PVC references and namespace membership provide graph edges. Similar names alone never create an edge.

Charts derive from stored snapshots, with a minimum of two snapshots for a trend. Partial scans are identified in the UI and history; reduced counts from partial inventories should not be interpreted as confirmed infrastructure removal. The initial historical diff records a baseline of added mapped resources. Subsequent transitions record classification and workload-health changes.

## Deployment boundaries

Browser → same-origin Go API or Nginx proxy. The backend needs Git egress and connectivity to Kubernetes API endpoints; cluster credentials never go to a hosted third party. Go reads through client-go and invokes Git/Helm with argument arrays, never a shell. Rendering has no post-renderer or repository-script execution path. Kubernetes write capabilities exist only in the explicitly invoked demo scripts.
