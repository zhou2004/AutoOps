#!/usr/bin/python3
"""Installed root-owned; allowlist is provisioned by the CMDB deployment."""
import json
import subprocess
import sys

allowed = json.load(open("/etc/autoops-fence-services.json", encoding="utf-8"))
unit = sys.argv[1] if len(sys.argv) == 2 else ""
if unit not in allowed:
    sys.exit(1)
subprocess.run(["systemctl", "stop", unit], check=True, timeout=60)
state = subprocess.run(["systemctl", "show", unit, "-p", "ActiveState", "--value"], check=True, capture_output=True, text=True, timeout=5)
pid = subprocess.run(["systemctl", "show", unit, "-p", "MainPID", "--value"], check=True, capture_output=True, text=True, timeout=5)
if state.stdout.strip() not in ("inactive", "failed") or pid.stdout.strip() != "0":
    sys.exit(1)
print("FENCED:"+unit)
