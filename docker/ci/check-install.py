#!/usr/bin/env python3
"""Verify a disposable Compose installation without printing its credentials."""
import json
import os
from pathlib import Path
import subprocess
import time
import urllib.error
import urllib.request

root = Path(__file__).resolve().parents[1]
settings = dict(
    line.split("=", 1)
    for line in (root / ".env").read_text().splitlines()
    if line and not line.startswith("#")
)
base = "http://127.0.0.1:" + settings.get("WEB_PORT", "8080")
compose = [
    "docker", "compose", "-p", os.environ.get("COMPOSE_PROJECT_NAME", "autoops-smoke"),
    "--project-directory", str(root),
]


def request(path, data=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(
        base + path, data=json.dumps(data).encode() if data is not None else None,
        headers=headers,
    )
    with urllib.request.urlopen(req, timeout=30) as response:
        return json.load(response)


deadline = time.monotonic() + 180
while True:
    try:
        options = request("/api/v1/auth/options")
        if options["code"] == 200:
            break
    except (urllib.error.URLError, TimeoutError, OSError):
        pass
    if time.monotonic() >= deadline:
        raise RuntimeError("API did not become ready within 180 seconds")
    time.sleep(3)

assert options["data"]["approvalRequired"], "Registration must require approval"
with urllib.request.urlopen(base + "/assets/brand/mark-dark.svg", timeout=15) as response:
    assert b"<svg" in response.read(), "Published frontend is missing its brand icon"

captcha = request("/api/v1/captcha")["data"]
assert captcha["image"].startswith("data:image/"), "Captcha image missing"
code = subprocess.check_output(
    compose + [
        "exec", "-T", "redis", "sh", "-c",
        'REDISCLI_AUTH="$REDIS_PASSWORD" redis-cli --raw get "$1"',
        "sh", "login_code:" + captcha["idKey"],
    ], text=True,
).strip()
assert len(code) == 4, "Captcha not persisted in Redis"
payload = {
    "username": "admin", "password": settings["AUTOOPS_INITIAL_ADMIN_PASSWORD"],
    "idKey": captcha["idKey"], "image": code,
}
login = request("/api/v1/login", payload)
assert login["code"] == 200 and login["data"]["token"], "Generated admin password login failed"
menus = request("/api/v1/menu/vo/list", token=login["data"]["token"])
assert menus["code"] == 200 and menus["data"], "Initial administrator has no menu permissions"
assert request("/api/v1/login", payload)["code"] != 200, "Captcha can be reused"
print("PASS: fresh installation, published branding, initial admin, menu permissions and single-use captcha")
