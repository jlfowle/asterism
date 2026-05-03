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
- Pull request titles must use Conventional Commits format, because PR title linting is enforced in GitHub Actions.
- Pull request branches should use signed commits so GitHub can verify the change origin.
- Asterism repository changes should move through pull requests and merge review; the only direct push in the release flow belongs to the separate GitOps repository.
- For any Asterism code or config change, default to the full GitHub workflow: create a dedicated branch, make signed commits on that branch, open a PR, and keep watching CI checks plus review threads until the PR is ready to merge or the user asks you to stop.
- CD is handled through OpenShift GitOps with Kustomize from a separate Argo CD repository.
- Release automation runs in the GitHub Actions `release` environment; keep release-only credentials there.
- Use a GitHub App installation token for the cross-repo promotion commit into `os-config` instead of a long-lived PAT.
- Argo CD automation should use the `asterism-release` token-only local user configured in the GitOps repo, not the human SSO path.
- That Argo CD token still needs read access to the `apps` project so the release job can fetch application status and resource trees.
- The only deployed environment is driven from `main`.
- Deployment consumes `latest` tags from successful `main` builds, while releases should still preserve traceability metadata.

## Agent Decision Governance
- Do not make major architectural, security, deployment, branch-protection, or validation-policy decisions unilaterally.
- For decisions with non-obvious tradeoffs, present options, risks, and a recommendation, then wait for user direction before proceeding.
- If a decision changes enforcement level (for example, required checks or policy gates), explicit user approval is required.

## Validation And Automation Rules
- Do not bypass, disable, or weaken validation gates to make a failing workflow pass.
- Do not exempt specific actors (including `dependabot[bot]`) from required validation checks unless the user explicitly approves that policy change.
- Fix root causes in code or automation so all contributors and bots can satisfy the same standards.
- Treat reductions in auditability, traceability, or policy enforcement as regressions.

## Working Norms
- Favor declarative configuration over manual cluster mutation.
- Keep services independently buildable and testable.
- Prefer changes that improve auditability, provenance, least privilege, and operational clarity.
- When in doubt, update the standards docs rather than leaving expectations implicit in chat history.
- Prefer reasonable-sized, reviewable commits that group one coherent change or a small related set of changes. Avoid letting large unrelated work accumulate in one local batch.
