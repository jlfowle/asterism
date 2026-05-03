#!/usr/bin/env python3

import argparse
import json
import sys
import time
import urllib.error
import urllib.parse
import urllib.request


def parse_args():
    parser = argparse.ArgumentParser(
        description="Verify Argo CD has deployed the released Asterism image digests."
    )
    parser.add_argument("--server", required=True)
    parser.add_argument("--token", required=True)
    parser.add_argument("--app", required=True)
    parser.add_argument("--project", default="")
    parser.add_argument("--namespace", default="")
    parser.add_argument("--release-manifest", required=True)
    parser.add_argument("--release-version", required=True)
    parser.add_argument("--release-commit", required=True)
    parser.add_argument("--manifest-sha256", required=True)
    parser.add_argument("--os-config-revision", required=True)
    parser.add_argument("--timeout-seconds", type=int, default=900)
    parser.add_argument("--poll-interval-seconds", type=int, default=20)
    return parser.parse_args()


class ArgoClient:
    def __init__(self, server, token):
        self.server = server.rstrip("/")
        self.token = token

    def request(self, method, path, params=None, body=None):
        url = f"{self.server}{path}"
        if params:
            query = urllib.parse.urlencode({k: v for k, v in params.items() if v is not None and v != ""})
            if query:
                url = f"{url}?{query}"

        data = None
        headers = {
            "Accept": "application/json",
            "Authorization": f"Bearer {self.token}",
        }

        if body is not None:
            data = json.dumps(body).encode("utf-8")
            headers["Content-Type"] = "application/json"

        request = urllib.request.Request(url, data=data, headers=headers, method=method)

        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                payload = response.read().decode("utf-8")
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"{method} {url} failed with HTTP {error.code}: {detail}") from error
        except urllib.error.URLError as error:
            raise RuntimeError(f"{method} {url} failed: {error}") from error

        if not payload:
            return {}
        return json.loads(payload)

    def get_application(self, app, project="", refresh=None):
        params = {"project": project}
        if refresh:
            params["refresh"] = refresh
        return self.request("GET", f"/api/v1/applications/{urllib.parse.quote(app, safe='')}", params=params)

    def resource_tree(self, app, project=""):
        return self.request(
            "GET",
            f"/api/v1/applications/{urllib.parse.quote(app, safe='')}/resource-tree",
            params={"project": project},
        )

    def resource(self, app, project, group, version, kind, namespace, name):
        params = {
            "project": project,
            "group": group,
            "version": version,
            "kind": kind,
            "namespace": namespace,
            "resourceName": name,
        }
        return self.request(
            "GET",
            f"/api/v1/applications/{urllib.parse.quote(app, safe='')}/resource",
            params=params,
        )


def load_manifest(path):
    with open(path, "r", encoding="utf-8") as handle:
        manifest = json.load(handle)

    services = manifest.get("services", [])
    if not services:
        raise SystemExit("Release manifest has no services; refusing to verify deployment.")

    expected = {}
    for service in services:
        name = service.get("service")
        digest = service.get("digest")
        image = service.get("image")
        if not name or not digest or not image:
            raise SystemExit(f"Invalid release manifest service entry: {service}")
        expected[name] = {"digest": digest, "image": image}
    return expected


def decode_resource_payload(payload):
    for key in ("manifest", "liveState"):
        value = payload.get(key)
        if isinstance(value, str) and value.strip():
            return json.loads(value)

    value = payload.get("resource")
    if isinstance(value, dict):
        return value

    if payload.get("kind") and payload.get("metadata"):
        return payload

    raise ValueError(f"Could not find a live resource payload in response keys: {sorted(payload.keys())}")


def app_revision(app):
    status = app.get("status", {})
    sync = status.get("sync", {})
    operation = status.get("operationState", {})
    sync_result = operation.get("syncResult", {})
    return sync.get("revision") or sync_result.get("revision") or ""


def app_has_revision(app, revision):
    if not revision:
        return False
    if app_revision(app) == revision:
        return True

    for entry in app.get("status", {}).get("history", []):
        if entry.get("revision") == revision:
            return True

    return False


def pod_digest_matches(pod, service, expected):
    statuses = pod.get("status", {}).get("containerStatuses", [])
    for status in statuses:
        image = status.get("image", "")
        image_id = status.get("imageID", "")
        name = status.get("name", "")
        if name != service and expected["image"] not in image:
            continue
        if expected["digest"] not in image_id:
            return False, f"{pod['metadata']['name']} has imageID {image_id}, expected {expected['digest']}"
        if not status.get("ready", False):
            return False, f"{pod['metadata']['name']} container {name} is not ready"
        return True, ""
    return False, f"{pod['metadata']['name']} has no matching container for {service}"


def deployment_ready(deployment):
    name = deployment.get("metadata", {}).get("name", "<unknown>")
    desired = deployment.get("spec", {}).get("replicas", 1)
    status = deployment.get("status", {})
    generation = deployment.get("metadata", {}).get("generation", 0)
    observed = status.get("observedGeneration", 0)

    if observed < generation:
        return False, f"{name} observedGeneration {observed} is behind generation {generation}"
    if status.get("updatedReplicas", 0) < desired:
        return False, f"{name} updatedReplicas {status.get('updatedReplicas', 0)} is below desired {desired}"
    if status.get("availableReplicas", 0) < desired:
        return False, f"{name} availableReplicas {status.get('availableReplicas', 0)} is below desired {desired}"

    for condition in status.get("conditions", []):
        if condition.get("type") == "Available" and condition.get("status") == "True":
            return True, ""

    return False, f"{name} is not Available"


def evaluate(client, args, expected_services, app):
    reasons = []
    namespace = args.namespace or app.get("spec", {}).get("destination", {}).get("namespace", "")

    if not namespace:
        reasons.append("Could not determine destination namespace from arguments or Argo CD application.")
        return False, reasons

    revision = app_revision(app)
    sync_status = app.get("status", {}).get("sync", {}).get("status", "")
    health_status = app.get("status", {}).get("health", {}).get("status", "")

    if not app_has_revision(app, args.os_config_revision):
        reasons.append(
            f"Argo CD has not reported revision {args.os_config_revision} yet; current revision is {revision or '<empty>'}."
        )
    if sync_status != "Synced":
        reasons.append(f"Argo CD sync status is {sync_status or '<empty>'}, expected Synced.")
    if health_status != "Healthy":
        reasons.append(f"Argo CD health is {health_status or '<empty>'}, expected Healthy.")

    tree = client.resource_tree(args.app, args.project)
    pod_nodes = [
        node for node in tree.get("nodes", [])
        if node.get("kind") == "Pod" and node.get("namespace") == namespace and node.get("name")
    ]

    pods_by_service = {service: [] for service in expected_services}
    for node in pod_nodes:
        try:
            pod_payload = client.resource(args.app, args.project, "", "v1", "Pod", namespace, node["name"])
            pod = decode_resource_payload(pod_payload)
        except Exception as error:
            reasons.append(f"Could not inspect Pod {node['name']}: {error}")
            continue

        labels = pod.get("metadata", {}).get("labels", {})
        service = labels.get("app.kubernetes.io/name")
        if service in pods_by_service:
            pods_by_service[service].append(pod)

    for service, expected in sorted(expected_services.items()):
        try:
            deployment_payload = client.resource(args.app, args.project, "apps", "v1", "Deployment", namespace, service)
            deployment = decode_resource_payload(deployment_payload)
        except Exception as error:
            reasons.append(f"Could not inspect Deployment {service}: {error}")
            continue

        annotations = deployment.get("spec", {}).get("template", {}).get("metadata", {}).get("annotations", {})
        expected_annotations = {
            "asterism.dev/release-version": args.release_version,
            "asterism.dev/release-commit": args.release_commit,
            "asterism.dev/release-manifest-sha256": args.manifest_sha256,
        }
        for key, value in expected_annotations.items():
            if annotations.get(key) != value:
                reasons.append(f"Deployment {service} annotation {key} is {annotations.get(key)!r}, expected {value!r}.")

        ready, reason = deployment_ready(deployment)
        if not ready:
            reasons.append(reason)

        pods = pods_by_service.get(service, [])
        if not pods:
            reasons.append(f"No pods found for service {service}.")
            continue

        for pod in pods:
            phase = pod.get("status", {}).get("phase")
            pod_name = pod.get("metadata", {}).get("name", "<unknown>")
            if phase != "Running":
                reasons.append(f"Pod {pod_name} phase is {phase}, expected Running.")
                continue
            matches, reason = pod_digest_matches(pod, service, expected)
            if not matches:
                reasons.append(reason)

    return len(reasons) == 0, reasons


def main():
    args = parse_args()
    expected_services = load_manifest(args.release_manifest)
    client = ArgoClient(args.server, args.token)

    print(f"Refreshing Argo CD application {args.app}.")
    deadline = time.monotonic() + args.timeout_seconds
    last_reasons = []

    while time.monotonic() < deadline:
        app = client.get_application(args.app, args.project, refresh="hard")
        ok, reasons = evaluate(client, args, expected_services, app)
        if ok:
            print("Argo CD deployment verification succeeded.")
            return

        last_reasons = reasons
        print("Waiting for Argo CD deployment verification:")
        for reason in reasons[:12]:
            print(f"- {reason}")
        if len(reasons) > 12:
            print(f"- ...and {len(reasons) - 12} more")

        time.sleep(args.poll_interval_seconds)

    print("Timed out waiting for Argo CD deployment verification.", file=sys.stderr)
    for reason in last_reasons:
        print(f"- {reason}", file=sys.stderr)
    sys.exit(1)


if __name__ == "__main__":
    main()
