# Asterism

Home control center platform using microservices (Go), a runtime-composed microfrontend shell (`polaris`), and OpenShift GitOps deployment.

## Repository Standards
- Contributor and AI quick-start guidance: `AGENTS.md`
- Durable engineering standards: `docs/engineering-standards.md`
- Architecture overview: `docs/architecture.md`
- Release artifact contract: `docs/release-contract.md`

## Services
- `polaris`: host shell UI and runtime microfrontend registry client.
- `unifi`: UniFi integration service scaffold.
- `cluster`: OpenShift integration service scaffold.
- `pfsense`: pfSense integration service scaffold.

## Local Development
1. `cd services/polaris && make run`
2. `cd services/unifi && make run`
3. `cd services/cluster && make run`
4. `cd services/pfsense && make run`

## Contracts and Deploy
- API contracts: `services/*/api/openapi.yaml`
- Event contracts: `services/*/api/asyncapi.yaml`
- Deployment manifests: `services/*/deploy` (entrypoint: `kustomization.yaml`)
- Consolidated deploy (auto-generated): `deploy/kustomization.yaml`
- Platform security policy scaffolding: `deploy/platform/security`

## CI/CD
GitHub Actions in `.github/workflows/ci.yaml` provides:
- multi-service test/lint/build validation,
- image build and GHCR publish,
- image signing, SLSA attestation, SBOM generation,
- rendered Kustomize release assets and machine-readable release manifest.

Main-branch pushes also refresh open PR branches through `.github/workflows/update-pr-branches.yaml`.
