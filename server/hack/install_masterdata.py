#!/usr/bin/env python3
"""Safely patch, validate, install, and optionally reload master data."""

from __future__ import annotations

import argparse
import hashlib
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request


ADMIN_TOKEN_ENV = "LUNAR_ADMIN_TOKEN"


def run_patcher(patcher: Path, source: Path, output: Path) -> None:
    subprocess.run(
        [
            sys.executable,
            str(patcher),
            "--input",
            str(source),
            "--output",
            str(output),
        ],
        check=True,
    )


def validate_candidate(patcher: Path, candidate: Path) -> None:
    print("\nValidating patched master data ...", flush=True)
    subprocess.run(
        [sys.executable, str(patcher), "--input", str(candidate), "--dry-run"],
        check=True,
    )


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def install_candidate(candidate: Path, target: Path) -> None:
    if candidate.stat().st_size == 0:
        raise RuntimeError("patcher produced an empty master data file")

    target.parent.mkdir(parents=True, exist_ok=True)
    mode = stat.S_IMODE(target.stat().st_mode) if target.exists() else 0o644
    candidate.chmod(mode)
    os.replace(candidate, target)


def reload_masterdata(url: str) -> None:
    if not url:
        print("Master data reload disabled (MASTERDATA_RELOAD_URL is empty).")
        return

    token = os.environ.get(ADMIN_TOKEN_ENV, "")
    if not token:
        print(f"Master data reload skipped ({ADMIN_TOKEN_ENV} is not set).")
        return

    request = urllib.request.Request(
        url,
        data=b"",
        method="POST",
        headers={"Authorization": f"Bearer {token}"},
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            body = response.read().decode("utf-8", errors="replace")
            if response.status < 200 or response.status >= 300:
                raise RuntimeError(
                    f"master data reload returned HTTP {response.status}: {body}"
                )
    except urllib.error.URLError as exc:
        raise RuntimeError(f"master data installed, but reload failed: {exc}") from exc

    print(f"Master data reloaded through {url}.")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Patch, validate, atomically install, and reload master data."
    )
    parser.add_argument("--patcher", required=True, type=Path)
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--target", required=True, type=Path)
    parser.add_argument("--reload-url", default="")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    patcher = args.patcher.resolve()
    source = args.input.resolve()
    target = args.target.resolve()

    if not patcher.is_file():
        raise FileNotFoundError(f"master data patcher not found: {patcher}")
    if not source.is_file():
        raise FileNotFoundError(f"master data input not found: {source}")

    target.parent.mkdir(parents=True, exist_ok=True)
    file_descriptor, temporary_name = tempfile.mkstemp(
        prefix=f".{target.name}.", suffix=".tmp", dir=target.parent
    )
    os.close(file_descriptor)
    candidate = Path(temporary_name)

    try:
        print(f"Master data input:  {source}")
        print(f"Master data target: {target}")
        run_patcher(patcher, source, candidate)
        validate_candidate(patcher, candidate)
        candidate_hash = sha256(candidate)
        install_candidate(candidate, target)
        print(f"Installed master data atomically: {target}")
        print(f"SHA-256: {candidate_hash}")
    finally:
        candidate.unlink(missing_ok=True)

    reload_masterdata(args.reload_url)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(1)
