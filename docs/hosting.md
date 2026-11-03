# Hosting after the initial build

The application is prepared for Docker Compose but is not published to an external host. It needs a persistent volume, Git access, and network connectivity to the selected Kubernetes API servers.

## Practical options

1. **Private VM in the cluster’s network:** run Compose with a persistent data volume and read-only credential/source mounts. Put TLS and SSO at an internal reverse proxy; keep the backend private. This is the simplest first deployment.
2. **A deployment inside a non-production Kubernetes cluster:** package the Go API and frontend images, use a persistent volume for SQLite, and mount read-only credentials. Add a Helm chart for DriftLens itself as a follow-up. Cluster-local authentication is not yet wired as a separate connector; provide a kubeconfig that references the mounted service-account token.
3. **Workstation/VPN:** run the native application and connect through existing network access. This is useful for evaluating multiple environments without adding a public service.

Static-only hosting cannot run Go, SQLite, Git, Helm and Kubernetes discovery. Hosting only the frontend would leave the primary product functionality unavailable. Select the backend location first, then DNS/TLS and access control.

Before deployment, decide the environment, endpoint reachability, read-only identity provisioning, persistence/backup location, expected inventory size, scan cadence and authentication boundary. Use a fresh strong API token and a TLS reverse proxy. Do not expose kubeconfig mounts or the SQLite directory through Nginx.

For local Kind + Compose, `demo-docker.kubeconfig` rewrites the API endpoint to `host.docker.internal` while preserving TLS server-name verification. Docker Desktop supports this host path. On rootful Linux, Kind may publish its API only on loopback; configure a dedicated reachable bridge bind address for that disposable cluster, or use the native application for the demo. Never disable TLS verification to solve routing problems.
