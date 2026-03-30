# Asterism Repository Guidance

This file is the fast-start guidance for any AI agent or contributor working in this repository. The durable source of truth is [docs/engineering-standards.md](/workspaces/asterism/docs/engineering-standards.md).

## Mission
- Build Asterism as an OpenShift-native home control platform.
- Use a microservices backend with a runtime-composed microfrontend shell.
- Keep the repository production-minded and enterprise-grade in its security and compliance posture.

## Architecture
- `services/polaris` is the microfrontend canvas and orchestration shell.
- Integration services own their APIs, events, and service-owned UI resources.
- UI behavior must flow through service APIs. Do not add UI-only side channels that bypass service contracts.
- Prefer OpenShift-native patterns, APIs, and deployment constructs when there is a reasonable choice.

## Security Model
- External user authentication is delegated through OpenID Connect with AWS Cognito.
- External user authorization is enforced in the application.
- Internal service-to-service authentication and authorization are delegated to the service mesh with mTLS and mesh policy.
- Sensitive runtime values must come from External Secrets Operator backed by AWS Secrets Manager.
- Do not commit secrets, static credentials, or environment-specific confidential material to the repository.

## API And Contract Rules
- API-first is mandatory.
- Each service owns its REST contract in `api/openapi.yaml`.
- Each service owns its async/event contract in `api/asyncapi.yaml`.
- Changes to user-visible behavior should be reflected in contracts and tests.

## Delivery Rules
- CI runs in GitHub Actions.
- Pull requests must build, lint, and test to an enterprise-minded security and compliance standard.
- CD is handled through OpenShift GitOps with Kustomize from a separate Argo CD repository.
- The only deployed environment is driven from `main`.
- Deployment consumes `latest` tags from successful `main` builds, while releases should still preserve traceability metadata.

## Working Norms
- Favor declarative configuration over manual cluster mutation.
- Keep services independently buildable and testable.
- Prefer changes that improve auditability, provenance, least privilege, and operational clarity.
- When in doubt, update the standards docs rather than leaving expectations implicit in chat history.
- Prefer reasonable-sized, reviewable commits that group one coherent change or a small related set of changes. Avoid letting large unrelated work accumulate in one local batch.
