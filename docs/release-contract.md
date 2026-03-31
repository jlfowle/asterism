# Release Contract

Each GitHub release includes:
- `release-manifest.json`
- `*-latest.yaml` Kustomize rendered manifests
- `sbom-*.spdx.json`
- `image-metadata-*.json`

## release-manifest.json
```json
{
  "version": "vX.Y.Z",
  "commit": "<git-sha>",
  "created": "<UTC timestamp>",
  "services": [
    {
      "service": "polaris",
      "image": "ghcr.io/<owner>/<repo>-polaris",
      "digest": "sha256:...",
      "version": "vX.Y.Z"
    }
  ]
}
```

Consumers (GitOps automation, audit jobs, or promotion tooling) can parse this file without scraping release notes.
