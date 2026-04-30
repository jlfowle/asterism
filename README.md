# Asterism

Home control center platform using microservices (Go), a runtime-composed microfrontend shell (`polaris`), and OpenShift GitOps deployment.

## Repository Standards
- Contributor and AI quick-start guidance: `AGENTS.md`
- Durable engineering standards: `docs/engineering-standards.md`
- Architecture overview: `docs/architecture.md`
- Release artifact contract: `docs/release-contract.md`
- Developer workflow guide: `docs/development.md`

## Services
- `polaris`: host shell UI and runtime microfrontend registry client.
- `unifi`: UniFi integration service scaffold.
- `cluster`: OpenShift integration service scaffold.
- `pfsense`: pfSense integration service scaffold.

## Local Development
1. `make dev`
2. Open `http://localhost:3000`
3. Use `make test`, `make lint`, `make build`, and `make render-deploy` for validation

Ports used by the local stack:

- Polaris: `3000`
- UniFi: `8081`
- Cluster: `8082`
- pfSense: `8083`

## Contracts and Deploy
- API contracts: `services/*/api/openapi.yaml`
- Event contracts: `services/*/api/asyncapi.yaml`
- Deployment manifests: `services/*/deploy` (entrypoint: `kustomization.yaml`)
- Consolidated deploy (auto-generated): `deploy/kustomization.yaml`
- Platform security policy scaffolding: `deploy/platform/security`

## CI/CD
GitHub Actions in `.github/workflows/ci.yaml` provides:
- multi-service test/lint/build validation,
- PR-built image archives, SBOMs, and image metadata for each containerized service,
- rendered Kustomize release assets,
- release automation script checks.

GitHub Actions in `.github/workflows/release.yaml` provides:
- immutable release image tags and moving `latest` tags in GHCR,
- keyless image signing and machine-readable release metadata,
- automated GitOps promotion to the configured `os-config` repository,
- Argo CD sync, health, rollout, and image digest verification.

Pull request titles must use Conventional Commits format, for example `chore: cleanup developer workflow and deploy manifests`, because the PR title check enforces it.
Pull request branches should use signed commits. If you rewrite branch history, re-sign the new commits before pushing.

Main-branch pushes also refresh open PR branches through `.github/workflows/update-pr-branches.yaml`.
