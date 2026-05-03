# Release Contract

Each GitHub release includes:
- `release-manifest.json`
- `*.yaml` Kustomize rendered manifests
- `sbom-*.spdx.json`
- `image-metadata-*.json`

The shipped image bytes and rendered manifests are reused from the reviewed PR build. Release publication loads those PR-built image archives, pushes immutable `vX.Y.Z` tags and the moving `latest` deployment tags, signs each immutable image digest, and fails if any expected service is missing an image archive, SBOM, digest, or metadata entry.

Release reruns are idempotent for a merged commit that already has a published release manifest: the workflow reuses that manifest and version instead of minting a second release.

After image publication succeeds, release automation commits the matching Asterism release ref to the separate GitOps repository and adds pod-template annotations that force a rollout while deployments continue to reference `:latest`. Argo CD verification must confirm the app is synced and healthy and that running pod image IDs match the release manifest digests.

The release workflow runs in the GitHub Actions `release` environment. Store release-only credentials there so they are only exposed to the release job.

## release-manifest.json
```json
{
  "version": "vX.Y.Z",
  "commit": "<git-sha>",
  "created": "<UTC timestamp>",
  "sourceCiRunId": "<workflow-run-id>",
  "sourceCiRunUrl": "<workflow-run-url>",
  "services": [
    {
      "service": "polaris",
      "image": "ghcr.io/<owner>/<repo>-polaris",
      "digest": "sha256:...",
      "version": "vX.Y.Z",
      "releaseTag": "vX.Y.Z",
      "latestTag": "latest",
      "sourceTag": "vA.B.C-pr.<number>-commit.<short-sha>"
    }
  ]
}
```

Consumers (GitOps automation, audit jobs, or promotion tooling) can parse this file without scraping release notes.

## Required Release Configuration
- GitHub App credentials used by the release workflow to update the private GitOps repository (`os-config`):
  - Environment secret: `CD_REPO_GITHUB_APP_PRIVATE_KEY`
  - Repository variable: `CD_REPO_GITHUB_APP_ID`
- Argo CD verification credential:
  - Environment secret: `ARGOCD_AUTH_TOKEN`
  - The token user must have Argo CD read and sync access to the `apps` project so the verifier can fetch application status, resource trees, and request a sync when needed.
  - At minimum, the local Argo CD account should be granted `applications, get` and `applications, sync` on `apps/*`.
- Repository variables:
  - `CD_REPO_FULL_NAME`, for example `jlfowle/os-config`
  - `ARGOCD_SERVER`
  - `ARGOCD_APP_NAME`
- Optional repository variables:
  - `ARGOCD_PROJECT`
  - `ARGOCD_APP_NAMESPACE` when the namespace cannot be inferred from the Argo CD application

This GitHub App is for the release workflow's promotion commit into `os-config`; it is separate from any Argo CD repository connectivity or human SSO setup.

For Argo CD instances that use OpenShift OAuth for human access, keep the SSO configuration separate from automation access. Define a local Argo CD user with `apiKey: true` and `login: false` in the Argo CD custom resource, then read the generated `{username}-local-user` secret and store its `apiToken` value in `ARGOCD_AUTH_TOKEN` for the `release` environment.
