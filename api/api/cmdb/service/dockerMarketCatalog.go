package service

// marketImage 镜像市场条目。
type marketImage struct {
	Name        string
	Description string
	Category    string
	StarCount   int
}

// marketImages 内置镜像市场清单。
// 名称采用 Docker Hub 风格（官方镜像用短名，第三方用 owner/name）。
// 下载时由前端拼接镜像站前缀。
var marketImages = []marketImage{
	// 基础系统
	{"alpine", "轻量级 Linux 发行版，常用于构建精简容器", "基础系统", 12000},
	{"ubuntu", "最流行的通用 Linux 发行版", "基础系统", 15000},
	{"debian", "稳定可靠的 Linux 发行版", "基础系统", 4800},
	{"centos", "企业级 Linux 发行版（已 EOL）", "基础系统", 5200},
	{"busybox", "极小体积的 Unix 工具箱", "基础系统", 1500},
	{"rockylinux", "CentOS 的社区替代发行版", "基础系统", 1600},

	// Web 服务器
	{"nginx", "高性能 HTTP 与反向代理服务器", "Web 服务器", 21000},
	{"httpd", "Apache HTTP 服务器", "Web 服务器", 5200},
	{"caddy", "自动 HTTPS 的现代 Web 服务器", "Web 服务器", 7000},
	{"traefik", "云原生边缘路由器与反向代理", "Web 服务器", 9000},
	{"tomcat", "Java Servlet 容器", "Web 服务器", 3800},
	{"openresty/openresty", "基于 Nginx 与 Lua 的高性能平台", "Web 服务器", 1500},

	// 数据库
	{"mysql", "最流行的开源关系型数据库", "数据库", 22000},
	{"mariadb", "MySQL 的社区分支", "数据库", 4000},
	{"postgres", "功能强大的开源对象关系型数据库", "数据库", 18000},
	{"redis", "高性能键值内存数据库", "数据库", 21000},
	{"mongo", "文档型 NoSQL 数据库", "数据库", 11000},
	{"elasticsearch", "分布式搜索与分析引擎", "数据库", 12000},
	{"logstash", "日志采集与处理管道", "数据库", 3000},
	{"kibana", "Elasticsearch 可视化面板", "数据库", 5000},
	{"influxdb", "时序数据库", "数据库", 4000},
	{"cassandra", "分布式宽列 NoSQL 数据库", "数据库", 2800},
	{"neo4j", "图数据库", "数据库", 2200},
	{"clickhouse/clickhouse-server", "列式分析型数据库", "数据库", 3800},
	{"etcd", "分布式键值存储", "数据库", 2600},

	// 缓存与消息
	{"memcached", "分布式内存对象缓存", "缓存与消息", 2400},
	{"rabbitmq", "AMQP 消息队列", "缓存与消息", 4800},
	{"kafka", "分布式流处理平台（Apache Kafka）", "缓存与消息", 7000},
	{"zookeeper", "分布式协调服务", "缓存与消息", 3200},
	{"activemq", "Apache 消息中间件", "缓存与消息", 800},

	// 语言运行时
	{"node", "Node.js JavaScript 运行时", "语言运行时", 13000},
	{"python", "Python 解释器运行时", "语言运行时", 9000},
	{"golang", "Go 语言构建与运行环境", "语言运行时", 5000},
	{"openjdk", "OpenJDK Java 运行时", "语言运行时", 7500},
	{"php", "PHP 脚本语言运行环境", "语言运行时", 4200},
	{"ruby", "Ruby 脚本语言运行环境", "语言运行时", 2200},
	{"rust", "Rust 语言构建环境", "语言运行时", 1900},
	{"dotnet", ".NET 运行时", "语言运行时", 1400},

	// 运维与监控
	{"prom/prometheus", "云原生监控与告警系统", "运维与监控", 11000},
	{"grafana/grafana", "可视化监控面板", "运维与监控", 13000},
	{"portainer/portainer-ce", "容器管理 Web UI", "运维与监控", 7000},
	{"prom/node-exporter", "主机指标采集器", "运维与监控", 3200},
	{"prom/alertmanager", "Prometheus 告警管理", "运维与监控", 1800},
	{"grafana/loki", "轻量级日志聚合系统", "运维与监控", 2800},
	{"google/cadvisor", "容器资源使用分析", "运维与监控", 2200},
	{"victoriametrics/victoria-metrics", "高性能时序数据库与监控", "运维与监控", 1500},

	// 应用与中间件
	{"jenkins/jenkins", "持续集成与交付平台", "应用与中间件", 8000},
	{"gitlab/gitlab-ce", "自托管 Git 代码仓库", "应用与中间件", 6000},
	{"gitea/gitea", "轻量级自托管 Git 服务", "应用与中间件", 3000},
	{"sonarqube", "代码质量与安全扫描", "应用与中间件", 2800},
	{"nextcloud", "私有云盘与协作平台", "应用与中间件", 4000},
	{"wordpress", "最流行的博客与建站系统", "应用与中间件", 7000},
	{"vault", "密钥与机密管理", "应用与中间件", 2400},
	{"consul", "服务发现与配置中心", "应用与中间件", 2600},
	{"minio/minio", "S3 兼容对象存储", "应用与中间件", 4800},
	{"nginxinc/nginx-unprivileged", "非 root 运行的 Nginx", "应用与中间件", 800},
	{"jumpserver/jms_all", "开源堡垒机", "应用与中间件", 1300},
	{"nacos/nacos-server", "注册中心与配置中心", "应用与中间件", 2200},
	{"apache/dubbo-admin", "Dubbo 管理控制台", "应用与中间件", 600},
}
