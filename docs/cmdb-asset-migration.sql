-- ======================= CMDB 资产扩展表 =======================

-- 机房表
CREATE TABLE IF NOT EXISTS `cmdb_idc` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` varchar(100) NOT NULL COMMENT '机房名称',
    `short_name` varchar(50) DEFAULT '' COMMENT '机房简称',
    `address` varchar(255) DEFAULT '' COMMENT '机房地址',
    `contact` varchar(50) DEFAULT '' COMMENT '联系人',
    `phone` varchar(30) DEFAULT '' COMMENT '联系电话',
    `level` varchar(20) DEFAULT '' COMMENT '机房等级(T1-T4)',
    `status` int(11) DEFAULT 1 COMMENT '状态:1-启用,2-停用',
    `description` text COMMENT '描述',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cmdb_idc_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CMDB机房表';

-- 机柜表
CREATE TABLE IF NOT EXISTS `cmdb_cabinet` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` varchar(100) NOT NULL COMMENT '机柜名称/编号',
    `idc_id` int(10) unsigned NOT NULL COMMENT '所属机房ID',
    `position` varchar(100) DEFAULT '' COMMENT '位置描述',
    `unit_num` int(11) DEFAULT 42 COMMENT '机柜U数',
    `used_unit` int(11) DEFAULT 0 COMMENT '已用U位数',
    `power_kw` decimal(10,2) DEFAULT 0.00 COMMENT '额定功率(KW)',
    `status` int(11) DEFAULT 1 COMMENT '状态:1-启用,2-停用',
    `remark` text COMMENT '备注',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cmdb_cabinet_name` (`name`),
    KEY `idx_cmdb_cabinet_idc` (`idc_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CMDB机柜表';

-- 物理机表
CREATE TABLE IF NOT EXISTS `cmdb_physical_machine` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `sn` varchar(128) NOT NULL COMMENT '序列号/SN',
    `host_name` varchar(100) DEFAULT '' COMMENT '主机名',
    `manage_ip` varchar(64) DEFAULT '' COMMENT '管理IP(BMC/iLO/iDRAC)',
    `business_ip` varchar(64) DEFAULT '' COMMENT '业务IP',
    `brand` varchar(50) DEFAULT '' COMMENT '品牌',
    `model` varchar(100) DEFAULT '' COMMENT '型号',
    `cpu` varchar(100) DEFAULT '' COMMENT 'CPU信息',
    `memory` varchar(100) DEFAULT '' COMMENT '内存信息',
    `disk` varchar(255) DEFAULT '' COMMENT '磁盘信息',
    `raid` varchar(100) DEFAULT '' COMMENT 'RAID类型',
    `idc_id` int(10) unsigned DEFAULT NULL COMMENT '所属机房ID',
    `cabinet_id` int(10) unsigned DEFAULT NULL COMMENT '所属机柜ID',
    `unit_position` int(11) DEFAULT 0 COMMENT '机柜U位(起始)',
    `asset_status` int(11) DEFAULT 1 COMMENT '资产状态:1-在库,2-已上架,3-维修中,4-已下架,5-报废',
    `purchase_date` varchar(20) DEFAULT '' COMMENT '采购日期',
    `warranty_date` varchar(20) DEFAULT '' COMMENT '维保到期',
    `vendor` varchar(50) DEFAULT '' COMMENT '供应商',
    `status` int(11) DEFAULT 1 COMMENT '运行状态:1-运行中,2-关机,3-离线',
    `remark` text COMMENT '备注',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cmdb_physical_sn` (`sn`),
    KEY `idx_cmdb_physical_idc` (`idc_id`),
    KEY `idx_cmdb_physical_cabinet` (`cabinet_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CMDB物理机表';

-- 网络设备表
CREATE TABLE IF NOT EXISTS `cmdb_network_device` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `sn` varchar(128) DEFAULT NULL COMMENT '序列号/SN',
    `name` varchar(100) NOT NULL COMMENT '设备名称',
    `device_type` int(11) NOT NULL COMMENT '设备类型:1-路由器,2-交换机,3-防火墙,4-负载均衡,5-其他',
    `brand` varchar(50) DEFAULT '' COMMENT '品牌',
    `model` varchar(100) DEFAULT '' COMMENT '型号',
    `manage_ip` varchar(64) DEFAULT '' COMMENT '管理IP',
    `version` varchar(100) DEFAULT '' COMMENT '固件/系统版本',
    `port_num` int(11) DEFAULT 24 COMMENT '端口数量',
    `idc_id` int(10) unsigned DEFAULT NULL COMMENT '所属机房ID',
    `cabinet_id` int(10) unsigned DEFAULT NULL COMMENT '所属机柜ID',
    `asset_status` int(11) DEFAULT 1 COMMENT '资产状态:1-在库,2-已上架,3-维修中,4-已下架,5-报废',
    `purchase_date` varchar(20) DEFAULT '' COMMENT '采购日期',
    `warranty_date` varchar(20) DEFAULT '' COMMENT '维保到期',
    `status` int(11) DEFAULT 1 COMMENT '运行状态:1-运行中,2-关机,3-离线',
    `remark` text COMMENT '备注',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cmdb_network_name` (`name`),
    KEY `idx_cmdb_network_type` (`device_type`),
    KEY `idx_cmdb_network_idc` (`idc_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CMDB网络设备表';

-- 资产授权规则表（类似JumpServer）
CREATE TABLE IF NOT EXISTS `cmdb_asset_permission` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` varchar(100) NOT NULL COMMENT '授权规则名称',
    `description` text COMMENT '描述',
    `user_ids` text COMMENT '授权用户ID列表(JSON数组)',
    `group_ids` text COMMENT '授权用户组ID列表(JSON数组)',
    `asset_types` text COMMENT '授权资产类型(JSON数组)',
    `host_group_ids` text COMMENT '授权主机分组ID列表(JSON数组)',
    `physical_ids` text COMMENT '授权物理机ID列表(JSON数组)',
    `network_ids` text COMMENT '授权网络设备ID列表(JSON数组)',
    `database_ids` text COMMENT '授权数据库ID列表(JSON数组)',
    `idc_ids` text COMMENT '授权机房ID列表(JSON数组)',
    `permission_actions` text DEFAULT '["connect"]' COMMENT '权限操作(JSON数组: connect/upload/download/delete/admin)',
    `is_active` int(11) DEFAULT 1 COMMENT '是否启用:0-禁用,1-启用',
    `date_start` varchar(20) DEFAULT '' COMMENT '有效期开始(YYYY-MM-DD)',
    `date_expired` varchar(20) DEFAULT '' COMMENT '有效期结束(YYYY-MM-DD)',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cmdb_perm_name` (`name`),
    KEY `idx_cmdb_perm_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CMDB资产授权规则表';
