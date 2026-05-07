# Asterism Architecture (v1)

This document summarizes the current platform direction. Repository-wide behavioral rules and contributor expectations live in `AGENTS.md` and `docs/engineering-standards.md`.

## Core Direction
- Monorepo for workload code, release artifacts, and deployment manifests.
- API-first services: every integration service owns its REST and async contracts.
- Polaris is the runtime shell and layout orchestrator for service-owned microfrontend modules.
- OpenShift-native deployment with Kustomize and GitOps pull from a separate GitOps repository.
- One deployed environment, promoted from `main`, consuming `latest` tags from successful `main` builds.
- Asterism is an operator cockpit rather than a replacement configuration authority for backend systems.

## Operator Cockpit Boundary
- Asterism may expose backend-owned status, plain-language diagnostics, authoritative links, audit-ready action contracts, and future automation entrypoints.
- Asterism must not duplicate broad UniFi, pfSense, OpenShift, or Argo CD configuration state.
- User-facing controls should be intent-based and narrowly scoped, for example "restart this approved access point" or "sync this Argo CD application", not arbitrary backend config editing.
- The first implementation milestone is observe-first: guided actions are advertised as disabled contracts until mutating paths are explicitly designed and approved.

## Repository Layout
- `services/polaris`: host shell microfrontend canvas.
- `services/unifi`, `services/cluster`, `services/pfsense`: Go API services with event envelope scaffolding.
- `services/<service>/deploy`: Kubernetes deployment manifests (kustomization.yaml entrypoint + base/ directory).
- `deploy/kustomization.yaml`: Auto-generated consolidated kustomization (auto-discovered via `scripts/update-deploy.sh`).
- `deploy/platform/routing`: Gateway API HTTPRoutes for the single Asterism public entry point.
- `deploy/platform/security`: Service Mesh mTLS, edge auth proxy, and OPA authorization policy scaffolding.

## API And Event Contracts
Each service keeps:
- `api/openapi.yaml`
- `api/asyncapi.yaml`

This keeps contracts with implementation ownership and supports separate lifecycle per service.

## Microfrontend Model
- Polaris loads service registry data at runtime.
- Services may expose service-owned UI resources and module manifests.
- UI actions should interact through documented service APIs rather than bypassing backend contracts.
- Public service APIs are routed under `/api/services/{service}/...`; service-owned UI assets are routed under `/ui/services/{service}/...`.

## Operator Summaries
- `unifi`: reads local UniFi Network API device and client status from `UNIFI_API_BASE_URL`, `UNIFI_SITE_ID`, and `UNIFI_API_TOKEN`.
- `cluster`: reads Kubernetes, OpenShift, and Argo CD status from `CLUSTER_API_URL` using a dedicated read-only service account.
- `pfsense`: reads pfSense system and interface status with read-only SNMP using `PFSENSE_SNMP_HOST` and `PFSENSE_SNMP_COMMUNITY`.
- Service output is returned in `/api/v1/status.integration`, mirrored by `/api/v1/summary`, and includes severity, degraded reasons, recommended actions, authoritative links, and disabled guided action contracts.

## Security Model
- External user authentication is delegated through the OpenShift OAuth edge proxy.
- Polaris sits behind that edge proxy and does not own browser bearer-token handling directly.
- External user authorization is enforced in application code and backed by OPA where the service has a shared policy decision endpoint.
- Service endpoints consume verified identity context from the edge proxy and should only trust proxy-provided principal headers.
- Internal service-to-service authentication and authorization are delegated to the service mesh using mTLS and mesh policy.
- Secret material is sourced from AWS Secrets Manager through External Secrets Operator.
- Protected service endpoints (`/api/v1/status`) require identity context by default (`AUTH_MODE=enforced`) and can call a shared OPA decision service via `AUTHZ_OPA_URL`.
- External traffic enters through `https://asterism.apps.os.fowler.house/`, lands on the OpenShift OAuth front door, and then reaches Polaris and downstream service APIs through the mesh ingress path rather than direct per-service OpenShift Routes.

## CI/CD And Supply Chain
GitHub Actions pipeline includes:
- service discovery matrix for Node and Go services,
- test, lint, and build verification per service,
- security scanning and auditable outputs,
- PR-built container archives, release-time GHCR push for immutable `vX.Y.Z` and moving `latest` tags,
- keyless image signing and SBOM generation,
- release automation in the GitHub Actions `release` environment with environment-scoped release credentials,
- GitHub Release assets including a deployable `asterism-deploy.yaml` manifest and machine-readable release metadata.

## GitOps Flow
1. CI validates pull requests and uploads release-source image archives, SBOMs, metadata, and rendered manifests.
2. Release reuses the successful PR artifacts for the exact reviewed head SHA, publishes immutable and `latest` image tags, signs digests, creates the release manifest, and renders `asterism-deploy.yaml` with `vX.Y.Z@sha256:...` image refs.
3. Release automation commits the GitOps pointer to the GitHub release asset in the separate deployment repository.
4. OpenShift GitOps reconciles the deployed environment from the GitOps repository.
5. Release verification polls Argo CD until the application is synced, healthy, rolled out, and running the expected image digests.
