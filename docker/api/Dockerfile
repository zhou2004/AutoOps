ARG GO_IMAGE=golang:1.25-alpine
# Database Ansible 2.17 requires a Python 3.10–3.12 controller.
ARG PYTHON_IMAGE=python:3.12-alpine3.22

FROM ${GO_IMAGE} AS builder
WORKDIR /src
ENV CGO_ENABLED=0 GOPROXY=https://proxy.golang.org,direct
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/devops .

FROM ${PYTHON_IMAGE} AS runtime
WORKDIR /app
RUN apk add --no-cache \
    openssh-client sshpass libffi-dev openssl-dev gcc musl-dev \
    ca-certificates tzdata curl bash git coreutils rsync jq util-linux openssl

# Keep ordinary tasks, kubeasz and database automation in separate runtimes.
RUN python3 -m venv /opt/ansible-venv && \
    /opt/ansible-venv/bin/pip install --no-cache-dir \
        ansible==11.13.0 ansible-core==2.18.18 paramiko && \
    python3 -m venv /opt/kubeasz-venv && \
    /opt/kubeasz-venv/bin/pip install --no-cache-dir \
        ansible==11.13.0 ansible-core==2.18.18 && \
    python3 -m venv /opt/database-ansible-venv && \
    /opt/database-ansible-venv/bin/pip install --no-cache-dir \
        ansible-core==2.17.14 paramiko

COPY api/automation/database/requirements.yml /tmp/database-collections.yml
RUN /opt/ansible-venv/bin/ansible-galaxy collection install -r /tmp/database-collections.yml && \
    rm /tmp/database-collections.yml
COPY docker/kubeasz/install-system.sh /usr/local/bin/install-kubeasz-system
RUN /bin/sh /usr/local/bin/install-kubeasz-system

# The root .dockerignore excludes credentials, local configuration and runtime data.
COPY api/ /app/
COPY --from=builder /out/devops /app/devops
RUN mkdir -p /app/log /app/upload/xlsl /app/docs /app/ansible /app/data
ENV PATH="/opt/ansible-venv/bin:$PATH" \
    ANSIBLE_HOST_KEY_CHECKING=False \
    ANSIBLE_CONFIG=/app/ansible/ansible.cfg \
    DATABASE_ANSIBLE_PLAYBOOK=/opt/database-ansible-venv/bin/ansible-playbook \
    AUTOOPS_DATABASE_AUTOMATION_DIR=/app/automation/database \
    DATABASE_PLATFORM_ROOT=/app/automation/database-platform \
    DATABASE_WORKER_MODE=external
EXPOSE 8000
CMD ["/app/devops"]
