#!/bin/bash
# ============================================================
# TopoLVM 一键部署脚本（适配 AutoOps 测试集群, K8s v1.34.3）
# 部署方式: Helm Chart (topolvm/topolvm 17.1.0, app 0.41.0)
#
# 集群节点:
#   master-01: 192.168.80.51 (master, 仅控制面, kube-scheduler 所在)
#   worker-01: 192.168.80.52 (独立盘 /dev/sda 20G -> LVM VG: myvg)
#   worker-02: 192.168.80.53 (独立盘 /dev/sdb 20G -> LVM VG: myvg)
#
# 前置依赖:
#   1. 所有节点已安装 lvm2 + thin-provisioning-tools (worker 还需 mkfs.ext4)
#   2. master 已安装 helm3
#   3. 集群可访问 github.com (cert-manager)、ghcr.io (topolvm 镜像) 与国内镜像源
#   4. containerd 已配置镜像源: ghcr.io 走 ghcr.m.daocloud.io / ghcr.nju.edu.cn
#      docker.io 走 docker.m.daocloud.io / docker.1ms.run / hub1.nat.tf / hub.rat.dev
#      (参考 k8s/storage/topolvm/containerd-mirrors.md)
#
# 用法: ./deploy-topolvm.sh
# ============================================================
set -e

MASTER_IP="192.168.80.51"
WORKER1_IP="192.168.80.52"
WORKER2_IP="192.168.80.53"
SSH_PASS="123456"

SSH="sshpass -p ${SSH_PASS} ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10"
SCP="sshpass -p ${SSH_PASS} scp -o StrictHostKeyChecking=no -o ConnectTimeout=10"

HELM_CHART="topolvm/topolvm"
CHART_VERSION="17.1.0"
CERT_MANAGER_VERSION="v1.17.4"
NS="test-pv"                     # 测试命名空间
MYSQL_IMG="swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/mysql:8.0.40"

echo "================================================"
echo " TopoLVM 部署 - K8s v1.34.3 三节点集群 (Helm 方式)"
echo "================================================"

# 1. 为 worker 节点创建 LVM 卷组 (幂等)
echo "[1/8] 创建 LVM 卷组 myvg (worker-01:/dev/sda, worker-02:/dev/sdb)..."
$SSH root@${WORKER1_IP} "pvs /dev/sda >/dev/null 2>&1 || pvcreate -y /dev/sda; vgs myvg >/dev/null 2>&1 || vgcreate myvg /dev/sda; vgs myvg"
$SSH root@${WORKER2_IP} "pvs /dev/sdb >/dev/null 2>&1 || pvcreate -y /dev/sdb; vgs myvg >/dev/null 2>&1 || vgcreate myvg /dev/sdb; vgs myvg"

# 2. master 安装 helm (幂等)
echo "[2/8] 确保 master 已安装 helm3..."
$SSH root@${MASTER_IP} "command -v helm >/dev/null 2>&1 || (cd /tmp && curl -sSL -o helm.tgz https://get.helm.sh/helm-v3.16.4-linux-amd64.tar.gz && tar xzf helm.tgz && mv linux-amd64/helm /usr/local/bin/ && rm -rf linux-amd64 helm.tgz); helm version --short"

# 3. 安装 cert-manager (TopoLVM webhook 前置依赖)
echo "[3/8] 安装 cert-manager ${CERT_MANAGER_VERSION}..."
$SSH root@${MASTER_IP} "kubectl get ns cert-manager >/dev/null 2>&1 || (curl -sSL -o /tmp/cert-manager.yaml --max-time 60 https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml && kubectl apply -f /tmp/cert-manager.yaml); kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cert-manager -n cert-manager --timeout=120s"

# 4. 添加 TopoLVM Helm 仓库并部署
echo "[4/8] 添加 topolvm Helm 仓库并部署..."
$SSH root@${MASTER_IP} "helm repo add topolvm https://topolvm.github.io/topolvm >/dev/null 2>&1 || true; helm repo update >/dev/null 2>&1 || true"
$SCP topolvm-values.yaml root@${MASTER_IP}:/tmp/topolvm-values.yaml
$SSH root@${MASTER_IP} "kubectl label node master-01 node-role.kubernetes.io/control-plane= --overwrite >/dev/null 2>&1 || true; helm upgrade --install topolvm ${HELM_CHART} -n topolvm-system --create-namespace --version ${CHART_VERSION} -f /tmp/topolvm-values.yaml"

# 5. 配置 kube-scheduler 使用 TopoLVM extender (HostNetwork, 127.0.0.1:9251)
echo "[5/8] 配置 kube-scheduler extender (指向 topolvm-scheduler:9251)..."
$SCP kube-scheduler-config.yaml root@${MASTER_IP}:/etc/kubernetes/kube-scheduler-config.yaml
$SSH root@${MASTER_IP} 'cat > /etc/systemd/system/kube-scheduler.service << "EOF"
[Unit]
Description=Kubernetes Scheduler
Documentation=https://github.com/GoogleCloudPlatform/kubernetes

[Service]
ExecStart=/opt/kube/bin/kube-scheduler \
  --config=/etc/kubernetes/kube-scheduler-config.yaml \
  --authentication-kubeconfig=/etc/kubernetes/kube-scheduler.kubeconfig \
  --authorization-kubeconfig=/etc/kubernetes/kube-scheduler.kubeconfig \
  --bind-address=0.0.0.0 \
  --leader-elect=true \
  --v=2
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload && systemctl restart kube-scheduler && sleep 5 && systemctl is-active kube-scheduler'

# 6. 将 lvmd/node 限制到有 VG 的 worker 节点 (master 无 VG 会 CrashLoop)
echo "[6/8] 限制 lvmd/node DaemonSet 到 worker 节点 (master 无 VG)..."
$SSH root@${MASTER_IP} 'for ds in topolvm-lvmd-0 topolvm-node; do kubectl patch ds $ds -n topolvm-system --type=strategic -p "{\"spec\":{\"template\":{\"spec\":{\"affinity\":{\"nodeAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"kubernetes.io/hostname\",\"operator\":\"In\",\"values\":[\"worker-01\",\"worker-02\"]}]}]}}}}}}}" >/dev/null 2>&1 || true; done'

# 7. 等待 TopoLVM 组件就绪
echo "[7/8] 等待 TopoLVM 组件就绪 (首次拉取镜像可能需 5-10 分钟)..."
$SSH root@${MASTER_IP} "kubectl -n topolvm-system rollout status ds/topolvm-lvmd-0 --timeout=600s || true; kubectl -n topolvm-system rollout status ds/topolvm-node --timeout=600s || true; kubectl -n topolvm-system rollout status deploy/topolvm-controller --timeout=600s || true; kubectl -n topolvm-system get pods -o wide"

# 8. 创建测试资源并验证
echo "[8/8] 创建测试资源 (namespace=${NS}, mysql 镜像=${MYSQL_IMG})..."
$SSH root@${MASTER_IP} "kubectl create namespace ${NS} --dry-run=client -o yaml | kubectl apply -f -"
$SCP topolvm-deploy.yaml root@${MASTER_IP}:/root/topolvm-deploy.yaml
$SSH root@${MASTER_IP} "kubectl apply -f /root/topolvm-deploy.yaml 2>&1 | grep -vE 'last-applied|will be patched|is invalid' || true"
$SSH root@${MASTER_IP} "sleep 30; kubectl get sc; echo '---PVC---'; kubectl get pvc -n ${NS}; echo '---PV---'; kubectl get pv | grep topolvm; echo '---PODS---'; kubectl get pods -n ${NS} -o wide"

echo ""
echo "================================================"
echo " TopoLVM 部署完成！"
echo " 验证命令:"
echo "   kubectl -n topolvm-system get pods -o wide"
echo "   kubectl get sc/pvc/pv -A"
echo "   kubectl get pods -n ${NS} -o wide"
echo " 底层 LVM (在 worker 节点): vgs myvg; lvs myvg"
echo "================================================"
