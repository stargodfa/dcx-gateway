#!/usr/bin/env python3
"""Check documentation names, titles, local links and Swagger UI resources."""

from __future__ import annotations

import argparse
from html.parser import HTMLParser
from pathlib import Path
import re
from urllib.parse import unquote, urlsplit


NAME = re.compile(r"[a-z0-9]+(?:-[a-z0-9]+)*(?:\.en)?\.[a-z0-9]+")
DIRECTORY = re.compile(r"[a-z0-9]+(?:-[a-z0-9]+)*")
LINK = re.compile(r"\[[^\]\n]*\]\(([^)\n]+)\)")


def markdown_prose(text: str) -> str:
    lines = []
    fence = ""
    for line in text.splitlines():
        match = re.match(r"^\s*(`{3,}|~{3,})", line)
        if match:
            marker = match.group(1)
            if not fence:
                fence = marker
            elif marker[0] == fence[0] and len(marker) >= len(fence):
                fence = ""
            lines.append("")
        else:
            lines.append("" if fence else line)
    return "\n".join(lines)


def heading_anchors(text: str) -> set[str]:
    anchors = set()
    occurrences: dict[str, int] = {}
    for line in markdown_prose(text).splitlines():
        match = re.match(r"^#{1,6}\s+(.+?)\s*#*\s*$", line)
        if not match:
            continue
        heading = re.sub(r"<[^>]+>", "", match.group(1)).lower()
        heading = re.sub(r"\[([^\]]+)\]\([^)]*\)", r"\1", heading)
        slug = re.sub(r"[^\w\s-]", "", heading)
        slug = re.sub(r"\s", "-", slug)
        count = occurrences.get(slug, 0)
        occurrences[slug] = count + 1
        anchors.add(slug if count == 0 else f"{slug}-{count}")
    return anchors


class Resources(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.urls: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        for name, value in attrs:
            if name in {"src", "href"} and value:
                self.urls.append(value)


def check(root: Path) -> tuple[list[str], int]:
    docs = root / "docs"
    errors = []
    checked_links = 0
    if not (docs / "README.md").is_file():
        errors.append("docs/README.md: missing documentation entrypoint")
    for path in sorted(docs.rglob("*")):
        relative = path.relative_to(root).as_posix()
        if path.is_dir() and not DIRECTORY.fullmatch(path.name):
            errors.append(f"{relative}: directory must use lowercase kebab-case")
        if path.is_file() and path.name != "README.md" and not NAME.fullmatch(path.name):
            errors.append(f"{relative}: filename must use lowercase kebab-case")

    def check_url(source: Path, url: str) -> None:
        nonlocal checked_links
        parsed = urlsplit(url.strip().removeprefix("<").removesuffix(">"))
        if parsed.scheme or parsed.netloc:
            return
        target = (source.parent / unquote(parsed.path)).resolve() if parsed.path else source
        checked_links += 1
        label = source.relative_to(root).as_posix()
        if not target.is_relative_to(root):
            errors.append(f"{label}: local link leaves repository: {url}")
        elif not target.exists():
            errors.append(f"{label}: missing local target: {url}")
        elif parsed.fragment and target.suffix == ".md":
            anchors = heading_anchors(target.read_text(encoding="utf-8-sig"))
            if unquote(parsed.fragment) not in anchors:
                errors.append(f"{label}: missing heading anchor: {url}")

    markdowns = sorted(docs.rglob("*.md"))
    if (root / "readme.md").is_file():
        markdowns.append(root / "readme.md")
    for path in markdowns:
        prose = markdown_prose(path.read_text(encoding="utf-8-sig"))
        if len(re.findall(r"^#\s+", prose, re.MULTILINE)) != 1:
            errors.append(f"{path.relative_to(root).as_posix()}: expected exactly one H1 title")
        for match in LINK.finditer(prose):
            check_url(path, match.group(1))
    ui = docs / "api" / "swagger-ui" / "index.html"
    if not ui.is_file():
        errors.append("docs/api/swagger-ui/index.html: missing Swagger UI entrypoint")
    else:
        text = ui.read_text(encoding="utf-8-sig")
        resources = Resources()
        resources.feed(text)
        for url in resources.urls:
            check_url(ui, url)
        for url in re.findall(r'\burl:\s*["\']([^"\']+)["\']', text):
            check_url(ui, url)
    return errors, checked_links


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    root = args.root.resolve()
    if not (root / "docs").is_dir():
        parser.error("root must contain a docs directory")
    errors, count = check(root)
    for error in errors:
        print(error)
    print(f"Documentation check: {len(errors)} error(s), {count} local link(s) checked.")
    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
