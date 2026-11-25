#!/usr/bin/env python3
"""Audit a public source tree without printing matched credentials or data."""

from __future__ import annotations

import argparse
import ipaddress
import re
from pathlib import Path


PRIVATE_NETWORKS = tuple(
    ipaddress.ip_network(network)
    for network in ("10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16")
)
SKIP_DIRECTORIES = {".git", "node_modules", ".venv", "venv", "__pycache__"}
VENDORED_ASSETS = {
    "docs/api/swagger-ui/swagger-ui.css",
    "docs/api/swagger-ui/swagger-ui-bundle.js",
    "docs/api/swagger-ui/swagger-ui-standalone-preset.js",
}
OPAQUE_SUFFIXES = {
    ".zip", ".gz", ".tgz", ".tar", ".7z", ".rar", ".pdf", ".exe",
    ".db", ".sqlite", ".sqlite3", ".bak", ".pem", ".key", ".p12", ".pfx",
}
IPV4 = re.compile(r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])")
CONTACT = re.compile(r"[\w.+-]+@([\w.-]+\.[A-Za-z]{2,})")
PERSONAL_PATH = re.compile(r"(?<![A-Za-z])[A-Za-z]:[\\/](?![\\/])[A-Za-z0-9_.-]|/(?:home|Users)/[^/\s`]+")
CREDENTIAL = re.compile(
    r"\b[\"']?(?:password|passwd|secret|static_token|api_key|token)[\"']?"
    r"\s*[:=]\s*([\"'])([^\"'\r\n]*)\1", re.IGNORECASE
)
DSN_PASSWORD = re.compile(r"[^\s:\"']+:([^\s@\"']+)@tcp\(")
SECRET_PATTERNS = (
    re.compile(r"-----BEGIN (?:[A-Z]+ )?PRIVATE KEY-----"),
    re.compile(r"\b(?:AKIA|ASIA)[A-Z0-9]{16}\b"),
    re.compile(r"\bLTAI[A-Za-z0-9]{16,28}\b"),
    re.compile(r"\bgh[pousr]_[A-Za-z0-9]{30,}\b"),
    re.compile(r"\bsk-(?:proj-|svcacct-)?[A-Za-z0-9_-]{24,}\b"),
    re.compile(r"\beyJ[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}\b"),
)
BCRYPT_HASH = re.compile(r"\$2[aby]\$\d{2}\$[./A-Za-z0-9]{53}")
LEGACY_DUMP = re.compile(
    r"^\s*(?:--\s*)?(?:Navicat.*Data Transfer|Dump completed)|\bDEFINER\s*=", re.IGNORECASE
)


def placeholder(value: str) -> bool:
    return (
        not value
        or (value.startswith("<") and value.endswith(">"))
        or value.startswith(("${", "replace-with-"))
    )


def private_address(line: str) -> bool:
    for match in IPV4.finditer(line):
        # The RFC range definitions themselves are public network constants.
        if any(line[match.start():].startswith(str(network)) for network in PRIVATE_NETWORKS):
            continue
        try:
            address = ipaddress.ip_address(match.group())
        except ValueError:
            continue
        if any(address in network for network in PRIVATE_NETWORKS):
            return True
    return False


def example_contact(domain: str) -> bool:
    domain = domain.lower()
    return domain in {"example.com", "example.org", "example.net"} or domain.endswith(
        (".example.com", ".example.org", ".example.net", ".test", ".example", ".invalid")
    )


def audit(root: Path) -> list[tuple[str, int, str]]:
    findings: set[tuple[str, int, str]] = set()
    for path in root.rglob("*"):
        relative = path.relative_to(root)
        if any(part in SKIP_DIRECTORIES for part in relative.parts):
            continue
        name = relative.as_posix()
        if path.is_symlink():
            findings.add((name, 0, "symlink_requires_review"))
            continue
        if not path.is_file() or name in VENDORED_ASSETS or name == "go.sum":
            continue
        if path.suffix.lower() in OPAQUE_SUFFIXES:
            findings.add((name, 0, "generated_export_or_private_key"))
            continue
        if (path.name.startswith(".env") and path.name != ".env.example") or (
            relative.parts[0] == "configs" and path.name != "config.example.yaml"
        ):
            findings.add((name, 0, "local_configuration"))
        if relative.parts[0] == "migrations" and path.suffix == ".sql" and not re.fullmatch(
            r"\d{3}_[a-z0-9_]+\.sql", path.name
        ):
            findings.add((name, 0, "unreviewed_database_export"))
        try:
            text = path.read_text(encoding="utf-8-sig")
        except (UnicodeError, OSError):
            findings.add((name, 0, "unreadable_or_binary_file"))
            continue
        is_test = path.name.endswith("_test.go")
        for line_number, line in enumerate(text.splitlines(), 1):
            categories = []
            if private_address(line):
                categories.append("private_network_address")
            if PERSONAL_PATH.search(line):
                categories.append("absolute_personal_path")
            if any(not example_contact(match.group(1)) for match in CONTACT.finditer(line)):
                categories.append("non_example_contact")
            if any(pattern.search(line) for pattern in SECRET_PATTERNS):
                categories.append("credential_or_password_hash")
            # This existing public dummy is not stored against any login user;
            # it equalizes timing for nonexistent accounts and must be retained.
            dummy_hash = name == "internal/service/auth_service.go" and line.startswith("const dummyBcryptHash = ")
            if BCRYPT_HASH.search(line) and not dummy_hash:
                categories.append("credential_or_password_hash")
            if not is_test and (
                any(not placeholder(match.group(2)) for match in CREDENTIAL.finditer(line))
                or any(not placeholder(match.group(1)) for match in DSN_PASSWORD.finditer(line))
            ):
                categories.append("non_placeholder_credential")
            if path.suffix == ".sql" and LEGACY_DUMP.search(line):
                categories.append("database_dump_metadata")
            for category in categories:
                findings.add((name, line_number, category))
    return sorted(findings)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    root = args.root.resolve()
    if not root.is_dir():
        parser.error("root must be an existing directory")
    findings = audit(root)
    for name, line, category in findings:
        print(f"{name}:{line}: {category}")
    print(f"Public release audit: {len(findings)} finding(s). Matched content is never printed.")
    return 1 if findings else 0


if __name__ == "__main__":
    raise SystemExit(main())
