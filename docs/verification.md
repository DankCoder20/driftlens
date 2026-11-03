# Verification record

Initial local build: 3 November 2026, Windows, Go 1.25.5, Node 22.12.0, Helm 4.3.0.

- Backend tests were executed successfully, including real Helm rendering of the eight-object demo chart, ordered values-file loading, structural comparisons, list ordering, API validation/auth, path containment, Secret redaction, Kubernetes adapter reads, topology evidence and SQLite restart recovery.
- Frontend component tests and the production TypeScript/Vite build were executed successfully. Tests cover resource selection, exact replica differences, onboarding actions and API authentication errors.
- Native Go service started successfully. Browser verification exercised authentication, honest empty-state onboarding, responsive layouts and repository creation. Screenshots are actual local application captures.
- Local Docker/Kind execution was blocked by Docker Desktop startup failure (`initializing Inference manager`, inaccessible local socket). The product did not connect to any pre-existing company cluster.
- Final application commit `539d609354f298cf147d8846a220273534fa0349` passed all four jobs in [GitHub Actions run 41643150216](https://github.com/shankarn-monkci/driftlens/actions/runs/41643150216): backend race tests/vet/build, frontend tests/build/audit, container build/Compose startup/HTTP health checks, and real Kind integration.
- The Kind test observed **8 IN_SYNC** resources initially. After deliberate mutations, the scan reported **5 IN_SYNC, 2 MODIFIED, 1 MISSING, 1 EXTRA**. Assertions verified the exact replicas `3 → 1` field change, changed container image, missing/extra ConfigMaps and degraded deployment health. Cleanup deleted the disposable cluster on the GitHub-hosted runner.
- Local suites contain **20 backend tests and 4 frontend tests**. The final frontend dependency audit reported **0 known vulnerabilities**. A repository check found no tracked local API token, kubeconfig, credentials directory, database or runtime data.
- Final local service restart recovered stored repository scans successfully. Its saved configuration contains **0 connected Kubernetes clusters** and only the bundled `driftlens-demo` local repository. No company infrastructure was accessed.

Subsequent documentation-only delivery commits do not change the application tested by that run. Hosting remains unconfigured; see `hosting.md` for deployment choices.
