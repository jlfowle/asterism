#!/usr/bin/env python3

import argparse
from pathlib import Path


def parse_args():
    parser = argparse.ArgumentParser(
        description="Update the os-config Asterism overlay for a released Asterism version."
    )
    parser.add_argument("--repo-dir", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--source-repo", required=True)
    parser.add_argument("--asset-name", default="asterism-deploy.yaml")
    return parser.parse_args()


def build_kustomization(source_repo, version, asset_name):
    asset_url = f"https://github.com/{source_repo}/releases/download/{version}/{asset_name}"
    return f"""apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - {asset_url}
"""


def main():
    args = parse_args()
    overlay_path = Path(args.repo_dir) / "app" / "asterism" / "kustomization.yaml"

    if not overlay_path.parent.is_dir():
        raise SystemExit(f"Asterism overlay directory does not exist: {overlay_path.parent}")

    rendered = build_kustomization(
        source_repo=args.source_repo,
        version=args.version,
        asset_name=args.asset_name,
    )

    if overlay_path.exists() and overlay_path.read_text(encoding="utf-8") == rendered:
        print(f"{overlay_path} already points at {args.version}.")
        return

    overlay_path.write_text(rendered, encoding="utf-8")
    print(f"Updated {overlay_path} to {args.version}.")


if __name__ == "__main__":
    main()
