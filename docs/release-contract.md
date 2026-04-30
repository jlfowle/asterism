# Release Contract

Each GitHub release includes:
- `release-manifest.json`
- `*.yaml` Kustomize rendered manifests
- `sbom-*.spdx.json`
- `image-metadata-*.json`

The shipped image bytes and rendered manifests are reused from the reviewed PR build. Release publication loads those PR-built image archives, pushes immutable `vX.Y.Z` tags and the moving `latest` deployment tags, signs each immutable image digest, and fails if any expected service is missing an image archive, SBOM, digest, or metadata entry.

After image publication succeeds, release automation commits the matching Asterism release ref to the separate GitOps repository and adds pod-template annotations that force a rollout while deployments continue to reference `:latest`. Argo CD verification must confirm the app is synced and healthy and that running pod image IDs match the release manifest digests.

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
- GitHub App credentials for the private GitOps repository:
  - `CD_REPO_GITHUB_APP_ID`
  - `CD_REPO_GITHUB_APP_PRIVATE_KEY`
- Argo CD verification credential:
  - `ARGOCD_AUTH_TOKEN`
- Repository variables:
  - `CD_REPO_FULL_NAME`, for example `jlfowle/os-config`
  - `ARGOCD_SERVER`
  - `ARGOCD_APP_NAME`
  - `ARGOCD_PROJECT`
  - `ARGOCD_APP_NAMESPACE` when the namespace cannot be inferred from the Argo CD application
