# Authn/Authz Rollout Verification

This checklist is the working verification plan for the OpenShift OAuth edge proxy and OPA-backed service authorization rollout.

## Pre-merge Checks

- [ ] `make test`
- [ ] `kustomize build deploy`
- [ ] PR checks are green: `Lint PR`, `CI`, and `PR Merge Readiness`
- [ ] The Polaris build still emits `microfrontends.json`

## Public Edge Checks

- [ ] Unauthenticated `GET /` reaches the OpenShift login gate through oauth-proxy
- [ ] Authenticated `GET /` renders the Polaris shell
- [ ] Authenticated `GET /` does not return a 503 or upstream reset from oauth-proxy
- [ ] Sign-out returns through the OpenShift OAuth logout path
- [ ] Public requests use `GET`, not `HEAD`, when checking the routed edge behavior
- [ ] `asterism-internal.apps.os.fowler.house` serves the Polaris shell through the same auth-proxy front door
- [ ] oauth-proxy forwards the shell service Host header upstream, not the public browser host

## Service Authorization Checks

- [ ] `GET /api/services/{service}/api/v1/status` succeeds for an authenticated principal through the Polaris proxy
- [ ] `GET /api/services/{service}/api/v1/summary` succeeds for an authenticated principal through the Polaris proxy
- [ ] `GET /api/services/{service}/api/v1/actions` stays read-only and returns disabled contracts through the Polaris proxy
- [ ] Spoofed `X-Asterism-Principal` headers are ignored
- [ ] The service still accepts proxy-provided `X-Forwarded-User` and `X-Forwarded-Email` headers
- [ ] When `AUTHZ_OPA_URL` is set, OPA denial returns `403`

## GitOps And Release Checks

- [ ] The GitOps repository reports the app as Synced and Healthy
- [ ] The running pod image digests match the release manifest
- [ ] The release asset still renders with pod-template annotations and digest-pinned image references
- [ ] The `polaris` Service resolves only the Polaris shell pod and not the auth-proxy pod

## Notes

- Keep the tracker in `/workspace/docs/asterism-authn-authz-stabilization-tracker.md` up to date as each branch lands.
- Check the current and previous PRs for comments before starting the next branch.
