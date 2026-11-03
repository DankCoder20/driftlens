# Verification record

Initial local build: 3 November 2026, Windows, Go 1.25.5, Node 22.12.0, Helm 4.3.0.

- Backend tests were executed successfully, including real Helm rendering of the eight-object demo chart, ordered values-file loading, structural comparisons, list ordering, API validation/auth, path containment, Secret redaction, Kubernetes adapter reads, topology evidence and SQLite restart recovery.
- Frontend component tests and the production TypeScript/Vite build were executed successfully. Tests cover resource selection, exact replica differences, onboarding actions and API authentication errors.
- Native Go service started successfully. Browser verification exercised authentication, honest empty-state onboarding, responsive layouts and repository creation. Screenshots are actual local application captures.
- Local Docker/Kind execution was blocked by Docker Desktop startup failure (`initializing Inference manager`, inaccessible local socket). The product did not connect to any pre-existing company cluster.
- GitHub CI includes real Kind installation, initial-sync verification, deliberate mutation and verification of replica/image/missing/extra/health differences. Its result is recorded after the initial push; a defined job alone is not evidence that it passed.

The final delivery message reports final test/dependency/CI results after any follow-up corrections. See the repository’s Actions runs for reproducible logs tied to commits.
