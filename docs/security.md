# Security model

## Isolation for this installation

Automated development tests use this repository's bundled sample infrastructure only. Live integration tests run in a disposable `driftlens-demo` Kind cluster created on a GitHub-hosted Actions runner. The demo's kubectl and Helm mutations explicitly select its dedicated kubeconfig/context; they do not use a workstation's default Kubernetes context. Company infrastructure is never part of the automated test setup. Operator-requested application connections are separate, read-only, and retained only in local runtime configuration; they are not committed or used by CI.

## Application boundary

DriftLens is a single-operator, self-hosted MVP. Use a private workstation, VPN or authenticated internal reverse proxy. The default listener and Compose published port bind to loopback. Binding Go beyond loopback requires an API token. Do not put the token in query parameters or commit it. Nginx forwards Authorization to Go; SSE uses authenticated fetch streaming.

Kubernetes access consists of discovery, version and list requests. The API exposes no apply/patch/delete operation. Supply a read-only identity; the application cannot downgrade privileges embedded in an administrator’s kubeconfig. The included demo deliberately uses a separate read-only service account. Secret access uses `PartialObjectMetadataList`; defensive sanitization also removes any payload and non-ownership Secret annotations before persistence.

Kubeconfigs are referenced under an operator-configured directory. They never appear in API responses, snapshots or logs. Credential file references must remain under that root. Exec authentication is disabled by default because kubeconfig exec plugins run a local command. Prefer `DRIFTLENS_EXEC_AUTH_COMMANDS` with a comma-separated list of exact trusted commands (for example `gke-gcloud-auth-plugin.exe` on Windows). A different executable path is not approved merely because its basename matches. `DRIFTLENS_ALLOW_EXEC_AUTH=true` remains available for operator-provisioned configurations that intentionally allow all exec commands. There is no browser kubeconfig-upload endpoint.

Local repository and values paths are resolved, including symlinks, before checking containment. Chart and manifest symlinks are rejected. Remote repositories accept HTTPS or SSH URLs without embedded passwords/tokens. Git uses noninteractive, restricted-protocol commands; local repository scripts, submodule updates and Helm post-renderers are not executed. Operators remain responsible for the trusted server Git configuration and credential helpers.

Git/Helm stderr is not returned verbatim, because rendering and authentication errors can contain sensitive values. Errors instead identify the failing stage and troubleshooting action. Kubernetes errors are classified without echoing bearer tokens, client certificates or full kubeconfig content.

ConfigMaps and non-Secret manifest configuration are visible by design, including plaintext values that an infrastructure author incorrectly placed there. Restrict access to the API and SQLite volume accordingly. This MVP is not a general-purpose secret scanner, an encrypted credential vault, or a tenant isolation boundary.

Request bodies have size limits, HTTP headers/read operations have timeouts, client-go requests have 15-second timeouts and rate limits, and scans use a five-minute context. Helm renders only with fixed argument names. Backups should protect the entire data directory; deleting connections does not erase historical snapshots.
