# Asterism Architecture (v1)

This document summarizes the current platform direction. Repository-wide behavioral rules and contributor expectations live in `AGENTS.md` and `docs/engineering-standards.md`.

## Core Direction
- Monorepo for workload code, release artifacts, and deployment manifests.
- API-first services: every integration service owns its REST and async contracts.
- Polaris is the runtime shell and layout orchestrator for service-owned microfrontend modules.
- OpenShift-native deployment with Kustomize and GitOps pull from a separate GitOps repository.
- One deployed environment, promoted from `main`, consuming `latest` tags from successful `main` builds.

## Repository Layout
- `services/polaris`: host shell microfrontend canvas.
- `services/unifi`, `services/cluster`, `services/pfsense`: Go API services with event envelope scaffolding.
- `services/<service>/deploy`: Kubernetes deployment manifests (kustomization.yaml entrypoint + base/ directory).
- `deploy/kustomization.yaml`: Auto-generated consolidated kustomization (auto-discovered via `scripts/update-deploy.sh`).
- `deploy/platform/security`: Service Mesh mTLS and Cognito OIDC policy scaffolding.

## API And Event Contracts
Each service keeps:
- `api/openapi.yaml`
- `api/asyncapi.yaml`

This keeps contracts with implementation ownership and supports separate lifecycle per service.

## Microfrontend Model
- Polaris loads service registry data at runtime.
- Services may expose service-owned UI resources and module manifests.
- UI actions should interact through documented service APIs rather than bypassing backend contracts.

## Initial Integration Probes
- `unifi`: probes `UNIFI_API_URL` with optional `UNIFI_API_TOKEN`.
- `cluster`: probes `CLUSTER_API_URL` using in-cluster service account token and CA.
- `pfsense`: probes `PFSENSE_API_URL` with optional `PFSENSE_API_TOKEN`.
- Probe output is returned in `/api/v1/status.integration` for dashboard consumption.

## Security Model
- External user authentication is delegated through Cognito OIDC.
- External user authorization is enforced in application code.
- Internal service-to-service authentication and authorization are delegated to the service mesh using mTLS and mesh policy.
- Secret material is sourced from AWS Secrets Manager through External Secrets Operator.
- Protected service endpoints (`/api/v1/status`) require identity context by default (`AUTH_MODE=enforced`) and read forwarded principal/group headers (`X-Asterism-Principal`, `X-Asterism-Groups`).

## CI/CD And Supply Chain
GitHub Actions pipeline includes:
- service discovery matrix for Node and Go services,
- test, lint, and build verification per service,
- security scanning and auditable outputs,
- PR-built container archives, release-time GHCR push for immutable `vX.Y.Z` and moving `latest` tags,
- keyless image signing and SBOM generation,
- release automation in the GitHub Actions `release` environment with environment-scoped release credentials,
- GitHub Release assets including rendered Kustomize manifests and machine-readable release metadata.

## GitOps Flow
1. CI validates pull requests and uploads release-source image archives, SBOMs, metadata, and rendered manifests.
2. Release reuses the successful PR artifacts for the exact reviewed head SHA, publishes immutable and `latest` image tags, signs digests, and creates the release manifest.
3. Release automation commits the new Asterism release ref and rollout annotations to the separate GitOps repository.
4. OpenShift GitOps reconciles the deployed environment from the GitOps repository.
5. Release verification polls Argo CD until the application is synced, healthy, rolled out, and running the expected image digests.
