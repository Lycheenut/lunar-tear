#!/usr/bin/env python3
"""Cross-platform background lifecycle for the cmd/dev service supervisor."""

from __future__ import annotations

import argparse
import ctypes
import ipaddress
import json
from pathlib import Path
import os
import socket
import subprocess
import sys
import time


def process_is_running(pid: int) -> bool:
    if pid <= 0:
        return False

    if os.name == "nt":
        process_query_limited_information = 0x1000
        still_active = 259
        handle = ctypes.windll.kernel32.OpenProcess(
            process_query_limited_information, False, pid
        )
        if not handle:
            return False
        try:
            exit_code = ctypes.c_ulong()
            if not ctypes.windll.kernel32.GetExitCodeProcess(
                handle, ctypes.byref(exit_code)
            ):
                return False
            return exit_code.value == still_active
        finally:
            ctypes.windll.kernel32.CloseHandle(handle)

    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    return True


def read_pid(pid_file: Path) -> int | None:
    try:
        return int(pid_file.read_text(encoding="utf-8").strip())
    except (FileNotFoundError, ValueError):
        return None


def clear_control_files(*paths: Path) -> None:
    for path in paths:
        path.unlink(missing_ok=True)


def wait_until(predicate, timeout: float) -> bool:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if predicate():
            return True
        time.sleep(0.2)
    return predicate()


def has_option(values: list[str], name: str) -> bool:
    prefix = f"{name}="
    return any(value == name or value.startswith(prefix) for value in values)


def option_value(values: list[str], name: str) -> str | None:
    prefix = f"{name}="
    for index, value in enumerate(values):
        if value.startswith(prefix):
            return value[len(prefix) :]
        if value == name and index + 1 < len(values):
            return values[index + 1]
    return None


def listen_port(values: list[str], name: str, default: int) -> int:
    value = option_value(values, name)
    if value is None:
        return default
    try:
        return int(value.rsplit(":", 1)[-1])
    except ValueError:
        return default


def detect_lan_ipv4() -> str:
    candidates: list[str] = []

    # UDP connect selects a route without sending traffic. It is the most
    # reliable cross-platform way to identify the active outbound interface.
    for target in ("1.1.1.1", "8.8.8.8"):
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
                sock.connect((target, 80))
                candidates.append(sock.getsockname()[0])
        except OSError:
            pass

    try:
        candidates.extend(
            address[4][0]
            for address in socket.getaddrinfo(
                socket.gethostname(), None, socket.AF_INET, socket.SOCK_DGRAM
            )
        )
    except OSError:
        pass

    for candidate in candidates:
        address = ipaddress.ip_address(candidate)
        if not (
            address.is_loopback
            or address.is_link_local
            or address.is_multicast
            or address.is_unspecified
        ):
            return candidate

    raise RuntimeError(
        "could not detect a host LAN IPv4 address; pass --cdn.public-addr "
        "and --grpc.public-addr explicitly through ARGS"
    )


def add_detected_public_addresses(values: list[str], lan_ip: str) -> list[str]:
    result = list(values)
    if not has_option(result, "--cdn.public-addr"):
        port = listen_port(result, "--cdn.listen", 8080)
        result.extend(("--cdn.public-addr", f"{lan_ip}:{port}"))
    if not has_option(result, "--grpc.public-addr"):
        port = listen_port(result, "--grpc.listen", 8003)
        result.extend(("--grpc.public-addr", f"{lan_ip}:{port}"))
    return result


def start(args: argparse.Namespace) -> None:
    pid = read_pid(args.pid_file)
    if pid is not None and process_is_running(pid):
        if args.ready_file.is_file():
            print(f"Services already running (pid {pid}).")
            return
        raise RuntimeError(
            f"supervisor pid {pid} is running but not ready; see {args.log_file}"
        )

    clear_control_files(args.pid_file, args.ready_file, args.stop_file)
    args.pid_file.parent.mkdir(parents=True, exist_ok=True)
    args.args_file.parent.mkdir(parents=True, exist_ok=True)
    args.log_file.parent.mkdir(parents=True, exist_ok=True)
    args.binary.parent.mkdir(parents=True, exist_ok=True)

    dev_args = list(args.dev_args)
    missing_public_address = not has_option(
        dev_args, "--cdn.public-addr"
    ) or not has_option(dev_args, "--grpc.public-addr")
    if args.auto_public_addr and missing_public_address:
        lan_ip = detect_lan_ipv4()
        dev_args = add_detected_public_addresses(dev_args, lan_ip)
        print(f"Detected host LAN IPv4: {lan_ip}", flush=True)

    print("Building service supervisor ...", flush=True)
    subprocess.run(
        [args.go, "build", "-o", str(args.binary), "./cmd/dev"], check=True
    )
    args.args_file.write_text(
        json.dumps(args.dev_args, ensure_ascii=False) + "\n", encoding="utf-8"
    )

    command = [
        str(args.binary.resolve()),
        "--no-color",
        "--ready-file",
        str(args.ready_file),
        "--stop-file",
        str(args.stop_file),
        *dev_args,
    ]
    creation_options: dict[str, object] = {}
    if os.name == "nt":
        creation_options["creationflags"] = (
            subprocess.CREATE_NEW_PROCESS_GROUP | subprocess.CREATE_NO_WINDOW
        )
    else:
        creation_options["start_new_session"] = True

    with args.log_file.open("a", encoding="utf-8") as log_file:
        process = subprocess.Popen(
            command,
            cwd=Path.cwd(),
            stdin=subprocess.DEVNULL,
            stdout=log_file,
            stderr=subprocess.STDOUT,
            **creation_options,
        )

    args.pid_file.write_text(f"{process.pid}\n", encoding="utf-8")

    def ready() -> bool:
        if not process_is_running(process.pid):
            raise RuntimeError(
                f"service supervisor exited during startup; see {args.log_file}"
            )
        if not args.ready_file.is_file():
            return False
        return args.ready_file.read_text(encoding="utf-8").strip() == str(
            process.pid
        )

    try:
        if not wait_until(ready, args.start_timeout):
            raise RuntimeError(
                f"service supervisor did not become ready within "
                f"{args.start_timeout:g}s; see {args.log_file}"
            )
    except Exception:
        if not process_is_running(process.pid):
            clear_control_files(args.pid_file, args.ready_file, args.stop_file)
        raise

    print(f"Services started (pid {process.pid}).")
    print(f"Log: {args.log_file.resolve()}")


def stop(args: argparse.Namespace) -> None:
    pid = read_pid(args.pid_file)
    if pid is None or not process_is_running(pid):
        clear_control_files(args.pid_file, args.ready_file, args.stop_file)
        print("Services are not running.")
        return

    args.stop_file.parent.mkdir(parents=True, exist_ok=True)
    args.stop_file.write_text("stop\n", encoding="utf-8")
    if not wait_until(lambda: not process_is_running(pid), args.stop_timeout):
        raise RuntimeError(
            f"supervisor pid {pid} did not stop within {args.stop_timeout:g}s; "
            f"refusing to kill an unverified process"
        )

    clear_control_files(args.pid_file, args.ready_file, args.stop_file)
    print(f"Services stopped (pid {pid}).")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Start, stop, or restart the cmd/dev service supervisor."
    )
    parser.add_argument("action", choices=("start", "stop", "restart"))
    parser.add_argument("--go", default="go")
    parser.add_argument("--binary", type=Path, default=Path("bin/dev"))
    parser.add_argument("--pid-file", required=True, type=Path)
    parser.add_argument("--ready-file", required=True, type=Path)
    parser.add_argument("--stop-file", required=True, type=Path)
    parser.add_argument("--args-file", required=True, type=Path)
    parser.add_argument("--log-file", required=True, type=Path)
    parser.add_argument("--start-timeout", type=float, default=90)
    parser.add_argument("--stop-timeout", type=float, default=30)
    parser.add_argument(
        "--auto-public-addr",
        action="store_true",
        help="fill missing CDN and gRPC public addresses from the host LAN IPv4",
    )
    parser.add_argument("dev_args", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if args.dev_args and args.dev_args[0] == "--":
        args.dev_args = args.dev_args[1:]
    return args


def main() -> int:
    args = parse_args()
    if args.action == "start":
        start(args)
    elif args.action == "stop":
        stop(args)
    else:
        if not args.dev_args and args.args_file.is_file():
            saved_args = json.loads(args.args_file.read_text(encoding="utf-8"))
            if not isinstance(saved_args, list) or not all(
                isinstance(value, str) for value in saved_args
            ):
                raise RuntimeError(f"invalid saved arguments: {args.args_file}")
            args.dev_args = saved_args
        stop(args)
        start(args)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (json.JSONDecodeError, OSError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(1)
