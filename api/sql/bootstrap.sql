-- AutoOps clean installation. Run only in a new empty MySQL 8 database.

SET NAMES utf8mb4;

SET FOREIGN_KEY_CHECKS=0;

CREATE TABLE IF NOT EXISTS `app_application` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `code` varchar(255) NOT NULL,
  `business_group_id` bigint unsigned NOT NULL,
  `business_dept_id` bigint unsigned NOT NULL,
  `description` text,
  `repo_url` varchar(500) DEFAULT NULL,
  `dev_owners` json DEFAULT NULL,
  `test_owners` json DEFAULT NULL,
  `ops_owners` json DEFAULT NULL,
  `programming_lang` varchar(100) DEFAULT NULL,
  `start_command` text,
  `stop_command` text,
  `health_api` varchar(500) DEFAULT NULL,
  `domains` json DEFAULT NULL,
  `hosts` json DEFAULT NULL,
  `databases` json DEFAULT NULL,
  `other_res` json DEFAULT NULL,
  `status` tinyint DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_app_application_code` (`code`),
  KEY `idx_app_application_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `app_jenkins_env` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint unsigned NOT NULL,
  `env_name` varchar(50) NOT NULL,
  `jenkins_server_id` bigint unsigned DEFAULT NULL,
  `job_name` varchar(255) DEFAULT '',
  `job_url` varchar(500) DEFAULT NULL,
  `build_params` json DEFAULT NULL,
  `deploy_config` json DEFAULT NULL,
  `notification` json DEFAULT NULL,
  `is_active` tinyint DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_app_jenkins_env_app_id` (`app_id`),
  KEY `idx_app_jenkins_env_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_app_application_jenkins_envs` FOREIGN KEY (`app_id`) REFERENCES `app_application` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `app_service_release` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL COMMENT '发布标题',
  `business_group_id` bigint unsigned NOT NULL COMMENT '业务组ID',
  `impact_feature` text COMMENT '影响功能描述',
  `applicant_id` bigint unsigned NOT NULL COMMENT '申请人ID',
  `applicant_name` varchar(100) NOT NULL COMMENT '申请人姓名',
  `owner_approver_id` bigint unsigned DEFAULT NULL COMMENT '负责人审批人ID',
  `owner_approver_name` varchar(100) DEFAULT NULL COMMENT '负责人审批人姓名',
  `security_approver_id` bigint unsigned DEFAULT NULL COMMENT '安全审批人ID',
  `security_approver_name` varchar(100) DEFAULT NULL COMMENT '安全审批人姓名',
  `test_approver_id` bigint unsigned DEFAULT NULL COMMENT '测试审批人ID',
  `test_approver_name` varchar(100) DEFAULT NULL COMMENT '测试审批人姓名',
  `owner_approval_status` bigint DEFAULT '1' COMMENT '负责人审批状态',
  `security_approval_status` bigint DEFAULT '1' COMMENT '安全审批状态',
  `test_approval_status` bigint DEFAULT '1' COMMENT '测试审批状态',
  `owner_approval_time` datetime(3) DEFAULT NULL COMMENT '负责人审批时间',
  `security_approval_time` datetime(3) DEFAULT NULL COMMENT '安全审批时间',
  `test_approval_time` datetime(3) DEFAULT NULL COMMENT '测试审批时间',
  `owner_approval_remark` text COMMENT '负责人审批意见',
  `security_approval_remark` text COMMENT '安全审批意见',
  `test_approval_remark` text COMMENT '测试审批意见',
  `deploy_status` bigint DEFAULT '1' COMMENT '运维发布状态',
  `regression_test_status` bigint DEFAULT '1' COMMENT '回归测试状态',
  `status` bigint DEFAULT '1' COMMENT '流程状态',
  `start_time` datetime(3) DEFAULT NULL COMMENT '发布开始时间',
  `end_time` datetime(3) DEFAULT NULL COMMENT '发布结束时间',
  `duration` bigint DEFAULT '0' COMMENT '发布耗时(秒)',
  `service_count` bigint DEFAULT '0' COMMENT '关联服务数量',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_app_service_release_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `app_service_release_item` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `release_id` bigint unsigned NOT NULL COMMENT '上线工单ID',
  `app_id` bigint unsigned NOT NULL COMMENT '应用ID',
  `app_name` varchar(255) NOT NULL COMMENT '应用名称',
  `app_code` varchar(100) NOT NULL COMMENT '应用编码',
  `project_name` varchar(255) NOT NULL COMMENT '项目服务名称',
  `repo_url` varchar(500) NOT NULL COMMENT '项目地址',
  `branch` varchar(100) DEFAULT 'master' COMMENT '发布分支',
  `commit_id` varchar(100) NOT NULL COMMENT 'Commit ID',
  `impact_feature` text COMMENT '影响功能',
  `function_module` text COMMENT '功能模块',
  `db_change` text COMMENT '数据库变更',
  `config_change` text COMMENT '配置变更',
  `remark` text COMMENT '备注信息',
  `jenkins_env_id` bigint unsigned DEFAULT NULL COMMENT 'Jenkins环境配置ID',
  `jenkins_job_url` varchar(500) DEFAULT NULL COMMENT 'Jenkins任务URL',
  `parameters` text COMMENT 'Jenkins构建参数(JSON格式)',
  `build_number` bigint DEFAULT '0' COMMENT '构建编号',
  `log_url` varchar(500) DEFAULT NULL COMMENT '构建日志URL',
  `status` bigint DEFAULT '1' COMMENT '发布状态',
  `start_time` datetime(3) DEFAULT NULL COMMENT '发布开始时间',
  `end_time` datetime(3) DEFAULT NULL COMMENT '发布结束时间',
  `duration` bigint DEFAULT '0' COMMENT '发布耗时(秒)',
  `error_message` text COMMENT '错误信息',
  `execute_order` bigint DEFAULT '0' COMMENT '执行顺序',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_app_service_release_item_release_id` (`release_id`),
  KEY `idx_app_service_release_item_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `app_sh_release` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL COMMENT '上线标题',
  `reason` text NOT NULL COMMENT '上线原因',
  `business_group_id` bigint unsigned NOT NULL COMMENT '业务线ID',
  `app_id` bigint unsigned NOT NULL COMMENT '服务ID',
  `app_name` varchar(255) NOT NULL COMMENT '服务名称',
  `app_code` varchar(100) NOT NULL COMMENT '服务编码',
  `applicant_id` bigint unsigned NOT NULL COMMENT '申请人ID',
  `applicant_name` varchar(100) NOT NULL COMMENT '申请人姓名',
  `approver_id` bigint unsigned DEFAULT NULL COMMENT '审核人ID',
  `approver_name` varchar(100) DEFAULT NULL COMMENT '审核人姓名',
  `executor_id` bigint unsigned DEFAULT NULL COMMENT '执行人ID',
  `executor_name` varchar(100) DEFAULT NULL COMMENT '执行人姓名',
  `execute_dir` varchar(500) NOT NULL COMMENT '执行目录',
  `script_content` text NOT NULL COMMENT '脚本内容',
  `approval_status` bigint DEFAULT '1' COMMENT '审核状态',
  `approval_time` datetime(3) DEFAULT NULL COMMENT '审核时间',
  `approval_remark` text COMMENT '审核意见',
  `execute_status` bigint DEFAULT '1' COMMENT '执行状态',
  `status` bigint DEFAULT '1' COMMENT '流程状态',
  `start_time` datetime(3) DEFAULT NULL COMMENT '脚本执行开始时间',
  `end_time` datetime(3) DEFAULT NULL COMMENT '脚本执行结束时间',
  `duration` bigint DEFAULT '0' COMMENT '执行耗时(秒)',
  `jenkins_env_id` bigint unsigned DEFAULT NULL COMMENT 'Jenkins环境配置ID',
  `build_number` bigint DEFAULT '0' COMMENT '构建编号',
  `log_url` varchar(500) DEFAULT NULL COMMENT '构建日志URL',
  `error_message` text COMMENT '错误信息',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `parameters` text COMMENT 'Jenkins构建参数(JSON格式)',
  `server_host_id` bigint unsigned NOT NULL COMMENT '执行服务器主机ID(关联cmdb_host)',
  `pull_code_start_time` datetime(3) DEFAULT NULL COMMENT '拉取代码开始时间',
  `pull_code_end_time` datetime(3) DEFAULT NULL COMMENT '拉取代码结束时间',
  `script_output` longtext COMMENT '脚本执行输出',
  PRIMARY KEY (`id`),
  KEY `idx_app_sh_release_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_asset_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) NOT NULL COMMENT '''授权规则名称''',
  `description` text COMMENT '''描述''',
  `user_ids` text COMMENT '''授权用户ID列表(JSON数组)''',
  `group_ids` text COMMENT '''授权用户组ID列表(JSON数组)''',
  `asset_types` text COMMENT '''授权资产类型(JSON数组: host/physical/network/database)''',
  `host_group_ids` text COMMENT '''授权主机分组ID列表(JSON数组)''',
  `physical_ids` text COMMENT '''授权物理机ID列表(JSON数组)''',
  `network_ids` text COMMENT '''授权网络设备ID列表(JSON数组)''',
  `database_ids` text COMMENT '''授权数据库ID列表(JSON数组)''',
  `idc_ids` text COMMENT '''授权机房ID列表(JSON数组,含其下所有资产)''',
  `permission_actions` text COMMENT '''权限操作(JSON数组: get/list/connect/create/update/delete/admin)''',
  `is_active` bigint DEFAULT '1' COMMENT '''是否启用:0-禁用,1-启用''',
  `date_start` varchar(20) DEFAULT NULL COMMENT '''有效期开始(YYYY-MM-DD)''',
  `date_expired` varchar(20) DEFAULT NULL COMMENT '''有效期结束(YYYY-MM-DD)''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `host_ids` text COMMENT '''授权具体主机ID列表(JSON数组)''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_asset_permission_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_cabinet` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(100) NOT NULL COMMENT '''机柜名称/编号''',
  `idc_id` bigint unsigned NOT NULL COMMENT '''所属机房ID''',
  `position` varchar(100) DEFAULT NULL COMMENT '''位置描述(如A列3排)''',
  `unit_num` bigint DEFAULT '42' COMMENT '''机柜U数(如42U)''',
  `used_unit` bigint DEFAULT '0' COMMENT '''已用U位数''',
  `power_kw` double DEFAULT '0' COMMENT '''额定功率(KW)''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1-启用,2-停用''',
  `remark` text COMMENT '''备注''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `id_c_id` bigint unsigned NOT NULL COMMENT '''所属机房ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_cabinet_name` (`name`),
  KEY `idx_cmdb_cabinet_idc` (`idc_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='CMDB机柜表';

CREATE TABLE IF NOT EXISTS `cmdb_credential_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) NOT NULL COMMENT '''授权规则名称''',
  `credential_id` bigint unsigned NOT NULL COMMENT '''凭据ID''',
  `user_ids` text COMMENT '''授权用户ID列表(JSON数组)''',
  `group_ids` text COMMENT '''授权用户组ID列表(JSON数组)''',
  `is_active` bigint DEFAULT '1' COMMENT '''是否启用:0-禁用,1-启用''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '''父级分组ID''',
  `name` longtext NOT NULL COMMENT '''分组名称''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `remark` longtext COMMENT '''备注''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_host` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `host_name` longtext NOT NULL COMMENT '''名称''',
  `group_id` bigint unsigned NOT NULL COMMENT '''分组ID''',
  `private_ip` longtext COMMENT '''私网IP''',
  `public_ip` longtext COMMENT '''公网IP''',
  `ssh_name` longtext COMMENT '''SSH用户名''',
  `ssh_key_id` bigint unsigned DEFAULT NULL COMMENT '''SSH凭据ID''',
  `ssh_port` bigint DEFAULT '22' COMMENT '''SSH端口''',
  `remark` longtext COMMENT '''备注''',
  `vendor` bigint DEFAULT NULL COMMENT '''1->自建,2->阿里云,3->腾讯云''',
  `region` longtext COMMENT '''区域''',
  `instance_id` longtext COMMENT '''实例ID''',
  `os` longtext COMMENT '''操作系统''',
  `status` bigint DEFAULT NULL COMMENT '''状态:1->认证成功,2->未认证,3->认证失败''',
  `cpu` longtext COMMENT '''CPU信息''',
  `memory` longtext COMMENT '''内存信息''',
  `disk` longtext COMMENT '''磁盘信息''',
  `billing_type` longtext COMMENT '''计费方式''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `expire_time` datetime(3) DEFAULT NULL COMMENT '''到期时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `ssh_ip` longtext NOT NULL COMMENT '''SSH连接IP''',
  `name` longtext NOT NULL COMMENT '''ecs主机名称''',
  `ssh_gateway_id` bigint unsigned DEFAULT NULL COMMENT '''中转网关凭据ID''',
  `tag` text COMMENT '''标签(格式:key=value,key=value)''',
  PRIMARY KEY (`id`),
  KEY `fk_cmdb_group_hosts` (`group_id`),
  CONSTRAINT `fk_cmdb_group_hosts` FOREIGN KEY (`group_id`) REFERENCES `cmdb_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_idc` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) NOT NULL COMMENT '''机房名称''',
  `short_name` varchar(50) DEFAULT NULL COMMENT '''机房简称''',
  `address` varchar(255) DEFAULT NULL COMMENT '''机房地址''',
  `contact` varchar(50) DEFAULT NULL COMMENT '''联系人''',
  `phone` varchar(30) DEFAULT NULL COMMENT '''联系电话''',
  `level` varchar(20) DEFAULT NULL COMMENT '''机房等级(T1-T4)''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1-启用,2-停用''',
  `description` text COMMENT '''描述''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_idc_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_network_device` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `sn` varchar(128) DEFAULT NULL COMMENT '''序列号/SN''',
  `name` varchar(100) NOT NULL COMMENT '''设备名称''',
  `device_type` bigint NOT NULL COMMENT '''设备类型:1-路由器,2-交换机,3-防火墙,4-负载均衡,5-其他''',
  `brand` varchar(50) DEFAULT NULL COMMENT '''品牌(Cisco/Huawei/H3C等)''',
  `model` varchar(100) DEFAULT NULL COMMENT '''型号''',
  `manage_ip` varchar(64) DEFAULT NULL COMMENT '''管理IP''',
  `version` varchar(100) DEFAULT NULL COMMENT '''固件/系统版本''',
  `port_num` bigint DEFAULT '24' COMMENT '''端口数量''',
  `id_c_id` bigint unsigned DEFAULT NULL COMMENT '''所属机房ID''',
  `cabinet_id` bigint unsigned DEFAULT NULL COMMENT '''所属机柜ID''',
  `asset_status` bigint DEFAULT '1' COMMENT '''资产状态:1-在库,2-已上架,3-维修中,4-已下架,5-报废''',
  `purchase_date` varchar(20) DEFAULT NULL COMMENT '''采购日期''',
  `warranty_date` varchar(20) DEFAULT NULL COMMENT '''维保到期''',
  `status` bigint DEFAULT '1' COMMENT '''运行状态:1-运行中,2-关机,3-离线''',
  `remark` text COMMENT '''备注''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `idc_id` bigint unsigned NOT NULL COMMENT '''所属机房ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_network_device_sn` (`sn`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_physical_machine` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `sn` varchar(128) DEFAULT NULL COMMENT '''序列号/SN''',
  `host_name` varchar(100) DEFAULT NULL COMMENT '''主机名''',
  `manage_ip` varchar(64) DEFAULT NULL COMMENT '''管理IP(BMC/iLO/iDRAC)''',
  `business_ip` varchar(64) DEFAULT NULL COMMENT '''业务IP''',
  `brand` varchar(50) DEFAULT NULL COMMENT '''品牌(Dell/HP/Inspur等)''',
  `model` varchar(100) DEFAULT NULL COMMENT '''型号(R750/DL380等)''',
  `cpu` varchar(100) DEFAULT NULL COMMENT '''CPU信息''',
  `memory` varchar(100) DEFAULT NULL COMMENT '''内存信息(GB)''',
  `disk` varchar(255) DEFAULT NULL COMMENT '''磁盘信息''',
  `raid` varchar(100) DEFAULT NULL COMMENT '''RAID类型''',
  `idc_id` bigint unsigned NOT NULL COMMENT '''所属机房ID''',
  `cabinet_id` bigint unsigned DEFAULT NULL COMMENT '''所属机柜ID''',
  `unit_position` bigint DEFAULT '0' COMMENT '''机柜U位(起始)''',
  `asset_status` bigint DEFAULT '1' COMMENT '''资产状态:1-在库,2-已上架,3-维修中,4-已下架,5-报废''',
  `purchase_date` varchar(20) DEFAULT NULL COMMENT '''采购日期''',
  `warranty_date` varchar(20) DEFAULT NULL COMMENT '''维保到期''',
  `vendor` varchar(50) DEFAULT NULL COMMENT '''供应商''',
  `status` bigint DEFAULT '1' COMMENT '''运行状态:1-运行中,2-关机,3-离线''',
  `remark` text COMMENT '''备注''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `id_c_id` bigint unsigned DEFAULT NULL COMMENT '''所属机房ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_physical_machine_sn` (`sn`),
  KEY `idx_cmdb_physical_idc` (`idc_id`),
  KEY `idx_cmdb_physical_cabinet` (`cabinet_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='CMDB物理机表';

CREATE TABLE IF NOT EXISTS `cmdb_sql` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `type` int NOT NULL,
  `account_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  `tags` varchar(255) DEFAULT NULL,
  `description` varchar(500) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_sql_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `instance_id` varchar(64) NOT NULL,
  `database` varchar(128) NOT NULL,
  `operation_type` varchar(32) NOT NULL,
  `sql_content` text NOT NULL,
  `exec_user` varchar(64) NOT NULL,
  `ip` varchar(64) NOT NULL,
  `scanned_rows` bigint DEFAULT '0',
  `affected_rows` bigint DEFAULT '0',
  `execution_time` bigint DEFAULT '0',
  `returned_rows` bigint DEFAULT '0',
  `result` varchar(32) NOT NULL,
  `query_time` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_cmdb_sql_log_query_time` (`query_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_sql_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `instance_id` varchar(64) NOT NULL,
  `database` varchar(128) NOT NULL,
  `operation_type` varchar(32) NOT NULL,
  `sql_content` text NOT NULL,
  `exec_user` varchar(64) NOT NULL,
  `scanned_rows` bigint DEFAULT '0',
  `affected_rows` bigint DEFAULT '0',
  `execution_time` bigint DEFAULT '0',
  `returned_rows` bigint DEFAULT '0',
  `result` varchar(32) NOT NULL,
  `query_time` datetime(3) NOT NULL,
  `name` varchar(64) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_cmdb_sql_records_query_time` (`query_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_user_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(255) NOT NULL COMMENT '''用户组名称''',
  `code` varchar(128) DEFAULT '' COMMENT '''用户组编码''',
  `description` varchar(512) DEFAULT '' COMMENT '''描述''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1-启用,0-禁用''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_user_group_name` (`name`),
  KEY `idx_cmdb_user_group_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `cmdb_user_group_member` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `group_id` bigint unsigned NOT NULL COMMENT '''用户组ID''',
  `user_id` bigint unsigned NOT NULL COMMENT '''用户ID''',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_ugm_group_user` (`group_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `config_account` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `alias` varchar(128) NOT NULL,
  `host` varchar(128) NOT NULL,
  `port` bigint NOT NULL,
  `name` varchar(128) NOT NULL,
  `password` text NOT NULL,
  `type` bigint NOT NULL,
  `remark` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `config_ansible` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''配置名称''',
  `type` bigint NOT NULL COMMENT '''1-inventory 2-global_vars 3-extra_vars 4-cli_args''',
  `content` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''内容：inventory为文本，vars/args为JSON''',
  `remark` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '''备注''',
  `created_by` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '''创建人''',
  `updated_by` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '''更新人''',
  `created_at` datetime(3) NOT NULL COMMENT '''创建时间''',
  `updated_at` datetime(3) NOT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_config_ansible_name` (`name`),
  KEY `idx_config_ansible_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `config_ecsauth` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `name` longtext NOT NULL COMMENT '''凭证名称''',
  `type` bigint NOT NULL COMMENT '''认证类型:1->密码,2->私钥,3->公钥(免认证)''',
  `username` longtext COMMENT '''用户名''',
  `password` longtext COMMENT '''密码(type=1时使用)''',
  `public_key` text COMMENT '''私钥内容(type=2时使用，字段名历史原因)''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `remark` longtext COMMENT '''备注''',
  `port` bigint DEFAULT '22' COMMENT '''端口号''',
  `bastion_host` longtext COMMENT '''跳板机IP''',
  `bastion_port` bigint DEFAULT '22' COMMENT '''跳板机端口''',
  `bastion_username` longtext COMMENT '''跳板机用户名''',
  `bastion_auth_type` bigint DEFAULT NULL COMMENT '''跳板机认证类型:1->密码,2->私钥,3->免密''',
  `bastion_password` longtext COMMENT '''跳板机密码''',
  `bastion_private_key` text COMMENT '''跳板机私钥''',
  `target_auth_type` bigint DEFAULT NULL COMMENT '''目标机器认证类型:1->密码,2->私钥,3->免密''',
  `target_password` longtext COMMENT '''目标机器密码''',
  `target_private_key` text COMMENT '''目标机器私钥''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `config_keymanage` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `key_type` bigint NOT NULL,
  `key_id` text NOT NULL,
  `key_secret` text NOT NULL,
  `remark` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `key_name` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `config_sync_schedule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `cron_expr` varchar(100) NOT NULL,
  `key_types` text NOT NULL,
  `status` bigint NOT NULL DEFAULT '1',
  `last_run_time` datetime(3) DEFAULT NULL,
  `next_run_time` datetime(3) DEFAULT NULL,
  `remark` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `sync_log` text,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `code` varchar(36) NOT NULL COMMENT '''数据库编码''',
  `name` varchar(100) NOT NULL COMMENT '''数据库名称''',
  `database` varchar(500) NOT NULL COMMENT '''数据库名(多个用空格分隔)''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `instance_id` bigint unsigned NOT NULL COMMENT '''实例ID''',
  `instance_code` varchar(36) DEFAULT NULL COMMENT '''实例编码''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1->启用,2->禁用''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  `modifier` varchar(64) DEFAULT NULL COMMENT '''修改人''',
  `modifier_id` bigint unsigned DEFAULT NULL COMMENT '''修改人ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_db_code` (`code`),
  KEY `idx_db_instance_id` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db_es_instance` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `code` varchar(36) NOT NULL COMMENT '''实例编码''',
  `name` varchar(100) NOT NULL COMMENT '''实例名称''',
  `protocol` varchar(10) DEFAULT 'http' COMMENT '''协议:http|https''',
  `host` varchar(255) NOT NULL COMMENT '''主机''',
  `port` bigint DEFAULT '9200' COMMENT '''端口''',
  `username` varchar(100) DEFAULT NULL COMMENT '''用户名(可选)''',
  `password` varchar(500) DEFAULT '' COMMENT '''密码(加密)''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `ssh_tunnel_machine_id` bigint unsigned DEFAULT '0' COMMENT '''SSH隧道机器ID''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1->启用,2->禁用''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  `modifier` varchar(64) DEFAULT NULL COMMENT '''修改人''',
  `modifier_id` bigint unsigned DEFAULT NULL COMMENT '''修改人ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_db_es_instance_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db_export_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `task_id` varchar(36) NOT NULL COMMENT '''任务ID''',
  `db_id` bigint unsigned NOT NULL COMMENT '''数据库ID''',
  `db_name` varchar(100) NOT NULL COMMENT '''数据库名''',
  `export_type` varchar(20) NOT NULL COMMENT '''导出类型:structure/full''',
  `status` varchar(20) NOT NULL COMMENT '''状态''',
  `file_path` varchar(500) DEFAULT NULL COMMENT '''文件路径''',
  `file_size` bigint DEFAULT NULL COMMENT '''文件大小(字节)''',
  `error_message` text COMMENT '''错误信息''',
  `start_time` datetime(3) DEFAULT NULL COMMENT '''开始时间''',
  `end_time` datetime(3) DEFAULT NULL COMMENT '''结束时间''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_db_export_task_task_id` (`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db_instance` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `code` varchar(36) NOT NULL COMMENT '''实例编码''',
  `name` varchar(100) NOT NULL COMMENT '''实例名称''',
  `type` varchar(20) NOT NULL COMMENT '''数据库类型:mysql,postgres,oracle等''',
  `host` varchar(100) NOT NULL COMMENT '''主机地址''',
  `port` bigint NOT NULL COMMENT '''端口''',
  `network` varchar(20) DEFAULT 'tcp' COMMENT '''网络类型:tcp,unix''',
  `params` varchar(500) DEFAULT NULL COMMENT '''连接参数''',
  `username` varchar(100) NOT NULL COMMENT '''用户名''',
  `password` varchar(500) NOT NULL COMMENT '''密码(加密)''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `ssh_tunnel_machine_id` bigint unsigned DEFAULT '0' COMMENT '''SSH隧道机器ID''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1->启用,2->禁用''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  `modifier` varchar(64) DEFAULT NULL COMMENT '''修改人''',
  `modifier_id` bigint unsigned DEFAULT NULL COMMENT '''修改人ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_db_instance_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db_mongo_instance` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `code` varchar(36) NOT NULL COMMENT '''实例编码''',
  `name` varchar(100) NOT NULL COMMENT '''实例名称''',
  `uri` varchar(500) NOT NULL COMMENT '''连接URI''',
  `ssh_tunnel_machine_id` bigint unsigned DEFAULT '0' COMMENT '''SSH隧道机器ID''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1->启用,2->禁用''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  `modifier` varchar(64) DEFAULT NULL COMMENT '''修改人''',
  `modifier_id` bigint unsigned DEFAULT NULL COMMENT '''修改人ID''',
  `type` varchar(20) DEFAULT 'mongodb' COMMENT '''MongoDB类型:mongodb,mongodb-atlas等''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_db_mongo_instance_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db_redis_instance` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `code` varchar(36) NOT NULL COMMENT '''实例编码''',
  `name` varchar(100) NOT NULL COMMENT '''实例名称''',
  `mode` varchar(20) NOT NULL COMMENT '''模式:standalone,cluster,sentinel''',
  `host` varchar(300) NOT NULL COMMENT '''主机: standalone为host:port, cluster为逗号分隔, sentinel为master=hosts''',
  `port` bigint DEFAULT '0' COMMENT '''端口(standalone可用)''',
  `db` bigint DEFAULT '0' COMMENT '''默认库号''',
  `username` varchar(100) DEFAULT NULL COMMENT '''用户名(可选)''',
  `password` varchar(500) NOT NULL COMMENT '''密码(加密)''',
  `redis_node_password` varchar(500) DEFAULT '' COMMENT '''节点密码(仅sentinel)''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `ssh_tunnel_machine_id` bigint unsigned DEFAULT '0' COMMENT '''SSH隧道机器ID''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1->启用,2->禁用''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  `modifier` varchar(64) DEFAULT NULL COMMENT '''修改人''',
  `modifier_id` bigint unsigned DEFAULT NULL COMMENT '''修改人ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_db_redis_instance_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db_sql` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `db_id` bigint unsigned NOT NULL COMMENT '''数据库ID''',
  `db` varchar(100) NOT NULL COMMENT '''数据库名''',
  `name` varchar(100) NOT NULL COMMENT '''SQL名称''',
  `type` bigint DEFAULT '1' COMMENT '''类型:1->查询,2->更新''',
  `sql` text NOT NULL COMMENT '''SQL语句''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  `modifier` varchar(64) DEFAULT NULL COMMENT '''修改人''',
  `modifier_id` bigint unsigned DEFAULT NULL COMMENT '''修改人ID''',
  PRIMARY KEY (`id`),
  KEY `idx_db_sql_db_id` (`db_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `db_sql_exec` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `db_id` bigint unsigned NOT NULL COMMENT '''数据库ID''',
  `db_name` varchar(100) NOT NULL COMMENT '''数据库名''',
  `table_name` varchar(100) DEFAULT NULL COMMENT '''表名''',
  `type` tinyint NOT NULL COMMENT '''类型:1->查询,2->插入,3->更新,4->删除,5->DDL''',
  `sql` text NOT NULL COMMENT '''SQL语句''',
  `old_value` longtext COMMENT '''旧值(用于回滚)''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `status` tinyint NOT NULL COMMENT '''状态:1->成功,2->失败''',
  `res` text COMMENT '''执行结果''',
  `exec_time` bigint DEFAULT NULL COMMENT '''执行时长(ms)''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  PRIMARY KEY (`id`),
  KEY `idx_db_sql_exec_db_id` (`db_id`),
  KEY `idx_db_sql_exec_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `k8s_cluster` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) NOT NULL COMMENT '''集群名称''',
  `version` varchar(50) NOT NULL COMMENT '''集群版本''',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '''集群状态:1-创建中,2-运行中,3-离线''',
  `credential` text COMMENT '''集群凭证(kubeconfig)''',
  `description` text COMMENT '''集群描述''',
  `cluster_type` bigint NOT NULL DEFAULT '1' COMMENT '''集群类型:1-自建,2-导入''',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `node_count` bigint DEFAULT '0' COMMENT '''节点数量''',
  `ready_nodes` bigint DEFAULT '0' COMMENT '''就绪节点数''',
  `master_nodes` bigint DEFAULT '0' COMMENT '''Master节点数''',
  `worker_nodes` bigint DEFAULT '0' COMMENT '''Worker节点数''',
  `last_sync_at` datetime(3) DEFAULT NULL COMMENT '''最后同步时间''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_k8s_cluster_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `k8s_group_permission` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `group_id` int unsigned NOT NULL COMMENT '用户组ID(k8s_user_group.id)',
  `cluster_id` int unsigned NOT NULL COMMENT '集群ID(k8s_cluster.id)',
  `namespace` varchar(255) NOT NULL COMMENT '命名空间名称',
  `permission_type` varchar(64) DEFAULT 'readonly' COMMENT '权限类型: readonly/write/admin',
  `created_at` datetime(3) NOT NULL COMMENT '创建时间',
  `updated_at` datetime(3) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_group_cluster_ns` (`group_id`,`cluster_id`,`namespace`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s用户组权限表';

CREATE TABLE IF NOT EXISTS `k8s_permission` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` int unsigned NOT NULL COMMENT '用户ID(sys_admin.id)',
  `cluster_id` int unsigned NOT NULL COMMENT '集群ID(k8s_cluster.id)',
  `namespace` varchar(255) NOT NULL COMMENT '命名空间名称',
  `permission_type` varchar(64) DEFAULT 'readonly' COMMENT '权限类型: readonly/write/admin',
  `created_at` datetime(3) NOT NULL COMMENT '创建时间',
  `updated_at` datetime(3) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_cluster_ns` (`user_id`,`cluster_id`,`namespace`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s权限管理表';

CREATE TABLE IF NOT EXISTS `k8s_rbac_binding` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL,
  `namespace` varchar(255) DEFAULT '',
  `role_id` bigint unsigned NOT NULL,
  `subject_type` varchar(32) NOT NULL,
  `subject_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `k8s_rbac_role` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL,
  `namespace` varchar(255) DEFAULT '',
  `name` varchar(255) NOT NULL,
  `rules` json NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `k8s_user_group` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(255) NOT NULL COMMENT '用户组名称',
  `code` varchar(128) DEFAULT '' COMMENT '用户组编码(唯一标识)',
  `description` varchar(512) DEFAULT '' COMMENT '描述',
  `status` tinyint(1) DEFAULT '1' COMMENT '状态:1-启用,0-禁用',
  `created_at` datetime(3) NOT NULL COMMENT '创建时间',
  `updated_at` datetime(3) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`),
  KEY `idx_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s用户组表';

CREATE TABLE IF NOT EXISTS `k8s_user_group_member` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `group_id` int unsigned NOT NULL COMMENT '用户组ID(k8s_user_group.id)',
  `user_id` int unsigned NOT NULL COMMENT '用户ID(sys_admin.id)',
  `created_at` datetime(3) NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_group_user` (`group_id`,`user_id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s用户组成员关系表';

CREATE TABLE IF NOT EXISTS `monitor_agent` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `host_id` bigint unsigned NOT NULL COMMENT '''主机ID''',
  `host_name` longtext COMMENT '''主机名称''',
  `version` varchar(191) DEFAULT '1.0.0' COMMENT '''Agent版本''',
  `status` bigint DEFAULT NULL COMMENT '''状态:1->部署中,2->部署失败,3->运行中,4->已停止''',
  `install_path` longtext COMMENT '''安装路径''',
  `port` bigint DEFAULT '9100' COMMENT '''监听端口''',
  `pid` bigint DEFAULT NULL COMMENT '''进程ID''',
  `last_heartbeat` datetime(3) DEFAULT NULL COMMENT '''最后心跳时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `error_msg` text COMMENT '''错误信息''',
  `install_progress` bigint DEFAULT '0' COMMENT '''安装进度(0-100)''',
  `node_exporter_url` varchar(512) DEFAULT '' COMMENT '''node_exporter端点URL''',
  `node_exporter_status` tinyint(1) DEFAULT '0' COMMENT '''node_exporter状态:0-未扫描,1-在线,2-离线,3-未安装''',
  `node_exporter_port` bigint DEFAULT '9100' COMMENT '''node_exporter端口''',
  `node_exporter_scan_time` varchar(64) DEFAULT '' COMMENT '''最近扫描时间''',
  PRIMARY KEY (`id`),
  KEY `idx_monitor_agent_host_id` (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_alert_config` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `conf_key` varchar(255) DEFAULT NULL,
  `conf_value` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_monitor_alert_config_conf_key` (`conf_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_alert_group_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `data_source_id` bigint unsigned DEFAULT NULL COMMENT '数据源id',
  `group_name` varchar(255) DEFAULT NULL COMMENT '规则组名',
  `rule_content` text COMMENT '原生yaml内容',
  `labels` text COMMENT '该组的全局label (JSON格式)',
  PRIMARY KEY (`id`),
  KEY `idx_monitor_alert_group_rule_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_alert_record` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `alertname` longtext,
  `alert_level` longtext,
  `labels` longtext,
  `instance` longtext,
  `starts_at` longtext,
  `ends_at` longtext,
  `summary` longtext,
  `description` longtext,
  `alert_status` longtext,
  `created_time` datetime(3) DEFAULT NULL,
  `updated_by` longtext,
  `updated_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_alert_router` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` longtext,
  `tpl_id` bigint DEFAULT NULL,
  `rules` longtext,
  `url_or_phone` longtext,
  `at_some_one` longtext,
  `at_some_one_rr` tinyint(1) DEFAULT NULL,
  `send_resolved` tinyint(1) DEFAULT NULL,
  `created` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_alert_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `group_id` bigint unsigned DEFAULT NULL COMMENT '组ID',
  `alert` varchar(255) DEFAULT NULL COMMENT '告警名称',
  `expr` text COMMENT '告警表达式',
  `for_duration` varchar(64) DEFAULT NULL COMMENT '持续时间',
  `labels` text COMMENT '规则私有label (JSON格式)',
  `constraints` text COMMENT '规则约束条件 (JSON格式)',
  `severity` varchar(64) DEFAULT NULL COMMENT '严重程度',
  `summary` varchar(255) DEFAULT NULL COMMENT '摘要',
  `description` text COMMENT '描述',
  `rule_content` text COMMENT '单个规则的完整yaml',
  `status` varchar(64) DEFAULT 'inactive' COMMENT '运行状态',
  `style` varchar(64) DEFAULT NULL COMMENT '规则分类(如CPU,Memory)',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用(1启用,0禁用)',
  PRIMARY KEY (`id`),
  KEY `idx_monitor_alert_rule_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_alert_rule_style` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(64) NOT NULL COMMENT '分类名称(如CPU,Memory)',
  `description` varchar(255) DEFAULT NULL COMMENT '描述',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_monitor_alert_rule_style_name` (`name`),
  KEY `idx_monitor_alert_rule_style_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_aliyun_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''配置名称''',
  `access_key` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''AccessKey''',
  `access_secret` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''AccessSecret''',
  `region` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'cn-hangzhou' COMMENT '''区域''',
  `email` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''联系邮箱''',
  `phone` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''联系电话''',
  `username` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''联系人姓名''',
  `eab_kid` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''ACME EAB Key ID (ZeroSSL等需要)''',
  `eab_hmac_key` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''ACME EAB HMAC Key (ZeroSSL等需要)''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1->启用,0->禁用''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `monitor_api_endpoint` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(255) NOT NULL COMMENT '名称',
  `url` varchar(1024) NOT NULL COMMENT '监控URL',
  `method` varchar(16) DEFAULT 'GET' COMMENT '请求方法',
  `headers` json DEFAULT NULL COMMENT '请求头(JSON)',
  `body` text COMMENT '请求体',
  `check_interval` int DEFAULT '300' COMMENT '检查间隔(秒)',
  `timeout` int DEFAULT '10' COMMENT '超时时间(秒)',
  `expected_code` int DEFAULT '200' COMMENT '期望HTTP状态码',
  `expected_body` varchar(512) DEFAULT '' COMMENT '期望响应体包含内容',
  `last_status_code` int DEFAULT '0' COMMENT '最后HTTP状态码',
  `last_response_time` bigint DEFAULT '0' COMMENT '最后响应时间(ms)',
  `status` tinyint(1) DEFAULT '1' COMMENT '状态:1-正常,2-异常,3-超时,4-检查失败',
  `check_time` varchar(64) DEFAULT '' COMMENT '最近检查时间',
  `error_msg` text COMMENT '错误信息',
  `create_time` datetime(3) NOT NULL COMMENT '创建时间',
  `update_time` datetime(3) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API端点监控表';

CREATE TABLE IF NOT EXISTS `monitor_data_source` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(128) NOT NULL COMMENT '数据源名称，如：北京核心机房-K8s集群',
  `type` varchar(64) NOT NULL COMMENT '数据源类型：Prometheus, Nightingale, Zabbix等',
  `deploy_method` varchar(64) NOT NULL COMMENT '部署方式：Kubernetes, Docker, Host, CloudManaged等',
  `api_url` varchar(255) NOT NULL COMMENT '数据源的直连/查询接口地址',
  `config` json NOT NULL COMMENT '动态凭证配置 (格式由 deploy_method 决定)',
  `status` tinyint(1) DEFAULT '1' COMMENT '状态：1-启用，0-停用',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_domain` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `domain` varchar(255) NOT NULL,
  `tags` varchar(500) DEFAULT NULL,
  `remark` text,
  `status` bigint DEFAULT '1' COMMENT '状态(1:启用,0:禁用)',
  `is_alive` bigint DEFAULT '0' COMMENT '存活状态(1:正常,0:异常)',
  `status_code` bigint DEFAULT NULL COMMENT 'HTTP状态码',
  `response_time` bigint DEFAULT NULL COMMENT '响应时间(ms)',
  `ssl_expire_at` datetime DEFAULT NULL COMMENT 'SSL证书过期时间',
  `ssl_days_left` bigint DEFAULT NULL COMMENT 'SSL证书剩余天数',
  `ssl_issuer` varchar(255) DEFAULT NULL COMMENT 'SSL证书颁发者',
  `last_check_at` datetime DEFAULT NULL COMMENT '最后检查时间',
  `error_msg` text COMMENT '错误信息',
  `create_time` datetime(3) DEFAULT NULL,
  `update_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_domain_domain` (`domain`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_domain_cert` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `domain` varchar(255) NOT NULL COMMENT '域名',
  `port` int DEFAULT '443' COMMENT '端口',
  `issuer` varchar(512) DEFAULT '' COMMENT '颁发者',
  `subject` varchar(512) DEFAULT '' COMMENT '主题',
  `not_before` varchar(64) DEFAULT '' COMMENT '起始日期',
  `not_after` varchar(64) DEFAULT '' COMMENT '到期日期',
  `remaining_days` int DEFAULT '-1' COMMENT '剩余天数(-1=未知)',
  `status` tinyint(1) DEFAULT '1' COMMENT '状态:1-正常,2-即将过期(<=30天),3-已过期,4-检查失败',
  `check_time` varchar(64) DEFAULT '' COMMENT '最近检查时间',
  `error_msg` text COMMENT '错误信息',
  `create_time` datetime(3) NOT NULL COMMENT '创建时间',
  `update_time` datetime(3) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_domain` (`domain`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='域名证书监控表';

CREATE TABLE IF NOT EXISTS `monitor_domain_schedule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `enabled` tinyint(1) DEFAULT '0' COMMENT '是否启用',
  `interval_mins` bigint DEFAULT NULL COMMENT '检查间隔(分钟)',
  `next_run_at` datetime DEFAULT NULL COMMENT '下次执行时间',
  `last_run_at` datetime DEFAULT NULL COMMENT '上次执行时间',
  `status` varchar(50) DEFAULT NULL COMMENT '任务状态(running/paused/stopped)',
  `create_time` datetime(3) DEFAULT NULL,
  `update_time` datetime(3) DEFAULT NULL,
  `cron_expr` varchar(100) DEFAULT NULL COMMENT 'Cron表达式',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_incident` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `alert_time` datetime(3) NOT NULL COMMENT '''告警时间''',
  `business_line` longtext COMMENT '''业务线(saas3/saas4/saas5等)''',
  `frequency` longtext COMMENT '''频率(偶发/频繁)''',
  `alert_desc` text COMMENT '''告警描述''',
  `alert_level` varchar(191) DEFAULT 'P4' COMMENT '''告警级别(P1/P2/P3/P4)''',
  `incident_cause` text COMMENT '''故障原因''',
  `department` longtext COMMENT '''所属部门(研发部/运维部)''',
  `solution` text COMMENT '''解决方案''',
  `detail_url` longtext COMMENT '''故障详情URL链接''',
  `handler` longtext COMMENT '''处理人(用户名)''',
  `handler_id` bigint unsigned DEFAULT NULL COMMENT '''处理人ID''',
  `status` bigint DEFAULT '1' COMMENT '''处理状态:1->未处理,2->处理中,3->已完成''',
  `remark` text COMMENT '''备注信息''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `business_line_id` bigint unsigned DEFAULT NULL COMMENT '''业务线ID(关联cmdb_group二级分组)''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_prometheus_alert` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `tpltype` longtext,
  `tpluse` longtext,
  `tplname` varchar(191) DEFAULT NULL,
  `tpl` text,
  `webhook_content_type` longtext,
  `created` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_monitor_prometheus_alert_tplname` (`tplname`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `monitor_ssl_cert` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `domain` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''域名''',
  `cert_source` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'aliyun_cas' COMMENT '''证书来源:aliyun_cas/acme''',
  `ca_provider` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'DigiCert' COMMENT '''CA提供商:DigiCert/LetsEncrypt/ZeroSSL''',
  `aliyun_config_id` bigint unsigned DEFAULT NULL COMMENT '''阿里云配置ID''',
  `order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '''证书订单ID''',
  `cert_id` bigint DEFAULT NULL COMMENT '''证书ID''',
  `cert_name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''证书名称''',
  `product_code` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'digicert-free-1-free' COMMENT '''产品代码''',
  `status` bigint DEFAULT NULL COMMENT '''状态:1->申请中,2->验证中,3->已签发,4->已过期,5->申请失败''',
  `validate_type` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'DNS' COMMENT '''验证方式:DNS/FILE''',
  `validate_info` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''验证信息(DNS记录或FILE内容)''',
  `cert` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''证书内容''',
  `private_key` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''私钥内容''',
  `issuer_cert` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''颁发者证书''',
  `algorithm` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'RSA2048' COMMENT '''加密算法:RSA2048/EC256''',
  `issue_time` datetime(3) DEFAULT NULL COMMENT '''签发时间''',
  `expire_time` datetime(3) DEFAULT NULL COMMENT '''过期时间''',
  `days_left` bigint DEFAULT NULL COMMENT '''剩余天数''',
  `error_msg` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''错误信息''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`),
  KEY `idx_monitor_ssl_cert_domain` (`domain`),
  KEY `idx_monitor_ssl_cert_cert_source` (`cert_source`),
  KEY `idx_monitor_ssl_cert_aliyun_config_id` (`aliyun_config_id`),
  KEY `idx_monitor_ssl_cert_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `quick_deployment_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `deployment_id` bigint unsigned NOT NULL,
  `app_id` bigint unsigned NOT NULL,
  `app_name` varchar(255) DEFAULT NULL,
  `app_code` varchar(255) DEFAULT NULL,
  `environment` varchar(50) DEFAULT NULL,
  `jenkins_env_id` bigint unsigned NOT NULL,
  `jenkins_job_url` varchar(500) DEFAULT NULL,
  `build_number` bigint DEFAULT NULL,
  `status` tinyint DEFAULT '1',
  `execute_order` bigint NOT NULL,
  `start_time` datetime(3) DEFAULT NULL,
  `end_time` datetime(3) DEFAULT NULL,
  `duration` bigint DEFAULT NULL,
  `error_message` text,
  `log_url` varchar(500) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `parameters` text,
  PRIMARY KEY (`id`),
  KEY `idx_quick_deployment_tasks_deployment_id` (`deployment_id`),
  KEY `fk_quick_deployment_tasks_application` (`app_id`),
  KEY `fk_quick_deployment_tasks_jenkins_env` (`jenkins_env_id`),
  CONSTRAINT `fk_quick_deployment_tasks_application` FOREIGN KEY (`app_id`) REFERENCES `app_application` (`id`),
  CONSTRAINT `fk_quick_deployment_tasks_jenkins_env` FOREIGN KEY (`jenkins_env_id`) REFERENCES `app_jenkins_env` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_quick_deployments_tasks` FOREIGN KEY (`deployment_id`) REFERENCES `quick_deployments` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `quick_deployments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL,
  `business_group_id` bigint unsigned NOT NULL,
  `business_dept_id` bigint unsigned NOT NULL,
  `description` text,
  `status` tinyint DEFAULT '1',
  `task_count` bigint NOT NULL DEFAULT '0',
  `creator_id` bigint unsigned NOT NULL,
  `creator_name` varchar(100) DEFAULT NULL,
  `start_time` datetime(3) DEFAULT NULL,
  `end_time` datetime(3) DEFAULT NULL,
  `duration` bigint DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `execution_mode` tinyint DEFAULT '1',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `redis_instance` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `code` varchar(36) NOT NULL COMMENT '''实例编码''',
  `name` varchar(100) NOT NULL COMMENT '''实例名称''',
  `mode` varchar(20) NOT NULL COMMENT '''模式:standalone,cluster,sentinel''',
  `host` varchar(300) NOT NULL COMMENT '''主机: standalone为host:port, cluster为逗号分隔, sentinel为master=hosts''',
  `port` bigint DEFAULT '0' COMMENT '''端口(standalone可用)''',
  `db` bigint DEFAULT '0' COMMENT '''默认库号''',
  `username` varchar(100) DEFAULT NULL COMMENT '''用户名(可选)''',
  `password` varchar(500) NOT NULL COMMENT '''密码(加密)''',
  `redis_node_password` varchar(500) DEFAULT '' COMMENT '''节点密码(仅sentinel)''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `ssh_tunnel_machine_id` bigint unsigned DEFAULT '0' COMMENT '''SSH隧道机器ID''',
  `status` bigint DEFAULT '1' COMMENT '''状态:1->启用,2->禁用''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  `creator` varchar(64) DEFAULT NULL COMMENT '''创建人''',
  `creator_id` bigint unsigned DEFAULT NULL COMMENT '''创建人ID''',
  `modifier` varchar(64) DEFAULT NULL COMMENT '''修改人''',
  `modifier_id` bigint unsigned DEFAULT NULL COMMENT '''修改人ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_redis_instance_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `sys_activity_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `activity_type` bigint NOT NULL COMMENT '''动态类型：1=密钥同步，2=域名检查，3=服务器巡检，4=定时任务，5=其他''',
  `title` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''动态标题''',
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''动态详细内容''',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '''状态：1=成功，2=失败，3=部分成功''',
  `related_id` bigint unsigned DEFAULT NULL COMMENT '''关联ID（如同步任务ID、域名ID等）''',
  `summary` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '''摘要信息''',
  `duration` bigint DEFAULT NULL COMMENT '''执行耗时（秒）''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_admin` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键',
  `post_id` int DEFAULT NULL COMMENT '岗位id',
  `dept_id` int DEFAULT NULL COMMENT '部门id',
  `username` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '账号',
  `password` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '密码',
  `nickname` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '昵称',
  `icon` varchar(500) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '头像',
  `email` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '手机',
  `note` varchar(500) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '备注信息',
  `create_time` datetime NOT NULL COMMENT '创建时间',
  `status` int NOT NULL DEFAULT '1' COMMENT '帐号启用状态：1->启用,2->禁用',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `username` (`username`) USING BTREE,
  KEY `create_time` (`create_time`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='后台管理员表';

CREATE TABLE IF NOT EXISTS `sys_admin_role` (
  `admin_id` int NOT NULL COMMENT '管理员id',
  `role_id` int NOT NULL COMMENT '角色id'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='管理员和角色关系表';

CREATE TABLE IF NOT EXISTS `sys_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `config_key` varchar(100) NOT NULL COMMENT '''配置键''',
  `config_type` varchar(50) NOT NULL COMMENT '''配置类型(ldap,email,sms等)''',
  `config_data` text NOT NULL COMMENT '''配置数据(JSON格式)''',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '''状态:1->启用,2->禁用''',
  `remark` varchar(500) DEFAULT NULL COMMENT '''备注''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) NOT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_config_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `sys_dept` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` int NOT NULL COMMENT '父id',
  `dept_type` int NOT NULL COMMENT '部门类型（1->公司, 2->中心，3->部门）',
  `dept_name` varchar(30) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin NOT NULL COMMENT '部门名称',
  `dept_status` int NOT NULL DEFAULT '1' COMMENT '部门状态（1->正常 2->停用）',
  `create_time` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `dept_name` (`dept_name`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin ROW_FORMAT=DYNAMIC COMMENT='部门表';

CREATE TABLE IF NOT EXISTS `sys_login_info` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `username` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT '' COMMENT '用户账号',
  `ip_address` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT '' COMMENT '登录IP地址',
  `login_location` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT '' COMMENT '登录地点',
  `browser` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT '' COMMENT '浏览器类型',
  `os` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT '' COMMENT '操作系统',
  `login_status` int DEFAULT NULL COMMENT '登录状态（1-成功 2-失败）',
  `message` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT '' COMMENT '提示消息',
  `login_time` datetime DEFAULT NULL COMMENT '访问时间',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin ROW_FORMAT=DYNAMIC COMMENT='登录日志记录';

CREATE TABLE IF NOT EXISTS `sys_menu` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` int DEFAULT NULL COMMENT '父级菜单id',
  `menu_name` varchar(100) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '菜单名称',
  `icon` varchar(100) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '图标',
  `value` varchar(100) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '接口权限值',
  `menu_type` int DEFAULT NULL COMMENT '菜单类型：1->目录；2->菜单；3->按钮（接口绑定权限）',
  `url` varchar(200) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '菜单url',
  `menu_status` int DEFAULT '2' COMMENT '启用状态；1->禁用；2->启用',
  `sort` int DEFAULT NULL COMMENT '排序',
  `create_time` datetime DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='菜单表';

CREATE TABLE IF NOT EXISTS `sys_operation_log` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键',
  `admin_id` bigint unsigned NOT NULL COMMENT '''管理员id''',
  `username` varchar(64) NOT NULL COMMENT '''管理员账号''',
  `method` varchar(64) NOT NULL COMMENT '''请求方式''',
  `ip` varchar(64) DEFAULT NULL COMMENT '''IP''',
  `url` varchar(500) DEFAULT NULL COMMENT '''URL''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `description` varchar(255) DEFAULT NULL COMMENT '''操作描述''',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `username` (`username`) USING BTREE,
  KEY `create_time` (`create_time`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='操作日志记录表';

CREATE TABLE IF NOT EXISTS `sys_post` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '岗位ID',
  `post_code` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin NOT NULL COMMENT '岗位编码',
  `post_name` varchar(50) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin NOT NULL COMMENT '岗位名称',
  `post_status` int NOT NULL DEFAULT '1' COMMENT '状态（1->正常 2->停用）',
  `create_time` datetime DEFAULT NULL COMMENT '创建时间',
  `remark` varchar(500) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin ROW_FORMAT=DYNAMIC COMMENT='岗位信息表';

CREATE TABLE IF NOT EXISTS `sys_role` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键',
  `role_name` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '角色名称',
  `role_key` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '角色权限字符串',
  `status` int NOT NULL DEFAULT '1' COMMENT '启用状态：1->启用；2->禁用',
  `description` varchar(500) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '描述',
  `create_time` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `name` (`role_name`) USING BTREE,
  KEY `create_time` (`create_time`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='后台角色表';

CREATE TABLE IF NOT EXISTS `sys_role_menu` (
  `role_id` int DEFAULT NULL COMMENT '角色ID',
  `menu_id` int DEFAULT NULL COMMENT '菜单ID'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='角色和菜单关系表';

CREATE TABLE IF NOT EXISTS `task_ansible` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) NOT NULL COMMENT '''任务名称''',
  `description` text COMMENT '''任务描述''',
  `type` bigint NOT NULL DEFAULT '1' COMMENT '''任务类型:1-手动,2-Git,3-K8s''',
  `git_repo` varchar(255) DEFAULT NULL COMMENT '''Git仓库地址''',
  `host_groups` text NOT NULL COMMENT '''主机分组JSON''',
  `all_host_ids` text NOT NULL COMMENT '''所有主机ID JSON数组''',
  `global_vars` text COMMENT '''全局变量JSON''',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '''任务状态:1-等待中,2-运行中,3-成功,4-异常''',
  `created_at` datetime(3) NOT NULL COMMENT '''创建时间''',
  `updated_at` datetime(3) NOT NULL COMMENT '''更新时间''',
  `error_msg` text COMMENT '''错误信息''',
  `task_count` bigint NOT NULL DEFAULT '0' COMMENT '''任务数量(Type=1时为上传文件数,Type=2时为解析的playbook数,Type=3时固定为1)''',
  `total_duration` bigint NOT NULL DEFAULT '0' COMMENT '''任务执行总耗时(秒,所有子任务耗时总和)''',
  `extra_vars` text COMMENT '''额外参数YAML/JSON''',
  `cli_args` text COMMENT '''cli命令行参数''',
  `use_config` bigint NOT NULL DEFAULT '0' COMMENT '''是否使用配置管理中的参数 0-不使用,1-使用''',
  `inventory_config_id` bigint unsigned DEFAULT NULL COMMENT '''选用的inventory配置ID''',
  `global_vars_config_id` bigint unsigned DEFAULT NULL COMMENT '''选用的global_vars配置ID''',
  `extra_vars_config_id` bigint unsigned DEFAULT NULL COMMENT '''选用的extra_vars配置ID''',
  `cli_args_config_id` bigint unsigned DEFAULT NULL COMMENT '''选用的cli_args配置ID''',
  `max_history_keep` bigint DEFAULT '3' COMMENT '''最大保留历史记录数''',
  `cron_expr` varchar(64) DEFAULT NULL COMMENT '''定时表达式''',
  `is_recurring` bigint NOT NULL DEFAULT '0' COMMENT '''是否周期性任务:0-否,1-是''',
  `view_id` bigint unsigned DEFAULT NULL COMMENT '''视图ID''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_task_ansible_name` (`name`),
  KEY `idx_task_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `task_ansible_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `task_id` bigint unsigned NOT NULL COMMENT '''关联的任务ID''',
  `uniq_id` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''任务唯一标识(每次执行生成)''',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '''执行状态:1-等待中,2-运行中,3-成功,4-异常''',
  `error_msg` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '''错误信息''',
  `total_duration` bigint NOT NULL DEFAULT '0' COMMENT '''任务执行总耗时(秒)''',
  `trigger` bigint NOT NULL DEFAULT '1' COMMENT '''触发方式:1-手动,2-定时,3-API''',
  `operator_id` bigint unsigned DEFAULT NULL COMMENT '''操作人ID''',
  `operator_name` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '''操作人姓名''',
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_history_task_id` (`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `task_ansible_view` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''视图名称''',
  `created_at` datetime(3) DEFAULT NULL COMMENT '''创建时间''',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_task_ansible_view_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `task_ansiblework` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `task_id` bigint unsigned NOT NULL COMMENT '''父任务ID''',
  `entry_file_name` varchar(255) NOT NULL COMMENT '''入口文件名''',
  `entry_file_path` varchar(255) NOT NULL COMMENT '''入口文件路径''',
  `log_path` varchar(255) DEFAULT NULL COMMENT '''日志路径''',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '''子任务状态:1-等待中,2-运行中,3-成功,4-异常''',
  `start_time` datetime(3) DEFAULT NULL COMMENT '''开始时间''',
  `end_time` datetime(3) DEFAULT NULL COMMENT '''结束时间''',
  `duration` bigint DEFAULT NULL COMMENT '''执行耗时(秒)''',
  `exit_code` bigint DEFAULT NULL COMMENT '''退出代码''',
  `error_msg` text COMMENT '''错误信息''',
  `log` text COMMENT '''日志内容''',
  PRIMARY KEY (`id`),
  KEY `idx_task_ansiblework_task_id` (`task_id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_task_work_composite` (`task_id`,`status`),
  CONSTRAINT `fk_task_ansible_works` FOREIGN KEY (`task_id`) REFERENCES `task_ansible` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `task_ansiblework_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `history_id` bigint unsigned NOT NULL COMMENT '''关联的历史记录ID''',
  `task_id` bigint unsigned NOT NULL COMMENT '''关联的任务ID''',
  `work_id` bigint unsigned DEFAULT NULL COMMENT '''关联的WorkID(如果有)''',
  `host_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''主机名/IP''',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '''状态:1-等待,2-执行中,3-成功,4-失败,5-跳过''',
  `log_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '''日志文件路径''',
  `duration` bigint NOT NULL DEFAULT '0' COMMENT '''耗时(秒)''',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_work_history_id` (`history_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `task_job` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL COMMENT '任务标题',
  `type` bigint DEFAULT NULL COMMENT '任务类型 1=普通任务,2=定时任务,3=ansible任务',
  `shell` text COMMENT '任务内容(任务模板ID,多个用逗号分隔)',
  `host_ids` text COMMENT '主机ID(多个用逗号分隔)',
  `cron_expr` varchar(255) DEFAULT NULL COMMENT '定时表达式(* * * * *)',
  `tasklog` text COMMENT '任务执行日志',
  `status` bigint DEFAULT NULL COMMENT '任务状态 1=等待中,2=运行中,3=成功,4=异常,5=已暂停',
  `duration` bigint DEFAULT NULL COMMENT '执行耗时(秒)',
  `remark` text COMMENT '任务备注',
  `start_time` datetime(3) DEFAULT NULL COMMENT '任务开始时间',
  `end_time` datetime(3) DEFAULT NULL COMMENT '任务结束时间',
  `created_at` datetime(3) DEFAULT NULL COMMENT '任务创建时间',
  `task_count` bigint DEFAULT NULL COMMENT '任务数量',
  `is_recurring` tinyint(1) DEFAULT NULL COMMENT '是否周期性任务',
  `scheduled_time` datetime(3) DEFAULT NULL COMMENT '计划执行时间',
  `log_path` varchar(500) DEFAULT NULL COMMENT '日志文件路径',
  `execute_count` bigint DEFAULT '0' COMMENT '执行次数',
  `next_run_time` datetime(3) DEFAULT NULL COMMENT '下次执行时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `task_template` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `type` bigint NOT NULL,
  `content` text NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `created_by` varchar(50) DEFAULT NULL,
  `updated_by` varchar(50) DEFAULT NULL,
  `remark` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `task_work` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `task_id` bigint unsigned DEFAULT NULL COMMENT '关联的任务ID',
  `template_id` bigint unsigned DEFAULT NULL COMMENT '任务模板ID',
  `host_id` bigint unsigned DEFAULT NULL COMMENT '执行主机ID',
  `status` bigint DEFAULT NULL COMMENT '任务状态 1=等待中,2=运行中,3=成功,4=异常',
  `log` text COMMENT '任务日志',
  `log_path` text COMMENT '日志文件路径',
  `start_time` datetime(3) DEFAULT NULL COMMENT '任务开始时间',
  `end_time` datetime(3) DEFAULT NULL COMMENT '任务结束时间',
  `duration` bigint DEFAULT NULL COMMENT '执行耗时(秒)',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `type` bigint DEFAULT NULL COMMENT '任务类型 1=普通任务,2=定时任务',
  `scheduled_time` datetime(3) DEFAULT NULL COMMENT '定时任务执行时间',
  `cron_expr` longtext COMMENT 'cron表达式',
  `is_recurring` tinyint(1) DEFAULT NULL COMMENT '是否周期性任务',
  PRIMARY KEY (`id`),
  KEY `idx_task_work_task_id` (`task_id`),
  KEY `idx_task_work_template_id` (`template_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `tool_link` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `title` longtext NOT NULL COMMENT '''导航标题''',
  `icon` longtext COMMENT '''导航图标''',
  `link` longtext NOT NULL COMMENT '''链接地址''',
  `sort` bigint DEFAULT '0' COMMENT '''排序''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `tool_service_deploy` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `service_name` longtext NOT NULL COMMENT '''服务名称''',
  `service_id` longtext NOT NULL COMMENT '''服务ID''',
  `version` longtext NOT NULL COMMENT '''服务版本''',
  `host_id` bigint unsigned NOT NULL COMMENT '''主机ID''',
  `host_ip` longtext NOT NULL COMMENT '''主机IP''',
  `install_dir` longtext NOT NULL COMMENT '''安装目录''',
  `container_name` longtext COMMENT '''容器名称''',
  `ports` longtext COMMENT '''端口映射(JSON)''',
  `env_vars` longtext COMMENT '''环境变量(JSON)''',
  `status` bigint DEFAULT '0' COMMENT '''状态:0->部署中,1->运行中,2->已停止,3->部署失败''',
  `deploy_log` longtext COMMENT '''部署日志''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO `sys_menu` VALUES (4,0,'系统管理','StarFilled','',1,'',2,7,'2022-09-04 13:57:39'),(6,4,'用户信息','Avatar','base:admin:list',2,'system/admin',2,1,'2022-09-04 13:59:39'),(7,4,'角色信息','InfoFilled','base:role:list',2,'system/role',2,2,'2022-09-04 14:00:12'),(8,4,'菜单信息','Histogram','base:menu:list',2,'system/menu',2,3,'2022-09-04 14:00:17'),(9,4,'部门信息','Menu','base:dept:list',2,'system/dept',2,4,'2022-09-04 14:01:58'),(10,4,'岗位信息','Promotion','base:post:list',2,'system/post',2,5,'2022-09-04 14:02:06'),(16,6,'新增用户','','base:admin:add',3,'',2,1,'2022-09-04 18:32:55'),(17,6,'修改用户','','base:admin:edit',3,'',2,2,'2022-09-04 18:33:29'),(18,6,'删除用户','','base:admin:delete',3,'',2,3,'2022-09-04 18:33:51'),(21,7,'新增角色','','base:role:add',3,'',2,1,'2022-09-04 18:44:47'),(22,7,'修改角色','','base:role:edit',3,'',2,2,'2022-09-04 18:45:10'),(23,7,'删除角色','','base:role:delete',3,'',2,3,'2022-09-04 18:45:46'),(24,7,'分配权限','','base:role:assign',3,'',2,4,'2022-09-04 18:46:20'),(26,8,'新增菜单','','base:menu:add',3,'',2,1,'2022-09-04 18:49:51'),(27,8,'修改菜单','','base:menu:edit',3,'',2,2,'2022-09-04 18:50:24'),(28,8,'删除菜单','','base:menu:delete',3,'',2,3,'2022-09-04 18:50:53'),(29,9,'新增部门','','base:dept:add',3,'',2,1,'2022-09-04 18:52:16'),(30,9,'修改部门','','base:dept:edit',3,'',2,2,'2022-09-04 18:52:37'),(31,9,'删除部门','','base:dept:delete',3,'',2,3,'2022-09-04 18:52:50'),(32,10,'新增岗位','','base:post:add',3,'',2,1,'2022-09-04 18:53:28'),(33,10,'修改岗位','','base:post:edit',3,'',2,2,'2022-09-04 18:53:48'),(34,10,'删除岗位','','base:post:delete',3,'',2,3,'2022-09-04 18:54:00'),(44,0,'操作审计','BellFilled','',1,'',2,9,'2022-09-05 11:06:57'),(45,44,'操作日志','User','monitor:operator:list',2,'monitor/operator',2,1,'2022-09-05 11:10:54'),(46,44,'登录日志','DocumentRemove','monitor:loginLog:list',2,'monitor/loginlog',2,2,'2022-09-05 11:11:31'),(47,45,'清空操作日志','','monitor:operator:clean',3,'',2,1,'2022-09-05 11:12:36'),(49,46,'清空登录日志','','monitor:loginLog:clean',3,'',2,1,'2022-09-05 11:16:01'),(60,6,'重置密码',NULL,'base:admin:reset',3,NULL,2,6,'2022-12-01 16:33:34'),(62,46,'删除登录日志','','monitor:loginLog:delete',3,'',2,2,'2022-12-02 15:41:56'),(72,0,'仪表盘','HomeFilled','',1,'dashboard',2,1,'2023-05-24 22:11:13'),(73,45,'删除操作日志','','monitor:operator:delete',3,'',2,3,'2023-06-02 10:09:38'),(78,80,'主机管理','Platform','cmdb:ecs:list',2,'cmdb/ecs',2,1,'2025-06-29 00:30:35'),(80,0,'资产管理','TrendCharts','',1,'',2,2,'2025-07-03 11:47:07'),(81,0,'容器管理','UploadFilled','',1,'',2,3,'2025-07-03 11:50:47'),(82,81,'集群管理','Menu','cloud:k8s:list',2,'k8s/list',2,1,'2025-07-03 11:56:44'),(83,81,'节点管理','Help','cloud:k8s:node',2,'k8s/node',2,2,'2025-07-03 12:04:59'),(84,0,'配置中心','Tools','',1,'',2,8,'2025-07-04 17:00:01'),(85,84,'主机凭据','Setting','config:ecs:key',2,'config/ecskey',2,1,'2025-07-04 17:03:10'),(86,84,'通用凭据','User','config:accountauth:key',2,'config/accountauth',2,2,'2025-07-04 17:08:20'),(88,80,'业务分组','Shop','cmdb:group',2,'cmdb/group',2,2,'2025-07-16 15:17:14'),(89,88,'创建分组','','cmdb:group:add',3,'',2,1,'2025-07-18 15:24:31'),(90,88,'修改分组','','cmdb:group:update',3,'',2,2,'2025-07-18 15:25:49'),(91,88,'删除分组','','cmdb:group:delete',3,'',2,3,'2025-07-18 15:26:21'),(93,81,'工作负载','Star','cloud:k8s:workload',2,'k8s/workload',2,4,'2025-07-24 14:38:31'),(95,80,'数据管理','Coin','cmdb:db',2,'cmdb/db',2,3,'2025-07-29 15:27:50'),(96,44,'数据日志','Coin','monitor:dblog:list',2,'monitor/dblog',2,3,'2025-07-31 12:44:07'),(97,0,'任务中心','User','',1,'',2,5,'2025-08-06 13:33:47'),(98,97,'任务模版','connection','task:template',2,'task/template',2,2,'2025-08-06 13:35:19'),(99,97,'任务作业','key','task:job',2,'task/job',2,1,'2025-08-06 13:36:06'),(100,97,'Ansible任务','Eleme','task:ansible',2,'task/ansible',2,3,'2025-08-23 18:35:24'),(101,0,'运维工具','Search','',1,'',2,6,'2025-08-29 10:59:35'),(102,101,'agent列表','price-tag','ops:agent',2,'ops/agent',2,2,'2025-08-29 11:22:20'),(103,101,'工具列表','present','ops:tools',2,'ops/tools',2,1,'2025-08-29 11:29:02'),(104,84,'密钥管理','Phone','config:keymanage:key',2,'config/keymanage',2,3,'2025-09-08 13:24:40'),(105,81,'命名空间','discount','k8s:namespace',2,'k8s/namespace',2,3,'2025-09-11 15:02:14'),(106,81,'网络管理','guide','k8s:network',2,'k8s/network',2,5,'2025-09-11 15:04:14'),(107,81,'配置管理','connection','k8s:config',2,'k8s/config',2,7,'2025-09-11 15:04:52'),(108,81,'存储管理','Coin','k8s:storage',2,'k8s/storage',2,6,'2025-09-11 15:05:40'),(109,0,'服务管理','ElemeFilled','',1,'',2,4,'2025-09-16 09:49:55'),(110,109,'应用列表','Menu','app:application',2,'app/application',2,1,'2025-09-16 09:52:58'),(111,109,'快速发布','TrendCharts','app:quick-release',2,'app/quick-release',2,2,'2025-09-16 17:12:11'),(113,45,'批量删除','','monitor:operator:delete',3,'',2,2,'2025-09-17 20:55:13'),(114,104,'修改密钥','','config:keymanage:edit',3,'',2,1,'2025-09-18 10:45:57'),(115,104,'删除密钥','','config:keymanage:delete',3,'',2,2,'2025-09-18 10:53:44'),(117,104,'同步主机','','config:keymanage:rsync',3,'',2,3,'2025-09-18 10:57:25'),(118,104,'创建密钥','','config:keymanage:create',3,'',2,4,'2025-09-18 11:01:12'),(119,86,'修改账号','','config:common:edit',3,'',2,2,'2025-09-18 11:47:33'),(120,86,'删除账号','','config:common:delete',3,'',2,3,'2025-09-18 11:48:17'),(121,86,'解密账号','','config:common:decrypt',3,'',2,4,'2025-09-18 11:48:57'),(122,86,'创建账号','','config:common:add',3,'',2,1,'2025-09-18 11:49:30'),(123,85,'修改凭据','','config:ecs:edit',3,'',2,1,'2025-09-18 11:54:16'),(124,85,'删除凭据','','config:ecs:delete',3,'',2,2,'2025-09-18 11:54:51'),(125,85,'创建凭据','','config:ecs:create',3,'',2,3,'2025-09-18 11:55:21'),(126,102,'卸载agent','','ops:agent:delete',3,'',2,1,'2025-09-18 12:47:48'),(127,102,'查看agent','','ops:agent:get',3,'',2,2,'2025-09-18 12:49:02'),(128,102,'部署agent','','ops:agent:create',3,'',2,3,'2025-09-18 12:49:56'),(129,102,'批量卸载agent','','ops:agent:deleteall',3,'',2,4,'2025-09-18 12:50:52'),(130,100,'启动ansible任务流程','','task:ansible:start',3,'',2,1,'2025-09-18 12:59:30'),(131,100,'删除ansible任务','','task:ansible:delete',3,'',2,2,'2025-09-18 13:00:03'),(132,100,'新增ansible任务','','task:ansible:create',3,'',2,3,'2025-09-18 13:00:45'),(133,98,'新增模版','','task:template:add',3,'',2,1,'2025-09-18 13:16:38'),(134,98,'修改模版','','task:template:edit',3,'',2,2,'2025-09-18 13:17:04'),(135,98,'删除模版','','task:template:delete',3,'',2,3,'2025-09-18 13:18:25'),(136,99,'新增任务','','task:job:add',3,'',2,1,'2025-09-18 13:24:19'),(137,99,'启动任务','','task:job:start',3,'',2,2,'2025-09-18 13:24:59'),(138,99,'删除任务','','task:job:delete',3,'',2,3,'2025-09-18 13:25:41'),(139,111,'新建发布','','app:quick-release:add',3,'',2,1,'2025-09-18 13:30:53'),(140,111,'启动发布','','app:quick-release:start',3,'',2,2,'2025-09-18 13:32:11'),(141,111,'删除发布','','app:quick-release:delete',3,'',2,3,'2025-09-18 13:32:32'),(142,110,'创建应用','','app:application:add',3,'',2,1,'2025-09-18 14:28:07'),(143,110,'修改应用','','app:application:edit',3,'',2,2,'2025-09-18 14:28:59'),(144,110,'配置应用环境','','app:application:env',3,'',2,3,'2025-09-18 14:29:34'),(145,110,'删除应用','','app:application:delete',3,'',2,4,'2025-09-18 14:30:11'),(146,95,'创建数据库账号','','cmdb:db:add',3,'',2,1,'2025-09-18 14:41:32'),(147,95,'修改数据库配置','','cmdb:db:edit',3,'',2,2,'2025-09-18 14:42:47'),(148,95,'删除数据库账号','','cmdb:db:delete',3,'',2,3,'2025-09-18 14:43:57'),(149,78,'创建主机','','cmdb:ecs:add',3,'',2,1,'2025-09-18 14:47:42'),(150,78,'主机终端','','cmdb:ecs:terminal',3,'',2,2,'2025-09-18 14:48:36'),(151,78,'修改主机信息','','cmdb:ecs:edit',3,'',2,3,'2025-09-18 14:49:43'),(152,78,'上传文件到主机','','cmdb:ecs:upload',3,'',2,4,'2025-09-18 14:50:38'),(153,78,'执行主机命令','','cmdb:ecs:shell',3,'',2,5,'2025-09-18 14:51:10'),(154,78,'监控主机','','cmdb:ecs:monitor',3,'',2,6,'2025-09-18 14:51:52'),(155,78,'删除主机','','cmdb:ecs:delete',3,'',2,7,'2025-09-18 14:52:20'),(156,99,'启动脚本','','task:job:jobstart',3,'',2,4,'2025-09-18 18:36:38'),(157,99,'停止脚本','','task:job:jobstop',3,'',2,5,'2025-09-18 18:39:23'),(159,100,'开始ansible任务作业','','task:ansible:jobstart',3,'',2,4,'2025-09-18 18:43:40'),(160,111,'启动jenkins任务','','app:quick-release:jobstart',3,'',2,4,'2025-09-18 18:47:39'),(161,111,'停止jenkins任务','','app:quick-release:jobstop',3,'',2,5,'2025-09-18 18:48:16'),(162,110,'新增环境','','app:application:envadd',3,'',2,5,'2025-09-18 21:02:28'),(163,110,'修改环境','','app:application:envedit',3,'',2,6,'2025-09-18 21:03:08'),(164,110,'删除环境','','app:application:envdelete',3,'',2,7,'2025-09-18 21:04:43'),(165,78,'连接主机终端','','cmdb:ecs:connecthost',3,'',2,8,'2025-09-18 21:11:43'),(166,78,'同步主机信息','','cmdb:ecs:rsync',3,'',2,9,'2025-09-19 21:35:06'),(167,82,'注册集群','','cloud:k8s:register',3,'',2,1,'2025-09-19 21:57:54'),(168,82,'创建集群','','cloud:k8s:add',3,'',2,2,'2025-09-19 21:58:19'),(169,82,'修改集群','','cloud:k8s:edit',3,'',2,3,'2025-09-19 21:59:06'),(170,82,'同步集群','','cloud:k8s:rsync',3,'',2,4,'2025-09-19 21:59:31'),(171,82,'删除集群','','cloud:k8s:delete',3,'',2,5,'2025-09-19 21:59:56'),(172,83,'查看监控仪表盘','','k8s:node:monitor',3,'',2,1,'2025-09-20 00:19:49'),(173,83,'查看节点资源详情','','k8s:node:details',3,'',2,2,'2025-09-20 00:21:20'),(174,83,'配置节点污点','','k8s:node:stain',3,'',2,3,'2025-09-20 00:22:17'),(175,83,'增加标签','','k8s:node:label',3,'',2,4,'2025-09-20 00:23:15'),(176,83,'封锁节点','','k8s:node:close',3,'',2,5,'2025-09-20 00:24:13'),(177,83,'驱逐节点','','k8s:node:expel',3,'',2,6,'2025-09-20 00:25:04'),(178,105,'创建命名空间','','k8s:namespace:add',3,'',2,1,'2025-09-20 00:36:14'),(179,105,'查看命名空间详情','','k8s:namespace:details',3,'',2,2,'2025-09-20 00:37:22'),(180,105,'查看命名空间资源配置','','k8s:namespace:setup',3,'',2,3,'2025-09-20 00:39:05'),(181,105,'添加命名空间资源配置','','k8s:namespace:setupadd',3,'',2,4,'2025-09-20 00:40:06'),(182,105,'查看限制命名空间资源','','k8s:namespace:restriction',3,'',2,5,'2025-09-20 00:41:25'),(183,105,'添加限制命名空间资源','','k8s:namespace:restrictionadd',3,'',2,6,'2025-09-20 00:42:23'),(184,105,'删除命名空间','','k8s:namespace:delete',3,'',2,7,'2025-09-20 00:43:03'),(185,93,'新增工作负载','','k8s:workload:add',3,'',2,1,'2025-09-20 01:05:08'),(186,93,'伸缩pod','','k8s:workload:expandable',3,'',2,2,'2025-09-20 01:06:18'),(187,93,'重启pod','','k8s:workload:restart',3,'',2,3,'2025-09-20 01:07:13'),(188,93,'更新pod资源限制','','k8s:workload:resource',3,'',2,4,'2025-09-20 01:08:39'),(189,93,'更新pod资调度','','k8s:workload:dispatch',3,'',2,5,'2025-09-20 01:09:36'),(190,93,'更新yaml文件','','k8s:workload:edityaml',3,'',2,6,'2025-09-20 01:10:57'),(191,93,'删除工作负载','','k8s:workload:delete',3,'',2,7,'2025-09-20 01:11:52'),(192,93,'回滚工作负载版本','','k8s:workload:rollback_version',3,'',2,8,'2025-09-20 01:39:38'),(193,93,'查看pod日志','','k8s:workload:podlog',3,'',2,9,'2025-09-20 01:44:37'),(194,93,'删除pod','','k8s:workload:poddelete',3,'',2,10,'2025-09-20 01:45:16'),(195,93,'登陆pod终端','','k8s:workload:terminal',3,'',2,11,'2025-09-20 01:46:07'),(196,93,'编辑pod yaml文件','','k8s:workload:edityaml',3,'',2,12,'2025-09-20 01:47:22'),(197,106,'新增service','','k8s:network:addservice',3,'',2,1,'2025-09-20 02:14:21'),(198,106,'编辑 Service','','k8s:network:editservice',3,'',2,2,'2025-09-20 02:15:19'),(199,106,'编辑service YAML','','k8s:network:edit_service_yaml',3,'',2,3,'2025-09-20 02:16:15'),(200,106,'查看Service 事件','','k8s:network:service_event',3,'',2,4,'2025-09-20 02:18:11'),(201,106,'删除Service','','k8s:network:deleteservice',3,'',2,5,'2025-09-20 02:18:59'),(202,106,'编辑ingress','','k8s:network:editingress',3,'',2,6,'2025-09-20 02:26:59'),(203,106,'新增ingress','','k8s:network:addingress',3,'',2,7,'2025-09-20 02:27:29'),(204,106,'编辑ingress_yaml','','k8s:network:edit_ingress_yaml',3,'',2,8,'2025-09-20 02:28:23'),(205,106,'查看ingress 事件','','k8s:network:ingress_event',3,'',2,9,'2025-09-20 02:29:24'),(206,106,'删除ingress','','k8s:network:delete_ingress',3,'',2,10,'2025-09-20 02:30:04'),(212,0,'监控中心','Shop','',1,'',2,4,'2025-12-03 21:21:04'),(213,212,'域名监控','UploadFilled','monitor:domain',2,'monitor/domain',2,1,'2025-12-03 21:22:11'),(215,4,'系统配置','List','system:config',2,'system/config',2,6,'2025-12-09 11:03:54'),(216,212,'故障管理','Help','monitor:incident',2,'monitor/incident',2,2,'2025-12-10 15:10:29'),(217,95,'查看密码','','cmdb:db:passwd',3,'',2,4,'2025-12-13 14:27:22'),(218,95,'测试链接','','cmdb:db:test',3,'',2,5,'2025-12-13 14:28:38'),(219,95,'DBMS终端','','cmdb:db:dbms',3,'',2,6,'2025-12-13 14:32:59'),(220,95,'Redis终端','','cmdb:db:redis',3,'',2,7,'2025-12-13 14:33:27'),(221,95,'ES终端','','cmdb:db:es',3,'',2,8,'2025-12-13 14:33:53'),(222,95,'Mongo终端','','cmdb:db:mongo',3,'',2,9,'2025-12-13 14:34:14'),(229,213,'新增域名','','monitor:domain:add',3,'',2,1,'2025-12-13 15:22:47'),(230,213,'批量新增域名','','monitor:domain:add_all',3,'',2,2,'2025-12-13 15:23:44'),(231,213,'域名自动巡检','','monitor:domain:auto_inspection',3,'',2,3,'2025-12-13 15:25:35'),(232,213,'域名手动巡检','','monitor:domain:manual_inspection',3,'',2,4,'2025-12-13 15:26:55'),(233,213,'单域名手动巡检','','monitor:domain:ops_inspection',3,'',2,5,'2025-12-13 15:36:47'),(234,213,'编辑域名','','monitor:domain:edit',3,'',2,6,'2025-12-13 15:38:30'),(235,213,'删除域名','','monitor:domain:delete',3,'',2,7,'2025-12-13 15:38:54'),(236,216,'新增故障记录','','monitor:incident:add',3,'',2,1,'2025-12-13 15:55:41'),(237,216,'编辑故障记录','','monitor:incident:edit',3,'',2,2,'2025-12-13 15:57:20'),(238,216,'编辑故障状态','','monitor:incident:status',3,'',2,3,'2025-12-13 15:57:39'),(239,216,'删除故障记录','','monitor:incident:delete',3,'',2,4,'2025-12-13 15:58:12'),(240,216,'访问故障链接','','monitor:incident:url',3,'',2,5,'2025-12-13 16:02:46'),(241,78,'下载文件','','cmdb:ecs:download',3,'',2,10,'2025-12-13 16:13:17'),(243,80,'物理机管理','Platform','cmdb:physical:list',2,'cmdb/physical',2,4,'2026-05-30 12:46:25'),(244,243,'创建物理机','','cmdb:physical:add',3,'',2,1,'2026-05-30 13:01:16');

INSERT INTO `sys_role` VALUES (1,'超级管理员','admin',1,'最大权限','2023-06-12 20:04:53'),(13,'游客','test',1,'test1','2025-07-03 18:47:25');

INSERT INTO `sys_role_menu` VALUES (1,72),(1,80),(1,78),(1,149),(1,150),(1,151),(1,152),(1,153),(1,154),(1,155),(1,165),(1,166),(1,241),(1,88),(1,89),(1,90),(1,91),(1,95),(1,146),(1,147),(1,148),(1,217),(1,218),(1,219),(1,220),(1,221),(1,222),(1,81),(1,82),(1,167),(1,168),(1,169),(1,170),(1,171),(1,83),(1,172),(1,173),(1,174),(1,175),(1,176),(1,177),(1,105),(1,178),(1,179),(1,180),(1,181),(1,182),(1,183),(1,184),(1,93),(1,185),(1,186),(1,187),(1,188),(1,189),(1,190),(1,191),(1,192),(1,193),(1,194),(1,195),(1,196),(1,106),(1,197),(1,198),(1,199),(1,200),(1,201),(1,202),(1,203),(1,204),(1,205),(1,206),(1,108),(1,107),(1,110),(1,142),(1,143),(1,144),(1,145),(1,162),(1,163),(1,164),(1,111),(1,139),(1,140),(1,141),(1,160),(1,161),(1,212),(1,213),(1,229),(1,230),(1,231),(1,232),(1,233),(1,234),(1,235),(1,216),(1,236),(1,237),(1,238),(1,239),(1,240),(1,97),(1,99),(1,136),(1,137),(1,138),(1,156),(1,157),(1,98),(1,133),(1,134),(1,135),(1,100),(1,130),(1,131),(1,132),(1,159),(1,101),(1,103),(1,102),(1,126),(1,127),(1,128),(1,129),(1,4),(1,6),(1,16),(1,17),(1,18),(1,60),(1,7),(1,21),(1,22),(1,23),(1,24),(1,8),(1,26),(1,27),(1,28),(1,9),(1,29),(1,30),(1,31),(1,10),(1,32),(1,33),(1,34),(1,215),(1,84),(1,85),(1,123),(1,124),(1,125),(1,86),(1,122),(1,119),(1,120),(1,121),(1,104),(1,114),(1,115),(1,117),(1,118),(1,44),(1,45),(1,47),(1,113),(1,73),(1,46),(1,49),(1,62),(1,96),(1,109),(13,72),(13,149),(13,150),(13,154),(13,165),(13,89),(13,146),(13,218),(13,219),(13,167),(13,168),(13,170),(13,172),(13,173),(13,174),(13,175),(13,178),(13,179),(13,180),(13,181),(13,182),(13,185),(13,186),(13,190),(13,193),(13,195),(13,197),(13,200),(13,203),(13,205),(13,108),(13,107),(13,142),(13,144),(13,162),(13,139),(13,140),(13,160),(13,229),(13,233),(13,236),(13,240),(13,136),(13,137),(13,156),(13,133),(13,130),(13,132),(13,159),(13,103),(13,127),(13,16),(13,125),(13,122),(13,118),(13,73),(13,62),(13,96),(13,80),(13,78),(13,88),(13,95),(13,81),(13,82),(13,83),(13,105),(13,93),(13,106),(13,109),(13,110),(13,111),(13,212),(13,213),(13,216),(13,97),(13,99),(13,98),(13,100),(13,101),(13,102),(13,4),(13,6),(13,84),(13,85),(13,86),(13,104),(13,44),(13,45),(13,46);

SET FOREIGN_KEY_CHECKS=1;

INSERT INTO sys_admin (id, post_id, dept_id, username, password, nickname, status, icon, email, phone, note, create_time) VALUES (89,0,0,'admin','e10adc3949ba59abbe56e057f20f883e','Administrator',1,'','','','Initial administrator',NOW());

INSERT INTO sys_admin_role (admin_id,role_id) VALUES (89,1);
