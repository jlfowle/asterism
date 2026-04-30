#!/usr/bin/env python3

import argparse
import re
from pathlib import Path


def parse_args():
    parser = argparse.ArgumentParser(
        description="Update the os-config Asterism overlay for a released Asterism version."
    )
    parser.add_argument("--repo-dir", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--source-repo", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--manifest-sha256", required=True)
    return parser.parse_args()


def read_namespace(path):
    if not path.exists():
        return "app-asterism"

    match = re.search(r"(?m)^namespace:\s*([^\s#]+)", path.read_text(encoding="utf-8"))
    return match.group(1) if match else "app-asterism"


def build_kustomization(namespace, source_repo, version, commit, manifest_sha256):
    resource_ref = f"github.com/{source_repo}//deploy?ref={version}"
    return f"""apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: {namespace}

resources:
  - {resource_ref}

labels:
  - pairs:
      app: asterism

patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      labelSelector: app.kubernetes.io/part-of=asterism
    patch: |-
      - op: add
        path: /spec/template/metadata/annotations/asterism.dev~1release-version
        value: "{version}"
      - op: add
        path: /spec/template/metadata/annotations/asterism.dev~1release-commit
        value: "{commit}"
      - op: add
        path: /spec/template/metadata/annotations/asterism.dev~1release-manifest-sha256
        value: "{manifest_sha256}"
"""


def main():
    args = parse_args()
    overlay_path = Path(args.repo_dir) / "app" / "asterism" / "kustomization.yaml"

    if not overlay_path.parent.is_dir():
        raise SystemExit(f"Asterism overlay directory does not exist: {overlay_path.parent}")

    namespace = read_namespace(overlay_path)
    rendered = build_kustomization(
        namespace=namespace,
        source_repo=args.source_repo,
        version=args.version,
        commit=args.commit,
        manifest_sha256=args.manifest_sha256,
    )

    if overlay_path.exists() and overlay_path.read_text(encoding="utf-8") == rendered:
        print(f"{overlay_path} already points at {args.version}.")
        return

    overlay_path.write_text(rendered, encoding="utf-8")
    print(f"Updated {overlay_path} to {args.version}.")


if __name__ == "__main__":
    main()
