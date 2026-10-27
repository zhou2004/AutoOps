#!/bin/sh
# Install source only. This does not connect to, provision, or alter K8s nodes.
set -eu
version=3.6.8
commit=0e9cee984c211e9e9f9e64910206baab9fbd3d8e
archive_sha=4710380c801caba3e157a0589e62dd839452f8826b5fcae921f44699ccb72c18
root=/etc/kubeasz
if [ -e "$root/ezctl" ]; then
  echo 'Refusing to overwrite existing /etc/kubeasz. Back up and review it first.' >&2
  exit 1
fi
archive=$(mktemp)
trap 'rm -f "$archive"' EXIT HUP INT TERM
curl --fail --location --proto '=https' --tlsv1.2 --retry 3 \
  "https://codeload.github.com/easzlab/kubeasz/tar.gz/$commit" -o "$archive"
printf '%s  %s\n' "$archive_sha" "$archive" | sha256sum -c -
mkdir -p "$root"
tar -xzf "$archive" --strip-components=1 -C "$root"
printf '%s %s\n' "$version" "$commit" > "$root/.autoops-version"
chmod +x "$root/ezctl" "$root/ezdown"
mkdir -p "$root/clusters" "$root/autoops-playbooks" "$root/.autoops"
chmod 700 "$root/clusters" "$root/.autoops"
for addon in metrics-server coredns local-path prometheus ingress-nginx; do
  case "$addon" in
    local-path) task=local-storage.yml ;;
    *) task="$addon.yml" ;;
  esac
  # This static wrapper imports the official, pinned role task without running
  # every other configured addon. Inputs never supply the role or task path.
  cat > "$root/autoops-playbooks/$addon.yml" <<YAML
---
- name: AutoOps system addon $addon
  hosts: localhost
  gather_facts: false
  vars:
    CHANGE_CA: false
  tasks:
    - name: Install $addon from the pinned kubeasz role
      ansible.builtin.import_role:
        name: cluster-addon
        tasks_from: $task
YAML
done
