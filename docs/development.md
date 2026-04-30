# Development Workflow

This repository is set up so a new contributor can get from checkout to a
working local platform quickly.

## Fast Start

1. Open the repo in the devcontainer if you have it available.
2. Run `make dev` from the repository root.
3. Open `http://localhost:3000` for Polaris.

Before opening a PR, make sure commit signing is configured on your machine. This repository expects signed commits on pull request branches so GitHub can verify the change origin. If you already have an SSH signing key loaded, `ssh-add -L` should show the public key you can register with GitHub and point Git at for signing.

`make dev` starts the full local stack:

- Polaris on `3000`
- UniFi on `8081`
- Cluster on `8082`
- pfSense on `8083`

The local run mode disables application auth and leaves integration endpoints
unset unless you override them, which keeps the stack usable without cloud or
home-network credentials.

## Common Commands

- `make test` runs the service test suite for all services.
- `make lint` runs the service linters for all services.
- `make build` builds every service.
- `make render-deploy` regenerates and validates the consolidated Kustomize
  output.
- `make clean` removes generated build output for every service.

## Commit Signing

GitHub verifies signed commits on pull request branches, so configure your local Git client before you start work:

1. Ensure your SSH signing key is loaded in `ssh-agent`.
2. Use `ssh-add -L` to confirm which public key GitHub should trust.
3. Add that key in GitHub as an SSH signing key.
4. Set Git to sign commits by default with SSH for this repository or globally.

Typical Git settings are `git config --global gpg.format ssh` and `git config --global commit.gpgsign true`.

## Service Layout

Each service owns its implementation, contracts, and deploy manifests:

- `services/{service}/api/openapi.yaml`
- `services/{service}/api/asyncapi.yaml`
- `services/{service}/deploy/kustomization.yaml`

## Adding Or Updating A Service

1. Make the code change in the service directory.
2. Update the service API contract if user-visible behavior changes.
3. Update the service deployment manifest if runtime config changes.
4. Run `make test`, `make lint`, and `make build` for the service.
5. Run `make render-deploy` if the service inventory or deployment manifests
   changed.

## Troubleshooting

- If Polaris loads but cards do not render, verify the service is running on the
  expected port and that the proxy path in `services/polaris/webpack.config.js`
  still matches the service route.
- If a Go service exits immediately in local development, check that
  `AUTH_MODE=disabled` is set and that any optional integration URL variables are
  blank.
