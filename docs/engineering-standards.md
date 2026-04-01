# Engineering Standards

These standards apply to both human contributors and AI agents working in Asterism. They define the expected architectural, security, and delivery posture for this repository.

## 1. Platform Direction
- Asterism is an OpenShift-native platform and should prefer OpenShift-compatible patterns, APIs, and operational models.
- The repository is a monorepo for workload code, contracts, deployment artifacts, and release metadata.
- The deployed runtime model is microservices plus a runtime-composed microfrontend shell.

## 2. Application Architecture
- `polaris` is the microfrontend canvas and host shell.
- Domain and integration capabilities belong in services, not in Polaris.
- Services may contribute UI resources, but Polaris remains the composition layer that discovers and renders them.
- Frontend behavior should consume backend APIs rather than re-implement service logic in the browser.

## 3. API-First Contract Model
- API-first is mandatory for every service.
- Each service owns:
  - `api/openapi.yaml` for REST interfaces.
  - `api/asyncapi.yaml` for async/event interfaces.
- Web and UI actions should interact through documented APIs and events.
- Contract changes should be made intentionally and validated alongside implementation and tests.

## 4. Authentication And Authorization

### External traffic
- User authentication is delegated through OpenID Connect with AWS Cognito.
- User authorization is handled in the application.
- Service endpoints should consume verified identity context and enforce authorization decisions in service code where user-facing access is concerned.

### Internal traffic
- Service-to-service authentication and authorization are delegated to the service mesh.
- Mutual TLS is the default trust model for internal communication.
- Mesh policy should be the first line of defense for internal access control.

## 5. Secrets And Sensitive Configuration
- Sensitive deployed values must be sourced through External Secrets Operator.
- AWS Secrets Manager is the backing secret system.
- Secrets must not be committed to the repository.
- CI should avoid printing or persisting sensitive values in logs or artifacts.
- Environment-specific confidential values belong in secret management and deployment configuration, not in application source.

## 6. OpenShift And Deployment Model
- Deployment is GitOps-driven through OpenShift GitOps using Kustomize.
- The Argo CD `Application` definitions live in a separate repository and are intentionally not managed here.
- Manifests in this repository should remain GitOps-friendly and declarative.
- Prefer OpenShift-native constructs when choosing between equivalent deployment options.

## 7. Environment Strategy
- There is only one deployed environment.
- That environment is promoted from the `main` branch only.
- Deployment consumes `latest` tags produced from successful `main` builds.
- Even with a `latest` deployment strategy, releases should preserve traceability through versioning, digests, provenance, and SBOM metadata.

## 8. CI Expectations
- CI is implemented in GitHub Actions.
- Pull requests must at minimum build, lint, and test the changed services.
- Pull request automation should expose a single aggregate merge-readiness check that is suitable for branch protection as the only required validation gate.
- CI should reflect the standards of a security-focused and compliance-focused enterprise.
- Security scanning, artifact integrity, provenance, SBOM generation, and auditable release outputs are preferred defaults rather than optional extras.
- Changes that reduce traceability, weaken controls, or bypass validation should be treated as regressions.

## 9. Delivery And Release Expectations
- Build outputs should be reproducible and traceable.
- Container artifacts should preserve supply chain metadata such as digests, signatures, attestations, and SBOMs where supported.
- Release artifacts should be machine-readable where practical to support audit, promotion, and GitOps automation.

## 10. Repository Working Agreements
- Prefer declarative automation over manual operational steps.
- Keep service boundaries clear: a service owns its contracts, implementation, and service-owned UI assets.
- Document durable architectural decisions in the repository instead of relying on chat memory.
- When a new pattern becomes important, update this file and any related architecture docs in the same change.
- Prefer reasonable-sized, reviewable commits that capture one coherent change or a small related set of changes. Avoid bundling large unrelated batches when the work can be checkpointed safely.

## 11. Default Assumptions For Contributors
- If a choice affects security, favor least privilege and stronger verification.
- If a choice affects deployment, favor OpenShift-native GitOps workflows.
- If a choice affects integration, prefer explicit API contracts over implicit coupling.
- If a choice affects runtime secrets, use External Secrets Operator with AWS Secrets Manager.
- If a choice affects internal trust, assume service mesh mTLS and mesh policy are the baseline.

## 12. Validation Integrity
- Required validation gates must remain enforced for all contributors, including automation and bots.
- Do not bypass, disable, or weaken checks as a workaround for failing automation.
- When automation (for example Dependabot) cannot satisfy a validation, fix the underlying workflow or automation behavior so policy remains intact.
- Any intentional reduction in validation enforcement requires explicit approval from the repository owner and should be documented in-repo with rationale.

## 13. Decision Governance
- Major decisions (architecture, security posture, deployment model, branch protection, required checks, and policy gate changes) require explicit user or maintainer direction.
- Contributors and AI agents should present tradeoffs and recommendations, then pause for approval before implementing major policy shifts.
- If there is uncertainty about whether a change is a major decision, treat it as major and escalate.
