#!/usr/bin/env python3
"""Render the OpenOMS Helm chart and guard Pod Security Standard assumptions."""

from __future__ import annotations

import subprocess
import sys


def render_chart() -> str:
    result = subprocess.run(
        [
            "helm",
            "template",
            "openoms",
            "deploy/helm/openoms",
            "--set",
            "monitoring.enabled=true",
            "--set",
            "networkPolicy.enabled=true",
        ],
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if result.returncode != 0:
        sys.stderr.write(result.stderr)
        raise SystemExit(result.returncode)
    return result.stdout


def documents(rendered: str) -> list[str]:
    return [doc.strip() for doc in rendered.split("---") if doc.strip()]


def find_doc(rendered_docs: list[str], *, kind: str, name: str) -> str:
    for doc in rendered_docs:
        if f"kind: {kind}" in doc and f"name: {name}" in doc:
            return doc
    raise AssertionError(f"missing {kind}/{name}")


def assert_contains(doc: str, needle: str, context: str) -> None:
    if needle not in doc:
        raise AssertionError(f"{context}: missing {needle!r}")


def assert_not_contains(doc: str, needle: str, context: str) -> None:
    if needle in doc:
        raise AssertionError(f"{context}: forbidden {needle!r}")


def main() -> None:
    rendered_docs = documents(render_chart())

    namespace = find_doc(rendered_docs, kind="Namespace", name="openoms")
    for label in (
        "pod-security.kubernetes.io/enforce: restricted",
        "pod-security.kubernetes.io/audit: restricted",
        "pod-security.kubernetes.io/warn: restricted",
    ):
        assert_contains(namespace, label, "Namespace/openoms")

    alloy = find_doc(rendered_docs, kind="Deployment", name="openoms-alloy")
    for forbidden in (
        "kind: DaemonSet",
        "hostPath:",
        "runAsNonRoot: false",
        "runAsUser: 0",
        "DAC_READ_SEARCH",
        "/var/log",
        "/var/lib/docker/containers",
    ):
        assert_not_contains(alloy, forbidden, "Deployment/openoms-alloy")

    for required in (
        "runAsNonRoot: true",
        "runAsUser: 65534",
        "runAsGroup: 65534",
        "fsGroup: 65534",
        "allowPrivilegeEscalation: false",
        "readOnlyRootFilesystem: true",
        'drop: ["ALL"]',
        "seccompProfile:",
        "type: RuntimeDefault",
    ):
        assert_contains(alloy, required, "Deployment/openoms-alloy")


if __name__ == "__main__":
    try:
        main()
    except AssertionError as exc:
        print(f"Helm PSS check failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
