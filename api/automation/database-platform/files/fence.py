"""Independent Orchestrator PreFailoverProcesses executor, fail closed.

The root-owned configuration maps exact database identities to a separate SSH
management endpoint and an allowlisted service. It never calls the AutoOps API.
An SSH timeout or an unreachable database port is never proof of isolation.
"""
import json
import os
import re
import stat
import subprocess
import sys

def fence(path, host, port, action='fence'):
    if action not in ('fence','persist-promotion'):raise ValueError('invalid hook action')
    st = os.stat(path)
    if st.st_uid != 0 or stat.S_IMODE(st.st_mode) & 0o022:
        raise ValueError("fencing configuration must be root owned and not writable by others")
    config = json.load(open(path, encoding="utf-8"))
    failed = f"{host}:{int(port)}"
    if failed not in config['targets']:raise ValueError('unknown database identity')
    # A stale native topology may name a former primary while another known
    # instance is writable. Fence every registered data instance before any
    # promotion. Healthy replicas stay running read-only. Unknown/unreachable
    # independent management channels fail the whole recovery closed.
    identities = sorted(config['targets']) if action == 'fence' else [failed]
    for identity in identities:
        execute(config,config['targets'][identity],action)

def execute(config,target,action):
    service = target["service"]
    if not re.fullmatch(r"autoops-[A-Za-z0-9_.@-]{1,100}", service):
        raise ValueError("service is not explicitly managed")
    if not re.fullmatch(r"[a-zA-Z0-9_.:-]+", target["address"]):
        raise ValueError("invalid management address")
    # A root-owned forced-command dispatcher limits the SSH key to these exact
    # managed services. A positive response is required, not a reachability test.
    remote = action + " " + service
    args = ["ssh", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=5", "-o", "UserKnownHostsFile="+config["knownHosts"], "-i", config["privateKey"], "-p", str(int(target.get("sshPort",22))), target["username"]+"@"+target["address"], remote]
    result = subprocess.run(args, capture_output=True, text=True, timeout=30, check=True)
    expected = 'FENCED:' if action == 'fence' else 'PERSISTED:'
    if result.stdout.strip() != expected+service:
        raise RuntimeError("independent management channel did not confirm fencing")

if __name__ == "__main__":
    try:
        fence(*sys.argv[1:])
    except Exception:
        print("FENCING FAILED: promotion blocked; inspect the independent management channel", file=sys.stderr)
        sys.exit(1)
