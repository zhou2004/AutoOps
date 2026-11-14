export interface Field {
  key: string;
  label: string;
  type?: string;
  values?: string[];
  options?: string;
  required?: boolean;
  tip?: string;
}
export const names: Record<string, string> = {
  sources: "数据源管理",
  clusters: "集群管理",
  sites: "站点管理",
  upstreams: "上游服务",
  configurations: "配置管理",
  "config-groups": "配置分组",
  "native-configs": "实例配置",
  certificates: "证书管理",
  "certificate-bindings": "证书部署绑定",
  "acme-accounts": "ACME 账户",
  "dns-accounts": "DNS 账户",
  "renewal-policies": "续期策略",
  terminal: "终端",
  logs: "日志管理",
  audit: "Web 操作审计"
};
export const fieldSets: Record<string, Field[]> = {
  sources: [
    {
      key: "hostId",
      label: "主机",
      type: "select",
      options: "hosts",
      required: true
    },
    {
      key: "connectionId",
      label: "SSH 连接",
      type: "select",
      options: "connections",
      required: true
    },
    {
      key: "runtime",
      label: "运行环境",
      type: "select",
      values: ["host", "docker"]
    },
    { key: "container", label: "容器名称" },
    { key: "binary", label: "Nginx 程序路径", required: true },
    { key: "configPath", label: "主配置入口", required: true },
    { key: "prefix", label: "运行前缀", required: true },
    { key: "service", label: "服务名称" },
    { key: "managedPaths", label: "受管目录", type: "lines" },
    { key: "healthUrl", label: "健康检查地址" },
    { key: "healthHost", label: "健康检查 Host" },
    { key: "statusUrl", label: "stub_status 地址" },
    { key: "expectedStatus", label: "预期状态码", type: "number" },
    { key: "sudo", label: "使用 sudo", type: "boolean" },
    { key: "image", label: "固定镜像摘要" },
    { key: "packageVersion", label: "固定软件包版本" },
    { key: "port", label: "业务端口", type: "number" }
  ],
  clusters: [],
  sites: [
    { key: "domains", label: "域名", type: "csv", required: true },
    { key: "ports", label: "监听端口", type: "numbers", required: true },
    { key: "path", label: "配置文件路径", required: true },
    {
      key: "mode",
      label: "编辑方式",
      type: "select",
      values: ["structured", "raw"]
    },
    { key: "tls", label: "启用 HTTPS", type: "boolean" },
    {
      key: "certificateId",
      label: "证书",
      type: "select",
      options: "certificates"
    },
    { key: "maxBodySize", label: "请求体上限" },
    { key: "accessLog", label: "访问日志路径" },
    { key: "healthUrl", label: "站点健康检查地址" },
    { key: "healthHost", label: "健康检查 Host" },
    { key: "expectedStatus", label: "预期状态码", type: "number" }
  ],
  upstreams: [
    { key: "name", label: "上游标识", required: true },
    { key: "path", label: "配置文件路径", required: true },
    {
      key: "strategy",
      label: "负载均衡策略",
      type: "select",
      values: ["round_robin", "least_conn", "ip_hash"]
    }
  ],
  configurations: [],
  "config-groups": [
    { key: "description", label: "分组说明" },
    {
      key: "paths",
      label: "配置文件路径",
      type: "lines",
      tip: "每行一个绝对路径，用于实例配置的分组筛选"
    },
    {
      key: "color",
      label: "标识颜色",
      tip: "使用 #RRGGBB 格式，便于在配置列表中区分"
    }
  ],
  certificates: [
    { key: "domains", label: "域名", type: "csv", required: true },
    {
      key: "owner",
      label: "管理归属",
      type: "select",
      values: ["platform", "external"]
    },
    {
      key: "challenge",
      label: "域名验证",
      type: "select",
      values: ["dns-01", "http-01"]
    },
    {
      key: "acmeAccountId",
      label: "ACME 账户",
      type: "select",
      options: "acme-accounts"
    },
    {
      key: "dnsAccountId",
      label: "DNS 账户",
      type: "select",
      options: "dns-accounts"
    },
    {
      key: "instanceIds",
      label: "HTTP 验证入口",
      type: "multiple",
      options: "instances"
    },
    { key: "challengeRoot", label: "HTTP 挑战根目录" }
  ],
  "certificate-bindings": [
    {
      key: "certificateId",
      label: "证书",
      type: "select",
      options: "certificates",
      required: true
    },
    {
      key: "versionId",
      label: "证书版本",
      type: "number",
      tip: "0 表示最新签发版本"
    },
    { key: "certificatePath", label: "证书部署路径", required: true },
    { key: "keyPath", label: "私钥部署路径", required: true },
    { key: "verifyHost", label: "节点 TLS 核验地址", required: true },
    {
      key: "verifyPort",
      label: "TLS 核验端口",
      type: "number",
      required: true
    },
    { key: "serverName", label: "TLS 核验域名", required: true }
  ],
  "acme-accounts": [
    { key: "email", label: "联系邮箱", required: true },
    { key: "directoryUrl", label: "ACME 服务地址", required: true }
  ],
  "dns-accounts": [
    {
      key: "provider",
      label: "DNS 服务商",
      type: "select",
      values: ["alidns", "tencentcloud", "cloudflare"]
    },
    {
      key: "credentialId",
      label: "配置中心云凭据 ID",
      type: "number",
      tip: "引用已有凭据，或填写下方专用 DNS 凭据"
    }
  ],
  "renewal-policies": [
    {
      key: "certificateId",
      label: "证书",
      type: "select",
      options: "certificates",
      required: true
    },
    { key: "enabled", label: "自动续期", type: "boolean" },
    { key: "autoDeploy", label: "续期后自动部署", type: "boolean" },
    {
      key: "bindingIds",
      label: "自动部署目标",
      type: "multiple",
      options: "certificate-bindings"
    }
  ]
};
export function defaults(kind: string): Record<string, any> {
  return (
    (
      {
        sources: {
          runtime: "host",
          binary: "/usr/sbin/nginx",
          configPath: "/etc/nginx/nginx.conf",
          prefix: "/etc/nginx",
          service: "nginx",
          managedPaths: ["/etc/nginx"],
          expectedStatus: 200,
          port: 80
        },
        clusters: { entries: [] },
        sites: {
          mode: "structured",
          domains: [],
          ports: [80],
          path: "/etc/nginx/conf.d/site.conf",
          locations: [
            {
              path: "/",
              kind: "proxy",
              target: "http://127.0.0.1:8080",
              timeout: 60,
              websocket: false,
              headers: {}
            }
          ]
        },
        upstreams: {
          name: "backend",
          path: "/etc/nginx/conf.d/upstream.conf",
          strategy: "round_robin",
          members: [{ address: "127.0.0.1:8080", weight: 1, backup: false }]
        },
        configurations: {
          files: [],
          baselines: {},
          variables: {}
        },
        "config-groups": { description: "", color: "#409EFF", paths: [] },
        certificates: {
          domains: [],
          owner: "platform",
          challenge: "dns-01",
          instanceIds: []
        },
        "certificate-bindings": {
          verifyHost: "127.0.0.1",
          verifyPort: 443,
          serverName: "",
          versionId: 0,
          certificatePath: "/etc/nginx/autoops-certs/fullchain.pem",
          keyPath: "/etc/nginx/autoops-certs/key.pem"
        },
        "acme-accounts": {
          directoryUrl: "https://acme-staging-v02.api.letsencrypt.org/directory"
        },
        "dns-accounts": { provider: "cloudflare", credentialId: 0 },
        "renewal-policies": { enabled: true, autoDeploy: false, bindingIds: [] }
      } as Record<string, any>
    )[kind] || {}
  );
}
export const statusNames: Record<string, string> = {
  unknown: "未知",
  draft: "草稿",
  configured: "已配置",
  published: "已发布",
  healthy: "正常",
  unavailable: "不可用",
  drifted: "配置漂移",
  degraded: "部分异常",
  issued: "已签发",
  deployed: "已部署",
  partial: "部分部署成功",
  queued: "排队中",
  preflight: "执行前检查",
  running: "执行中",
  verifying: "核验中",
  succeeded: "成功",
  failed: "失败",
  "rolled-back": "已回退",
  observe: "观察模式",
  operate: "运维托管",
  host: "主机",
  docker: "Docker",
  structured: "可视化配置",
  raw: "原文配置",
  platform: "平台托管",
  external: "外部管理"
};
