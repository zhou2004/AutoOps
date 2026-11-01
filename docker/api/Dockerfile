FROM crpi-aj3vgoxp9kzh2jx4.cn-hangzhou.personal.cr.aliyuncs.com/zhangfan_k8s/golang:1.25-alpine

WORKDIR /app

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# ========== 安装 Python + Ansible（虚拟环境）==========
RUN apk --no-cache add \
    python3 \
    py3-pip \
    py3-virtualenv \
    openssh-client \
    sshpass \
    libffi-dev \
    openssl-dev \
    python3-dev \
    gcc \
    musl-dev \
    ca-certificates \
    tzdata \
    curl \
    bash \
    git && \
    rm -rf /var/cache/apk/*

# 创建虚拟环境安装 Ansible
RUN python3 -m venv /opt/ansible-venv && \
    /opt/ansible-venv/bin/pip install --no-cache-dir \
        -i https://pypi.tuna.tsinghua.edu.cn/simple \
        ansible ansible-core paramiko && \
    rm -rf /root/.cache/pip

ENV PATH="/opt/ansible-venv/bin:$PATH" \
    ANSIBLE_HOST_KEY_CHECKING=False \
    ANSIBLE_CONFIG=/app/ansible/ansible.cfg

# ========== 固定 kubeasz 系统 Ansible 源码（不会执行集群操作） ==========
# 二进制 / 离线镜像、SSH 密钥和集群状态通过部署侧卷提供，详见 docs/2026-10-01-kubeasz-system-operations.md。
RUN apk --no-cache add coreutils rsync jq util-linux
# 独立运行时避免影响普通 Ansible 任务；固定 core 2.18 保持旧 role 模板语义。
RUN python3 -m venv /opt/kubeasz-venv && \
    /opt/kubeasz-venv/bin/pip install --no-cache-dir \
        -i https://pypi.tuna.tsinghua.edu.cn/simple \
        ansible==11.13.0 ansible-core==2.18.18

COPY docker/kubeasz/install-system.sh /usr/local/bin/install-kubeasz-system
RUN /bin/sh /usr/local/bin/install-kubeasz-system

# ========== Go 应用构建 ==========
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GOPATH=/go \
    GOCACHE=/go-cache

RUN mkdir -p /app/log /app/upload /app/docs /go/pkg/mod /go-cache /app/ansible && \
    chmod -R 777 /go /go-cache

COPY api/ /app/
RUN /opt/ansible-venv/bin/ansible-galaxy collection install -r /app/automation/database/requirements.yml
# Database roles use the tested controller release independently of ordinary jobs.
RUN python3 -m venv /opt/database-ansible-venv && \
    /opt/database-ansible-venv/bin/pip install --no-cache-dir ansible-core==2.17.14 paramiko && \
    /opt/database-ansible-venv/bin/ansible-galaxy collection install -r /app/automation/database/requirements.yml
ENV DATABASE_ANSIBLE_PLAYBOOK=/opt/database-ansible-venv/bin/ansible-playbook
ENV AUTOOPS_DATABASE_AUTOMATION_DIR=/app/automation/database
ENV DATABASE_PLATFORM_ROOT=/app/automation/database-platform
RUN CGO_ENABLED=0 GOOS=linux go build -o devops . && chmod +x /app/devops

EXPOSE 8000
CMD ["/app/devops"]
