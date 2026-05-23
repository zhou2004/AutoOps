CREATE TABLE IF NOT EXISTS `monitor_alert_group_rule` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `created_at` datetime DEFAULT NULL,
    `updated_at` datetime DEFAULT NULL,
    `deleted_at` datetime DEFAULT NULL,
    `data_source_id` int(10) unsigned DEFAULT NULL COMMENT '数据源id',
    `group_name` varchar(255) DEFAULT NULL COMMENT '规则组名',
    `rule_content` text COMMENT '原生yaml内容',
    `labels` text COMMENT '该组的全局label (JSON格式)',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 修改原有的子表，如果不存在对应新字段则添加，移除旧字段的操作留待手动按需执行防止删库
ALTER TABLE `monitor_alert_rule` ADD COLUMN IF NOT EXISTS `group_id` int(10) unsigned DEFAULT '0' COMMENT '父级组ID';
ALTER TABLE `monitor_alert_rule` ADD COLUMN IF NOT EXISTS `alert` varchar(255) DEFAULT '' COMMENT '告警名称';
ALTER TABLE `monitor_alert_rule` ADD COLUMN IF NOT EXISTS `expr` text COMMENT '告警表达式';
ALTER TABLE `monitor_alert_rule` ADD COLUMN IF NOT EXISTS `for_duration` varchar(64) DEFAULT '' COMMENT '持续时间';
ALTER TABLE `monitor_alert_rule` ADD COLUMN IF NOT EXISTS `severity` varchar(64) DEFAULT '' COMMENT '告警等级';
ALTER TABLE `monitor_alert_rule` ADD COLUMN IF NOT EXISTS `summary` varchar(255) DEFAULT '' COMMENT '告警摘要';
ALTER TABLE `monitor_alert_rule` ADD COLUMN IF NOT EXISTS `description` text COMMENT '告警详细描述';

-- 域名证书监控表
CREATE TABLE IF NOT EXISTS `monitor_domain_cert` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `domain` varchar(255) NOT NULL COMMENT '域名',
    `port` int(11) DEFAULT 443 COMMENT '端口',
    `issuer` varchar(512) DEFAULT '' COMMENT '颁发者',
    `subject` varchar(512) DEFAULT '' COMMENT '主题',
    `not_before` varchar(64) DEFAULT '' COMMENT '起始日期',
    `not_after` varchar(64) DEFAULT '' COMMENT '到期日期',
    `remaining_days` int(11) DEFAULT -1 COMMENT '剩余天数(-1=未知)',
    `status` tinyint(1) DEFAULT 1 COMMENT '状态:1-正常,2-即将过期(<=30天),3-已过期,4-检查失败',
    `check_time` varchar(64) DEFAULT '' COMMENT '最近检查时间',
    `error_msg` text COMMENT '错误信息',
    `create_time` datetime(3) NOT NULL COMMENT '创建时间',
    `update_time` datetime(3) NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_domain` (`domain`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='域名证书监控表';

-- API端点监控表
CREATE TABLE IF NOT EXISTS `monitor_api_endpoint` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` varchar(255) NOT NULL COMMENT '名称',
    `url` varchar(1024) NOT NULL COMMENT '监控URL',
    `method` varchar(16) DEFAULT 'GET' COMMENT '请求方法',
    `headers` json DEFAULT NULL COMMENT '请求头(JSON)',
    `body` text COMMENT '请求体',
    `check_interval` int(11) DEFAULT 300 COMMENT '检查间隔(秒)',
    `timeout` int(11) DEFAULT 10 COMMENT '超时时间(秒)',
    `expected_code` int(11) DEFAULT 200 COMMENT '期望HTTP状态码',
    `expected_body` varchar(512) DEFAULT '' COMMENT '期望响应体包含内容',
    `last_status_code` int(11) DEFAULT 0 COMMENT '最后HTTP状态码',
    `last_response_time` bigint(20) DEFAULT 0 COMMENT '最后响应时间(ms)',
    `status` tinyint(1) DEFAULT 1 COMMENT '状态:1-正常,2-异常,3-超时,4-检查失败',
    `check_time` varchar(64) DEFAULT '' COMMENT '最近检查时间',
    `error_msg` text COMMENT '错误信息',
    `create_time` datetime(3) NOT NULL COMMENT '创建时间',
    `update_time` datetime(3) NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API端点监控表';

-- K8s权限管理表
CREATE TABLE IF NOT EXISTS `k8s_permission` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `user_id` int(10) unsigned NOT NULL COMMENT '用户ID(sys_admin.id)',
    `cluster_id` int(10) unsigned NOT NULL COMMENT '集群ID(k8s_cluster.id)',
    `namespace` varchar(255) NOT NULL COMMENT '命名空间名称',
    `permission_type` varchar(64) DEFAULT 'readonly' COMMENT '权限类型: readonly/write/admin',
    `created_at` datetime(3) NOT NULL COMMENT '创建时间',
    `updated_at` datetime(3) NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_user_cluster_ns` (`user_id`, `cluster_id`, `namespace`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='K8s权限管理表';

-- 故障记录表
CREATE TABLE IF NOT EXISTS `monitor_incident` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `title` varchar(512) NOT NULL COMMENT '故障标题',
    `source` varchar(128) DEFAULT '' COMMENT '来源(domain_cert/api_endpoint/prometheus)',
    `source_id` int(11) DEFAULT 0 COMMENT '来源ID',
    `level` varchar(32) DEFAULT 'warning' COMMENT '告警等级:critical/warning/info',
    `status` varchar(32) DEFAULT 'firing' COMMENT '状态:firing/resolved',
    `description` text COMMENT '描述',
    `alert_time` varchar(64) DEFAULT '' COMMENT '告警时间',
    `resolved_at` varchar(64) DEFAULT '' COMMENT '解决时间',
    `create_time` datetime(3) NOT NULL COMMENT '创建时间',
    `update_time` datetime(3) NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_source` (`source`),
    KEY `idx_status` (`status`),
    KEY `idx_level` (`level`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警故障记录表';
