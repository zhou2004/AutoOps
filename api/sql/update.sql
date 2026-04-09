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
