# Deployment Manifest Standards

This document defines the standard layout, requirements, and discovery contract for Kubernetes deployment manifests in Asterism.

## Directory Layout

### Per-Service Structure

Each deployable service owns its Kubernetes manifests under `services/{service}/deploy/`:

```
services/{service}/
├── deploy/
│   ├── kustomization.yaml          # Contract entrypoint for the consolidated deploy
│   └── base/
│       ├── deployment.yaml
│       └── service.yaml            # Internal ClusterIP service
├── Dockerfile                       # Required for service to be deployable
├── Makefile                         # Required for service to be deployable
└── [other service files]
```

**Key Points:**
- `services/{service}/deploy/kustomization.yaml` is the **contract entrypoint** — it is what the consolidated deploy references
- `services/{service}/deploy/kustomization.yaml` is **not nested** in a subdirectory; it is a direct child of `deploy/`
- `services/{service}/deploy/base/` contains the service manifest files (no kustomization.yaml here)

### Consolidated Deploy Structure

The root-level `deploy/` directory contains the consolidated kustomization and cross-cutting concerns:

```
deploy/
├── kustomization.yaml              # AUTO-GENERATED; references all services and platform policies
├── platform/
│   ├── routing/                    # Gateway API HTTPRoutes for public app routing
│   └── security/                   # Cross-cutting Istio and OIDC policies
│       ├── authorization-policy.yaml
│       ├── request-authentication.yaml
│       ├── peer-authentication.yaml
│       └── kustomization.yaml
└── [other platform-level resources]
```

**Key Points:**
- `deploy/kustomization.yaml` is **auto-generated** by `scripts/update-deploy.sh`; do not edit it manually
- `deploy/platform/routing/` contains Asterism's Gateway API `HTTPRoute` resources for the single public entry point
- `deploy/platform/security/` contains infrastructure cross-cutting concerns, not service-owned
- The consolidation is a simple aggregation via kustomize resource references

## Service Deploy Manifest Requirements

### Mandatory Manifests

Each service's `deploy/base/` directory must contain the standard service
manifests. `externalsecret.yaml` is only present when a service has a real
runtime secret dependency.

#### 1. `deployment.yaml`
- Defines the Kubernetes Deployment for the service
- Container image is injected by kustomize (via `images:` configuration in `kustomization.yaml`)
- Port 8080 (application port)
- Required environment variables:
  - `PORT=8080`
  - `AUTH_MODE=enforced`
  - Service-specific environment variables (e.g., `CLUSTER_API_URL` for services that need it)
- Security context: non-root user, read-only root filesystem, no Linux capabilities
- Istio sidecar injection enabled (via label `sidecar.istio.io/inject: "true"`)

#### 2. `service.yaml`
- Kubernetes Service (ClusterIP) exposing the Deployment
- Port 80 (external/cluster port) → 8080 (container port)
- Selector matches the deployment: `app.kubernetes.io/name: {service}`
- No external traffic exposure from this manifest. Public traffic is attached through Gateway API routing in `deploy/platform/routing`.

#### 3. `externalsecret.yaml`
- Optional ExternalSecrets Operator CRD
- Synchronizes secrets from AWS Secrets Manager (`ClusterSecretStore: aws-secretsmanager`)
- Source key in Secrets Manager: `/asterism/{service}`
- Target secret name: `{service}-secrets`
- Auto-refresh interval: 1 hour
- Include this file only when the service has sensitive runtime data that must
  be delivered through the secret manager

### Public Routing Standard

Asterism uses a single public host, `asterism.apps.os.fowler.house`, and Kubernetes Gateway API `HTTPRoute` resources. The shared `Gateway` and OpenShift exposure live in the GitOps `os-config` repository; this repository owns the app path routing.

Service-owned surfaces are:
- Internal API: `/api/v1/*`
- Internal UI assets: `/ui/*`
- Public API: `/api/services/{service}/api/v1/*`
- Public UI: `/ui/services/{service}/*`

Do not add per-service OpenShift `Route` resources for Asterism workloads. Direct routes bypass the intended mesh ingress path and can conflict with `STRICT` mTLS.

### Kustomization Configuration

Each service's `services/{service}/deploy/kustomization.yaml` must set:

#### Namespace
```yaml
namespace: asterism
```

#### Resources
References the required manifest files in relative paths:
```yaml
resources:
  - ./base/deployment.yaml
  - ./base/service.yaml
```

#### Image Configuration
Patches the container image tag at build time:
```yaml
images:
  - name: {service}
    newName: ghcr.io/jlfowle/asterism-{service}
    newTag: latest
```

Replace `{service}` with the actual service name (e.g., `cluster`, `polaris`, `unifi`).

#### Labels
Applied to all resources:
```yaml
labels:
  - pairs:
      app.kubernetes.io/name: {service}
      app.kubernetes.io/part-of: asterism
    includeSelectors: true
    includeTemplates: true
```

## Service Discovery Contract

A service is considered **deployable** if and only if it meets all three criteria:

1. **Build Artifact:** `services/{service}/Dockerfile` exists
   - Indicates the service is containerized

2. **Build Logic:** `services/{service}/Makefile` exists
   - Indicates the service has a build process

3. **Deployment Ready:** `services/{service}/deploy/kustomization.yaml` exists
   - Indicates the service has deployment manifests following this standard

**Why Auto-Discovery?**
The list of deployable services is **not manually maintained**. Instead, `scripts/update-deploy.sh` scans `services/*/` and discovers all services meeting the above criteria, then auto-generates `deploy/kustomization.yaml` to reference them.

**Implication: Adding a New Service**
When you add a new service to the repository:
1. Create `services/{new-service}/`
2. Add `Dockerfile`, `Makefile`, and other build files
3. Create `services/{new-service}/deploy/kustomization.yaml` and `services/{new-service}/deploy/base/` with the service manifests required for that service
4. Run `scripts/update-deploy.sh` to regenerate the consolidated `deploy/kustomization.yaml`
5. Commit both the new service structure and the updated `deploy/kustomization.yaml`

## Consolidated Deploy Expectations

### Generated `deploy/kustomization.yaml`

This file is **auto-generated** by `scripts/update-deploy.sh`. It aggregates all discovered services and platform policies:

```yaml
# AUTO-GENERATED. Run: scripts/update-deploy.sh to regenerate
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../services/cluster/deploy
  - ../services/pfsense/deploy
  - ../services/polaris/deploy
  - ../services/unifi/deploy
  - platform/routing
  - platform/security
```

**Key Properties:**
- No `namespace:` set; each service's kustomization declares its own
- No `images:` or `labels:` set; inherited from service kustomizations
- Simple reference structure; easy to verify and audit
- **Do not edit this file manually**; regenerate via `scripts/update-deploy.sh`

### Auto-Generation Workflow

**When to run `scripts/update-deploy.sh`:**
- Before committing when you add/remove a service directory
- Automatically in CI to validate consistency (see CI workflow)

**What the script does:**
1. Scans `services/*/` for deployable services (checks for Dockerfile, Makefile, deploy/kustomization.yaml)
2. Generates `deploy/kustomization.yaml` with `resources:` entries for each discovered service
3. Adds `platform/routing` and `platform/security` references for cross-cutting policies
4. Writes the file with auto-generated header comment
5. Exits with code 0 (always succeeds; reports discoveries as informational)

**Manual Execution:**
```bash
./scripts/update-deploy.sh
```

**CI Validation:**
The CI workflow includes a validation job that:
- Runs `scripts/update-deploy.sh`
- Compares the output to the current `deploy/kustomization.yaml`
- Fails the job if they differ, with a helpful error message

## Building and Verifying

### Build a Single Service

```bash
cd services/{service}
kustomize build deploy
```

Should produce valid Kubernetes manifests with all labels, images, and namespace correctly applied.

### Build the Consolidated Deploy

```bash
cd deploy
kustomize build .
```

Should produce manifests for all services plus platform policies and HTTPRoutes, all in the `asterism` namespace with appropriate labels.

### Validate Service Discovery

```bash
./scripts/update-deploy.sh --dry-run  # If dry-run mode is implemented
# Or just:
./scripts/update-deploy.sh
git diff deploy/kustomization.yaml
```

If `git diff` shows no changes, the discovery is up to date. If changes exist, commit them.

## Common Tasks

### Adding a New Service

1. Create `services/{my-service}/` with Dockerfile, Makefile, and build configuration
2. Create `services/{my-service}/deploy/base/` with the service manifests required for that service, and add `externalsecret.yaml` only if the service needs it
3. Create `services/{my-service}/deploy/kustomization.yaml` following the template above
4. Run `./scripts/update-deploy.sh`
5. Verify `kustomize build services/{my-service}/deploy` succeeds
6. Commit changes including the updated `deploy/kustomization.yaml`

### Removing a Service

1. Delete or move the service directory (e.g., `services/{old-service}/`)
2. Run `./scripts/update-deploy.sh` to remove its reference from the consolidated deploy
3. Commit the updated `deploy/kustomization.yaml`

### Updating a Service's Manifests

1. Edit files in `services/{service}/deploy/base/`
2. Optionally update `services/{service}/deploy/kustomization.yaml` for configuration changes
3. Verify: `kustomize build services/{service}/deploy`
4. No need to run `scripts/update-deploy.sh` unless the service's discovery criteria changed

### Updating Platform Policies

1. Edit files in `deploy/platform/security/` or add new cross-cutting manifests
2. Update `deploy/platform/security/kustomization.yaml` if you add new files
3. Verify: `kustomize build deploy` includes the policy changes
4. No need to run `scripts/update-deploy.sh`; platform policies are not auto-discovered

## References

- [Kustomize Documentation](https://kustomize.io/)
- [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/)
- [ExternalSecrets Operator](https://external-secrets.io/)
- [Istio Injection](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/)
