#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Download Chrome for Testing into $CHOYSUM_HOME/browsers/chromium-<rev>/."""

from __future__ import annotations

import argparse
import io
import os
import platform
import sys
import urllib.request
import zipfile
from pathlib import Path

# Pinned CfT version/revision (keep in sync with internal/testing/e2e/cdp.PinnedCfTRevision).
PINNED_VERSION = "152.0.7977.82"
PINNED_REVISION = "1669021"

CFT_JSON = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"


def choysum_home() -> Path:
    env = (os.environ.get("CHOYSUM_HOME") or "").strip()
    if env:
        return Path(env).expanduser().resolve()
    return (Path.home() / ".choysum").resolve()


def platform_key() -> str:
    system = platform.system().lower()
    machine = platform.machine().lower()
    if system == "darwin":
        if machine in ("arm64", "aarch64"):
            return "mac-arm64"
        return "mac-x64"
    if system == "linux":
        return "linux64"
    raise SystemExit(f"unsupported platform: {system}/{machine}")


def chrome_download_url(version: str, plat: str) -> str:
    return f"https://storage.googleapis.com/chrome-for-testing-public/{version}/{plat}/chrome-{plat}.zip"


def resolve_binary(dest: Path, plat: str) -> Path | None:
    if plat.startswith("mac"):
        candidates = [
            dest / f"chrome-{plat}" / "Google Chrome for Testing.app" / "Contents" / "MacOS" / "Google Chrome for Testing",
            dest / "Google Chrome for Testing.app" / "Contents" / "MacOS" / "Google Chrome for Testing",
        ]
    else:
        candidates = [
            dest / f"chrome-{plat}" / "chrome",
            dest / "chrome",
        ]
    for c in candidates:
        if c.is_file():
            return c
    return None


def install(version: str = PINNED_VERSION, revision: str = PINNED_REVISION) -> Path:
    if version != PINNED_VERSION and revision == PINNED_REVISION:
        raise SystemExit(
            f"--version {version!r} requires a matching --revision "
            f"(default pin is version={PINNED_VERSION} revision={PINNED_REVISION})"
        )
    plat = platform_key()
    home = choysum_home()
    # Cache identity includes both version and revision so overrides cannot collide.
    dest = home / "browsers" / f"chromium-{revision}-{version}"
    # Keep the historical revision-only path for the pinned default.
    if version == PINNED_VERSION and revision == PINNED_REVISION:
        dest = home / "browsers" / f"chromium-{revision}"
    dest.mkdir(parents=True, exist_ok=True)
    existing = resolve_binary(dest, plat)
    if existing is not None:
        _ensure_executable_tree(dest)
        if platform.system().lower() == "darwin":
            _clear_macos_quarantine(dest)
        print(existing)
        return existing

    url = chrome_download_url(version, plat)
    print(f"downloading {url}", file=sys.stderr)
    with urllib.request.urlopen(url, timeout=300) as resp:
        data = resp.read()
    with zipfile.ZipFile(io.BytesIO(data)) as zf:
        zf.extractall(dest)

    # Zip entries often lack +x on helpers (GPU/Renderer/crashpad); macOS may also
    # quarantine the tree so subprocess launches fail with error_code=1003.
    _ensure_executable_tree(dest)
    if platform.system().lower() == "darwin":
        _clear_macos_quarantine(dest)

    binary = resolve_binary(dest, plat)
    if binary is None:
        raise SystemExit(f"chrome binary not found under {dest}")
    # Ensure executable bit on unix.
    mode = binary.stat().st_mode
    binary.chmod(mode | 0o111)
    print(binary)
    return binary


def _ensure_executable_tree(root: Path) -> None:
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        name = path.name
        # Heuristic: Chrome helpers and the main binary need execute.
        if (
            "Helper" in name
            or name in ("chrome", "chrome_crashpad_handler", "Google Chrome for Testing", "Chromium")
            or path.suffix == ""
        ):
            mode = path.stat().st_mode
            path.chmod(mode | 0o111)


def _clear_macos_quarantine(root: Path) -> None:
    import subprocess

    try:
        subprocess.run(["xattr", "-cr", str(root)], check=False, capture_output=True)
    except OSError as err:
        # Best-effort only: xattr may be missing or fail; do not abort install.
        print(f"warning: clear quarantine skipped: {err}", file=sys.stderr)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", default=PINNED_VERSION, help="Chrome for Testing version")
    parser.add_argument("--revision", default=PINNED_REVISION, help="Revision directory suffix")
    args = parser.parse_args()
    install(args.version, args.revision)


if __name__ == "__main__":
    main()
