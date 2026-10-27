#!/bin/bash
# ============================================================
# K8s 存储方案一键部署脚本
# 用法: ./deploy-storage.sh <topolvm|mayastor|rook-ceph>
# 适用于 192.168.80.51/52/53 三节点环境
# ============================================================
set -e

SCHEME=${1:-topolvm}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "================================================"
echo " 部署 K8s 存储方案: $SCHEME"
echo "================================================"

# 前置检查
check_prereqs() {
    echo "[1/4] 检查前置条件..."
    command -v kubectl >/dev/null 2>&1 || { echo "错误: 未找到 kubectl"; exit 1; }
    command -v helm >/dev/null 2>&1 || echo "警告: 未找到 helm，部分方案需要"
    
    # 检查 worker 节点磁盘
    for node in 192.168.80.52 192.168.80.53; do
        echo "  检查节点 $node 的 /dev/sdb..."
        ssh -o StrictHostKeyChecking=no root@$node "lsblk /dev/sdb" 2>/dev/null || \
            echo "  警告: 无法访问 $node 或 /dev/sdb 不存在"
    done
}

# 部署 TopoLVM
deploy_topolvm() {
    echo "[2/4] 部署 TopoLVM..."
    helm repo add topolvm https://topolvm.github.io/topolvm 2>/dev/null || true
    helm repo update
    helm install topolvm topolvm/topolvm \
        --namespace topolvm-system \
        --create-namespace \
        --wait

    echo "[3/4] 创建 LVM 卷组..."
    for node in 192.168.80.52 192.168.80.53; do
        echo "  在 $node 上创建卷组 myvg..."
        ssh -o StrictHostKeyChecking=no root@$node \
            "pvcreate /dev/sdb && vgcreate myvg /dev/sdb" 2>/dev/null || \
            echo "  $node 卷组创建失败（可能已存在或磁盘未就绪）"
    done

    echo "[4/4] 应用 StorageClass 和测试资源..."
    kubectl create namespace test_pv --dry-run=client -o yaml | kubectl apply -f -
    kubectl apply -f "$SCRIPT_DIR/topolvm/topolvm-deploy.yaml"
}

# 部署 OpenEBS Mayastor
deploy_mayastor() {
    echo "[2/4] 部署 OpenEBS Mayastor..."
    helm repo add openebs https://openebs.github.io/openebs 2>/dev/null || true
    helm repo update
    helm install mayastor openebs/mayastor \
        --namespace mayastor \
        --create-namespace \
        --set etcd.persistence.storageClass=local-path \
        --wait

    echo "[3/4] 等待 Mayastor 就绪..."
    kubectl -n mayastor wait --for=condition=Ready pod -l app=io-engine --timeout=300s || true

    echo "[4/4] 应用磁盘池和 StorageClass..."
    kubectl create namespace test_pv --dry-run=client -o yaml | kubectl apply -f -
    kubectl apply -f "$SCRIPT_DIR/openebs-mayastor/mayastor-deploy.yaml"
}

# 部署 Ceph + Rook
deploy_rook_ceph() {
    echo "[2/4] 部署 Rook Operator..."
    ROOK_VERSION="v1.14.0"
    kubectl apply -f "https://raw.githubusercontent.com/rook/rook/${ROOK_VERSION}/deploy/examples/crds.yaml"
    kubectl apply -f "https://raw.githubusercontent.com/rook/rook/${ROOK_VERSION}/deploy/examples/common.yaml"
    kubectl apply -f "https://raw.githubusercontent.com/rook/rook/${ROOK_VERSION}/deploy/examples/operator.yaml"

    echo "[3/4] 等待 Rook Operator 就绪..."
    kubectl -n rook-ceph wait --for=condition=Ready pod -l app=rook-ceph-operator --timeout=300s || true

    echo "[4/4] 应用 Ceph 集群和 StorageClass..."
    kubectl create namespace test_pv --dry-run=client -o yaml | kubectl apply -f -
    kubectl apply -f "$SCRIPT_DIR/ceph-rook/rook-ceph-deploy.yaml"
    echo "提示: Ceph 集群初始化需要 10-20 分钟，请耐心等待"
}

case "$SCHEME" in
    topolvm)
        check_prereqs
        deploy_topolvm
        ;;
    mayastor)
        check_prereqs
        deploy_mayastor
        ;;
    rook-ceph)
        check_prereqs
        deploy_rook_ceph
        ;;
    *)
        echo "用法: $0 <topolvm|mayastor|rook-ceph>"
        exit 1
        ;;
esac

echo ""
echo "================================================"
echo " 部署完成！"
echo " 查看状态:"
echo "   kubectl get sc"
echo "   kubectl get pvc -n test_pv"
echo "   kubectl get pods -n test_pv"
echo "================================================"
