-- MySQL dump 10.13  Distrib 8.0.33, for Win64 (x86_64)
--
-- Host: 172.22.107.76    Database: autoops
-- ------------------------------------------------------
-- Server version	8.0.46-0ubuntu0.22.04.3

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `app_application`
--

DROP TABLE IF EXISTS `app_application`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_application` (
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
) ENGINE=InnoDB AUTO_INCREMENT=22 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_application`
--

LOCK TABLES `app_application` WRITE;
/*!40000 ALTER TABLE `app_application` DISABLE KEYS */;
INSERT INTO `app_application` VALUES (21,'test','test',1,1,'','',NULL,NULL,NULL,'','','','',NULL,NULL,NULL,'{}',1,'2026-06-22 22:46:45.206','2026-06-22 22:46:45.206',NULL);
/*!40000 ALTER TABLE `app_application` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_jenkins_env`
--

DROP TABLE IF EXISTS `app_jenkins_env`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_jenkins_env` (
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
) ENGINE=InnoDB AUTO_INCREMENT=60 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_jenkins_env`
--

LOCK TABLES `app_jenkins_env` WRITE;
/*!40000 ALTER TABLE `app_jenkins_env` DISABLE KEYS */;
INSERT INTO `app_jenkins_env` VALUES (57,21,'prod',NULL,'',NULL,NULL,NULL,NULL,1,'2026-06-22 22:46:45.215','2026-06-22 22:46:45.215',NULL),(58,21,'test',NULL,'',NULL,NULL,NULL,NULL,1,'2026-06-22 22:46:45.224','2026-06-22 22:46:45.224',NULL),(59,21,'dev',NULL,'',NULL,NULL,NULL,NULL,1,'2026-06-22 22:46:45.234','2026-06-22 22:46:45.234',NULL);
/*!40000 ALTER TABLE `app_jenkins_env` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_service_release`
--

DROP TABLE IF EXISTS `app_service_release`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_service_release` (
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
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_service_release`
--

LOCK TABLES `app_service_release` WRITE;
/*!40000 ALTER TABLE `app_service_release` DISABLE KEYS */;
INSERT INTO `app_service_release` VALUES (2,'测试---发布工单上线',12,'影响用户登录',1,'管理员',102,'用户_102',98,'用户_98',89,'用户_89',2,2,2,'2025-11-22 18:16:13.748','2025-11-22 18:16:27.156','2025-11-22 18:16:36.689','同意','同意','同意',3,3,2,'2025-11-22 18:18:03.250','2025-11-22 18:21:45.450',220,1,'2025-11-22 18:13:37.037','2025-11-22 23:01:55.144',NULL),(3,'测试多任务工单发布',12,'测试多任务工单发布',1,'管理员',103,'用户_103',98,'用户_98',104,'用户_104',2,2,2,'2025-11-22 19:33:08.128','2025-11-22 19:34:25.711','2025-11-22 19:34:41.139','','','',4,1,3,'2025-11-22 19:36:09.589','2025-11-22 19:41:18.736',591,4,'2025-11-22 19:31:57.732','2025-11-22 19:41:18.789',NULL),(4,'测试审批流程',12,'测试审批流程',1,'管理员',103,'用户_103',98,'用户_98',104,'用户_104',2,2,2,'2025-11-22 20:28:35.898','2025-11-22 20:28:48.373','2025-11-22 20:28:55.066','','','',4,1,3,'2025-11-22 20:29:16.698','2025-11-22 20:34:22.328',303,2,'2025-11-22 20:00:18.967','2025-11-22 20:34:22.386',NULL),(5,'测试---发布工单上线',12,'测试---发布工单上线',1,'管理员',104,'用户_104',103,'用户_103',98,'用户_98',2,2,2,'2025-11-22 23:13:56.636','2025-11-22 23:16:26.840','2025-11-22 23:16:34.126','','通过','',3,3,2,'2025-11-22 23:52:17.626','2025-11-22 23:55:51.883',212,1,'2025-11-22 23:13:39.884','2025-11-23 00:50:11.059',NULL),(6,'双任务上线',14,'123',1,'管理员',104,'用户_104',103,'用户_103',98,'用户_98',2,2,2,'2025-11-23 11:33:16.353','2025-11-23 11:33:22.967','2025-11-23 12:57:50.091','','','111',1,1,1,NULL,NULL,0,2,'2025-11-23 11:32:19.689','2025-11-23 12:57:50.147',NULL),(7,'测试生产环境服务上线',18,'影响用户登录',1,'管理员',104,'王五',98,'李四',98,'李四',2,2,2,'2025-11-24 00:44:47.250','2025-11-24 00:45:40.169','2025-11-24 00:45:45.926','同意','','',3,3,2,'2025-11-24 00:46:52.721','2025-11-24 00:52:38.168',344,1,'2025-11-24 00:42:24.608','2025-11-24 00:53:04.351',NULL),(8,'tesst123',12,'影响用户登录',1,'管理员',89,'admin',89,'admin',89,'admin',2,2,2,'2025-12-05 20:47:13.905','2025-12-05 20:47:18.461','2025-12-05 20:47:27.382','ok','','',3,2,1,'2025-12-05 20:47:39.670','2025-12-05 20:47:53.645',12,1,'2025-12-05 20:46:53.205','2025-12-05 20:47:53.696',NULL);
/*!40000 ALTER TABLE `app_service_release` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_service_release_item`
--

DROP TABLE IF EXISTS `app_service_release_item`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_service_release_item` (
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
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_service_release_item`
--

LOCK TABLES `app_service_release_item` WRITE;
/*!40000 ALTER TABLE `app_service_release_item` DISABLE KEYS */;
INSERT INTO `app_service_release_item` VALUES (1,2,14,'dev-lku-sass-pack','dev-lku-sass-pack','dev-lku-sass-pack','git@code.dding.net/lockin/saas-toc-server.git','master','083a2d6669effe63670b0d5ee899d29eeb187abf','影响用户登录','','没有','没有','没有',35,'','{\"commit_id\":\"083a2d6669effe63670b0d5ee899d29eeb187abf\",\"compile\":\"true\"}',408,'http://test-ops-jenkins-tc2.dding.net:8080/job/dev-lku-sass-pack/408/console',3,'2025-11-22 18:18:03.823','2025-11-22 18:21:44.766',220,'',1,'2025-11-22 18:13:37.484','2025-11-22 18:21:44.821',NULL),(2,3,15,'dev-lku-sass-deploy','dev-lku-sass-deploy','dev-lku-sass-deploy','git@code.dding.net/lockin/saas-toc-server.git','master','c7417cbe54a11d89c819b4541316980dc1634687','测试多任务工单发布','','无','无','无',39,'','',19,'http://test-ops-jenkins-tc2.dding.net:8080/job/test1-saas-deploy/19/console',4,'2025-11-22 19:36:10.184','2025-11-22 19:36:37.458',27,'Jenkins构建失败',1,'2025-11-22 19:31:58.231','2025-11-22 19:36:37.524',NULL),(3,3,14,'dev-lku-sass-pack','dev-lku-sass-pack','dev-lku-sass-pack','git@code.dding.net/lockin/saas-toc-server.git','master','c7417cbe54a11d89c819b4541316980dc1634687','测试多任务工单发布','','无','无','无',35,'','',409,'http://test-ops-jenkins-tc2.dding.net:8080/job/dev-lku-sass-pack/409/console',3,'2025-11-22 19:36:10.184','2025-11-22 19:39:59.417',229,'',2,'2025-11-22 19:31:58.710','2025-11-22 19:39:59.476',NULL),(4,3,18,'test1-paas-deploy','test1-paas-deploy','test1-paas-deploy','git@code.dding.net/lockin/cloud-platform.git','master','c7417cbe54a11d89c819b4541316980dc1634687','测试多任务工单发布','','无','无','无',48,'','',13,'http://test-ops-jenkins-tc2.dding.net:8080/job/test1-paas-deploy/13/console',4,'2025-11-22 19:36:10.184','2025-11-22 19:36:38.200',28,'Jenkins构建失败',3,'2025-11-22 19:31:59.196','2025-11-22 19:36:38.282',NULL),(5,3,17,'test1-paas-pack','test1-paas-pack','test1-paas-pack','git@code.dding.net/lockin/cloud-platform.git','master','c7417cbe54a11d89c819b4541316980dc1634687','测试多任务工单发布','','无','无','无',45,'','',14,'http://test-ops-jenkins-tc2.dding.net:8080/job/test1-paas-pack/14/console',3,'2025-11-22 19:36:10.184','2025-11-22 19:41:18.055',307,'',4,'2025-11-22 19:31:59.673','2025-11-22 19:41:18.106',NULL),(6,4,17,'test1-paas-pack','test1-paas-pack','test1-paas-pack','git@code.dding.net/lockin/cloud-platform.git','master','c7417cbe54a11d89c819b4541316980dc1634687','测试审批流程','','测试审批流程','测试审批流程','测试审批流程',45,'','',15,'http://test-ops-jenkins-tc2.dding.net:8080/job/test1-paas-pack/15/console',3,'2025-11-22 20:29:17.308','2025-11-22 20:34:08.653',291,'',1,'2025-11-22 20:00:19.453','2025-11-22 20:34:08.746',NULL),(7,4,18,'test1-paas-deploy','test1-paas-deploy','test1-paas-deploy','git@code.dding.net/lockin/cloud-platform.git','master','c7417cbe54a11d89c819b4541316980dc1634687','测试审批流程','','测试审批流程','测试审批流程','测试审批流程',48,'','',14,'http://test-ops-jenkins-tc2.dding.net:8080/job/test1-paas-deploy/14/console',4,'2025-11-22 20:34:09.058','2025-11-22 20:34:21.625',12,'Jenkins构建失败',2,'2025-11-22 20:00:19.908','2025-11-22 20:34:21.687',NULL),(8,5,14,'dev-lku-sass-pack','dev-lku-sass-pack','dev-lku-sass-pack','git@code.dding.net/lockin/saas-toc-server.git','master','083a2d6669effe63670b0d5ee899d29eeb187abf','无','','无','无','无',35,'','',410,'http://test-ops-jenkins-tc2.dding.net:8080/job/dev-lku-sass-pack/410/console',3,'2025-11-22 23:52:18.208','2025-11-22 23:55:51.178',212,'',1,'2025-11-22 23:13:40.348','2025-11-22 23:55:51.229',NULL),(9,6,17,'test1-paas-pack','test1-paas-pack','test1-paas-pack','git@code.dding.net/lockin/cloud-platform.git','master','c7417cbe54a11d89c819b4541316980dc1634687','123','','123','123','123',45,'','{\"commit_id\":\"c7417cbe54a11d89c819b4541316980dc1634687\",\"compile\":\"true\"}',0,'',1,NULL,NULL,0,'',1,'2025-11-23 11:32:20.185','2025-11-23 11:32:20.185',NULL),(10,6,14,'dev-lku-sass-pack','dev-lku-sass-pack','dev-lku-sass-pack','git@code.dding.net/lockin/saas-toc-server.git','master','083a2d6669effe63670b0d5ee899d29eeb187abf','123','','123','123','123',35,'','{\"commit_id\":\"083a2d6669effe63670b0d5ee899d29eeb187abf\",\"compile\":\"true\"}',0,'',1,NULL,NULL,0,'',2,'2025-11-23 11:32:20.664','2025-11-23 11:32:20.664',NULL),(11,7,14,'dev-lku-sass-pack','dev-lku-sass-pack','dev-lku-sass-pack','git@code.dding.net/lockin/saas-toc-server.git','master','083a2d6669effe63670b0d5ee899d29eeb187abf','影响用户登录','','无','无','123',35,'','{\"commit_id\":\"083a2d6669effe63670b0d5ee899d29eeb187abf\",\"compile\":\"true\"}',411,'http://test-ops-jenkins-tc2.dding.net:8080/job/dev-lku-sass-pack/411/console',3,'2025-11-24 00:46:53.314','2025-11-24 00:52:37.448',344,'',1,'2025-11-24 00:42:25.108','2025-11-24 00:52:37.512',NULL),(12,8,19,'prod_saas3.0_data-export','prod-saas30-data-export','prod_saas3.0_data-export','git@gitee.com:zhang_fan1024/zf-k8s.git','master','','影响用户登录','','无','无','无',51,'','{\"commit_id\":\"123456789\"}',10,'http://test-ops-jenkins-tc2.dding.net:8080/job/prod_saas3.0_data-export/10/console',3,'2025-12-05 20:47:40.229','2025-12-05 20:47:52.992',12,'',1,'2025-12-05 20:46:53.654','2025-12-05 20:47:53.043',NULL);
/*!40000 ALTER TABLE `app_service_release_item` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_sh_release`
--

DROP TABLE IF EXISTS `app_sh_release`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_sh_release` (
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
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_sh_release`
--

LOCK TABLES `app_sh_release` WRITE;
/*!40000 ALTER TABLE `app_sh_release` DISABLE KEYS */;
INSERT INTO `app_sh_release` VALUES (1,'prod_saas3.0_data-export-数据导出','prod_saas3.0_data-export-数据导出',18,19,'prod_saas3.0_data-export','prod-saas30-data-export',1,'管理员',89,'admin',89,'admin','/home/dingding/saas3-data-export/','ls',2,'2025-11-28 12:42:48.493','111',3,2,'2025-11-28 12:43:08.943','2025-11-28 12:43:24.437',14,51,3,'http://test-ops-jenkins-tc2.dding.net:8080/job/prod_saas3.0_data-export/3/console','','2025-11-28 12:41:29.167','2025-11-28 12:43:24.488',NULL,'{\"commit_id\":\"123456789\"}',0,NULL,NULL,NULL),(2,'测试001','测试001',18,19,'prod_saas3.0_data-export','prod-saas30-data-export',1,'管理员',98,'李四',89,'admin','/home/','ls  /root/',2,'2025-11-28 14:18:58.866','同意',3,1,NULL,NULL,0,51,4,'http://test-ops-jenkins-tc2.dding.net:8080/job/prod_saas3.0_data-export/4/console','','2025-11-28 13:37:27.963','2025-11-28 14:47:38.777',NULL,'{\"commit_id\":\"123456789\"}',0,'2025-11-28 14:47:22.389','2025-11-28 14:47:38.725',NULL),(3,'测试脚本执行','测试脚本执行',18,19,'prod_saas3.0_data-export','prod-saas30-data-export',1,'管理员',89,'admin',89,'admin','/home/dingding/saas3-data-export/','pwd\nls\nhostname -I',2,'2025-11-28 15:19:15.004','111',6,2,'2025-11-28 15:24:27.057','2025-11-28 15:24:31.379',3,51,5,'http://test-ops-jenkins-tc2.dding.net:8080/job/prod_saas3.0_data-export/5/console','','2025-11-28 15:09:18.474','2025-11-28 15:24:31.432',NULL,'{\"commit_id\":\"123456789\"}',501,'2025-11-28 15:19:21.580','2025-11-28 15:19:34.271','bash: line 1: cd: /home/dingding/saas3-data-export/: No such file or directory\ndocker\nelk\njdk11\njdk17\njdk18\nluban-master\nnode\nprometheus\nsnap\n172.20.236.121 172.18.0.1 172.17.0.1 172.19.0.1 \n'),(4,'测试脚本002','测试脚本002',18,19,'prod_saas3.0_data-export','prod-saas30-data-export',1,'管理员',89,'admin',89,'admin','/home/dingding/saas3-data-export/','hostname\npwd\ndate',2,'2025-11-28 15:56:35.290','11',6,2,'2025-11-28 15:57:23.260','2025-11-28 15:57:27.569',3,51,6,'http://test-ops-jenkins-tc2.dding.net:8080/job/prod_saas3.0_data-export/6/console','','2025-11-28 15:56:13.417','2025-11-28 15:57:27.644',NULL,'{\"commit_id\":\"123456789\"}',501,'2025-11-28 15:56:44.717','2025-11-28 15:57:00.303','/root\nFri Nov 28 03:57:27 PM CST 2025\nbash: line 1: cd: /home/dingding/saas3-data-export/: No such file or directory\n'),(5,'测试002','测试002',18,19,'prod_saas3.0_data-export','prod-saas30-data-export',1,'管理员',89,'admin',89,'admin','/home/','pwd\nls\nhostname\n',2,'2025-11-28 16:12:04.059','ok',6,2,'2025-11-28 16:19:18.636','2025-11-28 16:19:22.516',3,51,7,'http://test-ops-jenkins-tc2.dding.net:8080/job/prod_saas3.0_data-export/7/console','','2025-11-28 16:10:22.217','2025-11-28 16:19:22.570',NULL,'{\"commit_id\":\"123456789\"}',501,'2025-11-28 16:12:11.076','2025-11-28 16:12:23.702','/home\ndevops\ndevops.tar.gz\ngo-ops\n'),(6,'test1111111111','test',18,19,'prod_saas3.0_data-export','prod-saas30-data-export',1,'管理员',89,'admin',89,'admin','/home/dingding/saas3-data-export/','pwd  \nls \nhostname',2,'2025-12-01 01:01:48.329','ok',6,2,'2025-12-01 01:02:36.996','2025-12-01 01:02:39.204',1,51,8,'http://test-ops-jenkins-tc2.dding.net:8080/job/prod_saas3.0_data-export/8/console','','2025-12-01 01:01:08.782','2025-12-01 01:02:39.260',NULL,'{\"commit_id\":\"123456789\"}',501,'2025-12-01 01:01:59.779','2025-12-01 01:02:16.154','/home/dingding/saas3-data-export\nxlsx\ngo-ops\n');
/*!40000 ALTER TABLE `app_sh_release` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_asset_permission`
--

DROP TABLE IF EXISTS `cmdb_asset_permission`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_asset_permission` (
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
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_asset_permission`
--

LOCK TABLES `cmdb_asset_permission` WRITE;
/*!40000 ALTER TABLE `cmdb_asset_permission` DISABLE KEYS */;
INSERT INTO `cmdb_asset_permission` VALUES (7,'test','','[106]','[]','[\"host\"]','[2,3,11,1]','[]','[]','[]','[]','[\"get\",\"list\"]',1,'','','2026-05-31 03:58:17.595','2026-06-01 00:10:39.800','[532,531,528,530]'),(8,'1','','[106]','[]','[\"network\"]','[]','[]','[5]','[]','[]','[\"get\",\"list\",\"delete\"]',1,'','','2026-05-31 04:23:17.008','2026-05-31 05:23:03.405','[]'),(9,'2','','[106]','[1]','[\"physical\"]','[]','[1,5]','[]','[]','[]','[\"get\",\"list\",\"update\"]',1,'','','2026-05-31 04:24:06.543','2026-06-26 02:07:44.389','[]');
/*!40000 ALTER TABLE `cmdb_asset_permission` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_cabinet`
--

DROP TABLE IF EXISTS `cmdb_cabinet`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_cabinet` (
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
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='CMDB机柜表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_cabinet`
--

LOCK TABLES `cmdb_cabinet` WRITE;
/*!40000 ALTER TABLE `cmdb_cabinet` DISABLE KEYS */;
INSERT INTO `cmdb_cabinet` VALUES (1,'A01',1,'A列3排',42,0,0,1,'','2026-05-30 17:38:57.454','2026-05-30 17:38:57.454',1),(7,'test',1,'',42,0,0,1,'','2026-05-30 18:19:13.933','2026-05-30 18:19:13.933',0),(8,'test2',3,'',42,0,0,1,'','2026-05-30 18:43:50.749','2026-05-31 00:11:20.779',0);
/*!40000 ALTER TABLE `cmdb_cabinet` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_credential_permission`
--

DROP TABLE IF EXISTS `cmdb_credential_permission`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_credential_permission` (
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_credential_permission`
--

LOCK TABLES `cmdb_credential_permission` WRITE;
/*!40000 ALTER TABLE `cmdb_credential_permission` DISABLE KEYS */;
/*!40000 ALTER TABLE `cmdb_credential_permission` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_group`
--

DROP TABLE IF EXISTS `cmdb_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '''父级分组ID''',
  `name` longtext NOT NULL COMMENT '''分组名称''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `remark` longtext COMMENT '''备注''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=37 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_group`
--

LOCK TABLES `cmdb_group` WRITE;
/*!40000 ALTER TABLE `cmdb_group` DISABLE KEYS */;
INSERT INTO `cmdb_group` VALUES (1,0,'默认业务组','2025-07-10 11:02:07.226',NULL,NULL),(2,0,'saas3业务线','2025-07-10 11:03:36.622',NULL,NULL),(3,0,'saas4业务线','2025-07-10 11:03:56.083',NULL,NULL),(4,11,'公寓业务','2025-07-10 11:05:45.431',NULL,NULL),(9,11,'智能窗帘业务','2025-07-17 10:17:52.017',NULL,NULL),(11,0,'saas5业务线','2025-07-17 10:21:44.700',NULL,NULL),(12,2,'saas3','2025-07-17 11:07:19.279',NULL,NULL),(14,3,'智慧门锁业务','2025-07-17 11:19:14.090',NULL,NULL),(16,15,'智能家居业务','2025-07-22 10:32:14.837',NULL,NULL),(18,2,'saas4','2025-07-23 10:40:12.787',NULL,NULL),(19,17,'运维组','2025-07-23 20:21:08.618',NULL,NULL),(20,17,'安全组','2025-07-23 20:21:33.222',NULL,NULL),(26,24,'国内鹿客云','2025-09-07 18:04:56.683',NULL,NULL),(27,24,'新加坡鹿客云','2025-09-07 18:05:13.713',NULL,NULL),(28,24,'北美鹿客云','2025-09-07 18:05:31.846',NULL,NULL),(29,17,'ai运维组','2025-10-01 15:17:35.617',NULL,NULL),(31,30,'rental-test','2025-12-05 20:17:32.169',NULL,NULL),(32,30,'rental-dev','2025-12-05 20:17:43.988',NULL,NULL),(34,33,'saas7','2025-12-09 21:14:55.863',NULL,NULL),(35,33,'saas8','2025-12-09 21:15:05.050',NULL,NULL);
/*!40000 ALTER TABLE `cmdb_group` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_host`
--

DROP TABLE IF EXISTS `cmdb_host`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_host` (
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
) ENGINE=InnoDB AUTO_INCREMENT=533 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_host`
--

LOCK TABLES `cmdb_host` WRITE;
/*!40000 ALTER TABLE `cmdb_host` DISABLE KEYS */;
INSERT INTO `cmdb_host` VALUES (528,'192.168.0.52',18,'192.168.0.52','119.4.87.30','root',24,22,'',1,'','','DebianGNU/Linux12(bookworm)',3,'4','','48','','2026-04-07 10:50:12.409',NULL,'2026-05-23 00:09:35.252','192.168.0.52','worker-01',NULL,NULL),(530,'192.168.0.51',18,'192.168.0.51','119.4.87.30','root',24,22,'',1,'','','DebianGNU/Linux12(bookworm)',1,'4','','48','','2026-05-10 09:40:56.817',NULL,'2026-05-23 00:09:16.259','192.168.0.51','master-01',NULL,NULL),(531,'192.168.0.53',18,'192.168.0.53','','root',24,22,'',1,'','','DebianGNU/Linux12(bookworm)',1,'4','','48','','2026-05-10 09:41:11.179',NULL,'2026-05-23 00:09:18.590','192.168.0.53','worker-02',NULL,NULL),(532,'172.22.107.76',18,'172.22.107.76','175.152.48.248','root',20,22,'',1,'','','Ubuntu22.04.5',1,'12','10','1007','','2026-05-17 15:29:42.483',NULL,'2026-05-31 03:56:27.727','172.22.107.76','zjj',NULL,NULL);
/*!40000 ALTER TABLE `cmdb_host` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_idc`
--

DROP TABLE IF EXISTS `cmdb_idc`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_idc` (
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
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_idc`
--

LOCK TABLES `cmdb_idc` WRITE;
/*!40000 ALTER TABLE `cmdb_idc` DISABLE KEYS */;
INSERT INTO `cmdb_idc` VALUES (1,'长沙高级机房','搞基','长沙市岳麓区','周俊杰','19174913526','T1',1,'高级机房','2026-05-30 13:03:32.863','2026-05-30 13:03:32.863'),(3,'test','','','','','T3',1,'','2026-05-30 18:42:55.826','2026-05-30 18:42:55.826');
/*!40000 ALTER TABLE `cmdb_idc` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_network_device`
--

DROP TABLE IF EXISTS `cmdb_network_device`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_network_device` (
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
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_network_device`
--

LOCK TABLES `cmdb_network_device` WRITE;
/*!40000 ALTER TABLE `cmdb_network_device` DISABLE KEYS */;
INSERT INTO `cmdb_network_device` VALUES (5,'test','test',2,'','','','',26,NULL,0,2,'','',1,'','2026-05-31 03:46:37.623','2026-05-31 04:55:14.264',0),(8,'test1','test1',2,'','','','',22,NULL,0,2,'','',1,'','2026-05-31 04:57:43.562','2026-05-31 04:59:05.609',3);
/*!40000 ALTER TABLE `cmdb_network_device` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_physical_machine`
--

DROP TABLE IF EXISTS `cmdb_physical_machine`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_physical_machine` (
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
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='CMDB物理机表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_physical_machine`
--

LOCK TABLES `cmdb_physical_machine` WRITE;
/*!40000 ALTER TABLE `cmdb_physical_machine` DISABLE KEYS */;
INSERT INTO `cmdb_physical_machine` VALUES (1,'test','','','','','','','256gb','','',1,1,0,2,'','','',1,'','2026-05-30 18:30:05.402','2026-06-01 00:14:49.519',NULL),(5,'test1','test11','','','','','','','','',3,0,0,2,'','','',1,'','2026-05-31 05:21:20.375','2026-05-31 05:22:25.615',NULL);
/*!40000 ALTER TABLE `cmdb_physical_machine` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_sql`
--

DROP TABLE IF EXISTS `cmdb_sql`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_sql` (
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
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_sql`
--

LOCK TABLES `cmdb_sql` WRITE;
/*!40000 ALTER TABLE `cmdb_sql` DISABLE KEYS */;
INSERT INTO `cmdb_sql` VALUES (1,'saas3-mysql',1,1,1,'prod','1111','2025-07-29 21:23:17.309','2025-07-29 21:23:17.309'),(2,'saas3-redis-1',3,3,1,'prod','1111','2025-07-29 21:24:57.985','2025-09-06 15:12:22.605'),(4,'saas3-pgsql',2,1,1,'prod','1111','2025-07-29 21:36:11.147','2025-09-06 15:12:44.586');
/*!40000 ALTER TABLE `cmdb_sql` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_sql_log`
--

DROP TABLE IF EXISTS `cmdb_sql_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_sql_log` (
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
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_sql_log`
--

LOCK TABLES `cmdb_sql_log` WRITE;
/*!40000 ALTER TABLE `cmdb_sql_log` DISABLE KEYS */;
INSERT INTO `cmdb_sql_log` VALUES (7,'8.130.14.34','gin-api','SELECT','select * from  cmdb_host;','admin','127.0.0.1',0,0,59,3,'SUCCESS','2025-08-25 10:17:24.672');
/*!40000 ALTER TABLE `cmdb_sql_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_sql_records`
--

DROP TABLE IF EXISTS `cmdb_sql_records`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_sql_records` (
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
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_sql_records`
--

LOCK TABLES `cmdb_sql_records` WRITE;
/*!40000 ALTER TABLE `cmdb_sql_records` DISABLE KEYS */;
INSERT INTO `cmdb_sql_records` VALUES (1,'8.130.14.34:3306','gin-api','SELECT','select * from cmdb_group;','anonymous',14,0,403,14,'SUCCESS','2025-07-29 11:14:13.686',''),(2,'8.130.14.30','saas3-mysql','SELECT','select * from cmdb_group;','',0,0,50,10,'SUCCESS','2025-07-30 13:29:24.409',''),(3,'8.130.14.34','gin-api','SELECT','select * from cmdb_group;','',0,0,54,14,'SUCCESS','2025-07-30 13:58:13.386',''),(4,'8.130.14.34','saas3-mysql','INSERT','UPDATE `cmdb_group`SET `name` = \'sql测试组0000\' WHERE `parent_id` = 17AND `name` = \'sql测试组\';','',0,1,80,0,'SUCCESS','2025-07-30 14:00:44.370',''),(5,'8.130.14.34','saas3-mysql','INSERT','UPDATE `cmdb_group` SET `name` = \'test123\' WHERE `id` = 22;','',0,1,80,0,'SUCCESS','2025-07-30 14:04:30.684',''),(6,'8.130.14.34','saas3-mysql','INSERT','UPDATE `cmdb_group` SET `name` = \'test123111\' WHERE `id` = 22;','',0,1,80,0,'SUCCESS','2025-07-30 14:06:34.692',''),(7,'8.130.14.34','gin-api','SELECT','UPDATE `cmdb_group` SET `name` = \'test123111\' WHERE `id` = 22;','',0,0,122,0,'SUCCESS','2025-07-30 14:07:44.151',''),(8,'8.130.14.34','gin-api','SELECT','UPDATE `cmdb_group` SET `name` = \'test001\' WHERE `id` = 22;','',0,0,55,0,'SUCCESS','2025-07-30 14:11:42.626',''),(9,'8.130.14.34','gin-api','SELECT','select * from cmdb_group;','',0,0,67,15,'SUCCESS','2025-07-30 15:24:57.109',''),(10,'8.130.14.34','gin-api','SELECT','select * from cmdb_host;','',0,0,114,75,'SUCCESS','2025-07-30 15:25:19.542',''),(11,'8.130.14.34','gin-api','SELECT','select * from cmdb_group;','',0,0,56,15,'SUCCESS','2025-07-30 15:29:16.507',''),(12,'8.130.14.34','saas3-mysql','EXECUTE','create databases  db;','',0,1,100,0,'SUCCESS','2025-07-30 15:34:31.246',''),(13,'8.130.14.34','gin-api','SELECT','select id,name from cmdb_group;','',0,0,51,15,'SUCCESS','2025-07-30 17:18:52.168',''),(14,'8.130.14.34','gin-api','SELECT','select id,name from cmdb_group;','',0,0,53,15,'SUCCESS','2025-07-30 17:29:36.465',''),(15,'8.130.14.34','gin-api','SELECT','select id,name from  cmdb_group;','',0,0,56,15,'SUCCESS','2025-07-30 21:06:07.136',''),(16,'8.130.14.34','gin-api','SELECT','select id,name from  cmdb_group;','',0,0,64,15,'SUCCESS','2025-07-30 21:12:04.886',''),(17,'8.130.14.34','gin-api','SELECT','select id,name from  cmdb_group;','admin',0,0,51,15,'SUCCESS','2025-07-30 21:23:55.532',''),(18,'8.130.14.34','gin-api','SELECT','select id,name from  cmdb_group;','zhangsan',0,0,54,15,'SUCCESS','2025-07-30 21:26:51.642','');
/*!40000 ALTER TABLE `cmdb_sql_records` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_user_group`
--

DROP TABLE IF EXISTS `cmdb_user_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_user_group` (
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
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_user_group`
--

LOCK TABLES `cmdb_user_group` WRITE;
/*!40000 ALTER TABLE `cmdb_user_group` DISABLE KEYS */;
INSERT INTO `cmdb_user_group` VALUES (1,'test','ops','',1,'2026-05-31 05:23:38.628','2026-05-31 05:23:38.628');
/*!40000 ALTER TABLE `cmdb_user_group` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cmdb_user_group_member`
--

DROP TABLE IF EXISTS `cmdb_user_group_member`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cmdb_user_group_member` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `group_id` bigint unsigned NOT NULL COMMENT '''用户组ID''',
  `user_id` bigint unsigned NOT NULL COMMENT '''用户ID''',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cmdb_ugm_group_user` (`group_id`,`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cmdb_user_group_member`
--

LOCK TABLES `cmdb_user_group_member` WRITE;
/*!40000 ALTER TABLE `cmdb_user_group_member` DISABLE KEYS */;
INSERT INTO `cmdb_user_group_member` VALUES (1,1,106,'2026-05-31 05:23:45.695');
/*!40000 ALTER TABLE `cmdb_user_group_member` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `config_account`
--

DROP TABLE IF EXISTS `config_account`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `config_account` (
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
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `config_account`
--

LOCK TABLES `config_account` WRITE;
/*!40000 ALTER TABLE `config_account` DISABLE KEYS */;
/*!40000 ALTER TABLE `config_account` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `config_ansible`
--

DROP TABLE IF EXISTS `config_ansible`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `config_ansible` (
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
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `config_ansible`
--

LOCK TABLES `config_ansible` WRITE;
/*!40000 ALTER TABLE `config_ansible` DISABLE KEYS */;
INSERT INTO `config_ansible` VALUES (1,'test',1,'[web] \nlocalhost ansible_ssh_port=22 ansible_ssh_user=root','','','system','2026-01-24 00:00:00.000','2026-03-06 00:30:14.466'),(2,'test2',2,'{\n    \"person_name\": \"李四\",\n    \"age\": \"22\"\n}','','','system','2026-01-24 00:00:00.000','2026-07-02 23:00:43.053'),(3,'test3',3,'\"person_name\": \"赵六\"\n\"age\": 32\n\n','fefw','','system','2026-01-24 00:00:00.000','2026-06-09 22:36:35.773'),(8,'2',1,'22','','system','system','2026-02-28 18:16:25.636','2026-02-28 18:16:25.636'),(9,'3',1,'3','','system','system','2026-02-28 18:16:30.469','2026-02-28 18:16:30.469'),(10,'4',1,'4','','system','system','2026-02-28 18:16:35.823','2026-02-28 18:16:35.823'),(11,'5',1,'5','','system','system','2026-02-28 18:16:41.045','2026-02-28 18:16:41.045'),(12,'6',1,'6','','system','system','2026-02-28 18:16:45.102','2026-02-28 18:16:45.102'),(13,'7',1,'7','','system','system','2026-02-28 18:16:48.801','2026-02-28 18:16:48.801'),(14,'8',1,'8','','system','system','2026-02-28 18:16:53.051','2026-02-28 18:16:53.051'),(17,'yaml',3,'\"person_name\": \"王五\"\n\"age\": 31','','system','system','2026-03-02 16:32:50.341','2026-03-02 16:32:50.341');
/*!40000 ALTER TABLE `config_ansible` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `config_ecsauth`
--

DROP TABLE IF EXISTS `config_ecsauth`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `config_ecsauth` (
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
) ENGINE=InnoDB AUTO_INCREMENT=25 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `config_ecsauth`
--

LOCK TABLES `config_ecsauth` WRITE;
/*!40000 ALTER TABLE `config_ecsauth` DISABLE KEYS */;
INSERT INTO `config_ecsauth` VALUES (20,'免密码认证',1,'root','123456','','2025-10-14 21:06:27.749','',22,NULL,22,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(24,'OPS',2,'ops','','-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABlwAAAAdzc2gtcn\nNhAAAAAwEAAQAAAYEAu3HXW0Rf6MLvYm0DMv1S0UrxA3PFAURyWoi60qPxqYDoBWmJWZ7S\ngaZU6n/BW6gO2yGHfpcp1Z8bcMXjr9HeCjHogWGbpkg0FsGOnsni4hR7sA3nYUWsKNfo3t\n/ObwkySUp5YDPa2T8G5ziu1TbWr4KhOIY3T6z6Sq5OmFthbmGQDiA/RtgMSm6vMDtF89Ja\n1Oh5P23fHgmmn7J7fSUM/kfxWYD34gJl3UFozhs/dovYSn3om4D57a4P318JOY+gTV1K+V\nPpnPhCWmvbKL9cl3U/rz0wsrDJssCFWtrCroW4jlM3/EMUgSvD6Tg/nl7VRrWOzPR8GYDl\n3PWzx0lkDsNBD2HoyQq6K83w5Vhz8xYIf4HbiGsL4iG/3Sg4HT4jLb2QOn+gh7xBupKwHM\nXoXvqPZjILb/5twoAWKeN1o/pqOPYXHo/QrI9bSXzW3sipFNcNOEUNrimkU7F1JSPvBwPB\n+BcQDvGhyF4Sv+/JmORLf/yM+/YR0CyOGQyeBm0LAAAFgNCHkVXQh5FVAAAAB3NzaC1yc2\nEAAAGBALtx11tEX+jC72JtAzL9UtFK8QNzxQFEclqIutKj8amA6AVpiVme0oGmVOp/wVuo\nDtshh36XKdWfG3DF46/R3gox6IFhm6ZINBbBjp7J4uIUe7AN52FFrCjX6N7fzm8JMklKeW\nAz2tk/Buc4rtU21q+CoTiGN0+s+kquTphbYW5hkA4gP0bYDEpurzA7RfPSWtToeT9t3x4J\npp+ye30lDP5H8VmA9+ICZd1BaM4bP3aL2Ep96JuA+e2uD99fCTmPoE1dSvlT6Zz4Qlpr2y\ni/XJd1P689MLKwybLAhVrawq6FuI5TN/xDFIErw+k4P55e1Ua1jsz0fBmA5dz1s8dJZA7D\nQQ9h6MkKuivN8OVYc/MWCH+B24hrC+Ihv90oOB0+Iy29kDp/oIe8QbqSsBzF6F76j2YyC2\n/+bcKAFinjdaP6ajj2Fx6P0KyPW0l81t7IqRTXDThFDa4ppFOxdSUj7wcDwfgXEA7xoche\nEr/vyZjkS3/8jPv2EdAsjhkMngZtCwAAAAMBAAEAAAGAB+gm8fIh65MhfvDjRCjcb/itzv\nv4sPN+sWP4IX+J56EI6IWJpi6laZOnHFc0RFYD/mldKlFdEeZKxYiLcLS1HY/6Y07HPo3o\nKJeUmQ0iFXBQwV3sxzUlrHljGNevAQ8NwHq0QQMe9bALbgB9m3/bMX8cpuI+fg1pZ3IP3a\nxpITtbVMU2dhoR9qfRmwK9Eipq63U8/Bh54237ydvBCsff0vAc9a4ThsScIGescJlEpY8D\nmizSfZDEuipv//elz3hH88ThCP9OLD2Z/B5pC6DZLZ5Sk20oICjarZzwsfHerKvDTN5nHR\nEknrr4Jj2iM4k8gcZ7VLn2VSm7nIH2zKEERKEwva3vxMCI6Zt27kypI/7zHJ/oM6uyfzQ6\nkQynvcEhqVomxm5ws1KRzp9sgwV1R917k3v1Xxem4EL9zNxeoZB10qdDM5ufqTLnoZvW0m\n//613E+JRilKazZl3zqWhcyEDKxqa9srt7HYLb4GzGqOXxjP6OXngdGjg/YzykTx1JAAAA\nwECBUZAq4RHVko5N5SXvrYV+huhCMW9Yhx16O/e4D9y1c8zOQ5smO183pgBJ4p/JSGItfH\nIL0afI3/uua2qc+uw2jg4nif31hb5tjUq/eNIKtNVeR2DuFNrv+6G01kQ53M3ZgcuEJQCu\nss0e6wWLQKn2I5FkNbO+87vqkRr4AfMLN8XxGWf/d0y1GbTZLpd+txFbKQBJXWmBADEt/j\nQBGnUInO6PHFa34CLBTZR7aw/ppkfKzfn9ZBmWMbK84CVT8wAAAMEA6t+oaejQSwrzxPMA\nna0tSTC+dp2bouEcRrp1zYESqbBeNqHSoEsboKHv/0jF2Nx6PLGttO11m0xaul7lMf62cQ\nkTNTss6EBaPwZ+lMjo7w/HVcJP5qII/5eNrCJj9X6Aze6zBPIAVX+zAL9Vc/SHsPCJ6n85\nUb0XDkmttZ2zdxYLUqgjSs/MH/4jwpAK9stXknWnWpAhEudV6LAFaJDVnlp/7JISMsizlP\nXHqB+Ez6gNqaqdLTlicsNm5CxLwZHJAAAAwQDMTg4tCvYsTg5MGB1x8HL0fUCPvaxeSzlw\nsGvLhum04HfHKvBEGYX4DRppSL/+snVuBuzhpH2dXCbqskgRuGv+XKmU0mCxfSD8/vVSrp\n3kZGvkbvNurYoAVkRokFXah54R2pzjlHdVihmoiKhbrsM5tLGKvkVDza5QZi9statmtsY7\n0KysjI1W8bl+XKq6lZ3fYeBBIQXiNn6oXlxNlo3kau8ILym1tpSP63mxMhh1puDS1mrfxw\nzFI7zOu5mkUjMAAAALcm9vdEBkZWJpYW4=\n-----END OPENSSH PRIVATE KEY-----','2026-01-19 13:39:56.921','',22,'',22,'',0,'','',0,'','');
/*!40000 ALTER TABLE `config_ecsauth` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `config_keymanage`
--

DROP TABLE IF EXISTS `config_keymanage`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `config_keymanage` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `key_type` bigint NOT NULL,
  `key_id` text NOT NULL,
  `key_secret` text NOT NULL,
  `remark` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `key_name` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `config_keymanage`
--

LOCK TABLES `config_keymanage` WRITE;
/*!40000 ALTER TABLE `config_keymanage` DISABLE KEYS */;
/*!40000 ALTER TABLE `config_keymanage` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `config_sync_schedule`
--

DROP TABLE IF EXISTS `config_sync_schedule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `config_sync_schedule` (
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
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `config_sync_schedule`
--

LOCK TABLES `config_sync_schedule` WRITE;
/*!40000 ALTER TABLE `config_sync_schedule` DISABLE KEYS */;
INSERT INTO `config_sync_schedule` VALUES (3,'阿里云定时同步','*/3 * * * *','[1]',0,'2025-10-01 13:33:00.001','2025-10-01 13:36:00.000','','2025-09-29 18:41:10.257','2025-10-01 13:34:41.439','[2025-10-01 13:33:00] 开始同步\n- 阿里云: 同步成功\n[2025-10-01 13:33:00] 同步完成，耗时: 827.31718ms\n');
/*!40000 ALTER TABLE `config_sync_schedule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `db`
--

DROP TABLE IF EXISTS `db`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db` (
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db`
--

LOCK TABLES `db` WRITE;
/*!40000 ALTER TABLE `db` DISABLE KEYS */;
/*!40000 ALTER TABLE `db` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `db_es_instance`
--

DROP TABLE IF EXISTS `db_es_instance`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db_es_instance` (
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
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db_es_instance`
--

LOCK TABLES `db_es_instance` WRITE;
/*!40000 ALTER TABLE `db_es_instance` DISABLE KEYS */;
/*!40000 ALTER TABLE `db_es_instance` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `db_export_task`
--

DROP TABLE IF EXISTS `db_export_task`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db_export_task` (
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
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db_export_task`
--

LOCK TABLES `db_export_task` WRITE;
/*!40000 ALTER TABLE `db_export_task` DISABLE KEYS */;
INSERT INTO `db_export_task` VALUES (1,'91c691d3-8e72-40d3-9de7-4453057ca1ee',1,'devops','full','completed','data/exports/devops_full_1765299147.sql',689403,'','2025-12-10 00:52:27.431','2025-12-10 00:52:40.023','2025-12-10 00:52:27.082','2025-12-10 00:52:40.023','',0),(2,'e1c6f2c1-7a85-43fb-8255-3d9a9fd229cb',1,'mayfly-go','full','completed','data/exports/mayfly-go_full_1765333697.sql',100718,'','2025-12-10 10:28:17.221','2025-12-10 10:28:28.959','2025-12-10 10:28:16.888','2025-12-10 10:28:28.959','',0),(3,'d788c68a-bab0-4d4c-8b6f-467d672876b1',2,'database_name','full','completed','data/exports/database_name_full_1765342425.sql',1307,'','2025-12-10 12:53:45.551','2025-12-10 12:53:46.084','2025-12-10 12:53:45.207','2025-12-10 12:53:46.084','',0),(4,'c6c4abdf-5dff-4a43-ba4c-64fac0dfc6de',1,'gin-api','full','completed','data/exports/gin-api_full_1765636630.sql',652608,'','2025-12-13 22:37:10.521','2025-12-13 22:37:24.556','2025-12-13 22:37:10.153','2025-12-13 22:37:24.556','',0);
/*!40000 ALTER TABLE `db_export_task` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `db_instance`
--

DROP TABLE IF EXISTS `db_instance`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db_instance` (
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
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db_instance`
--

LOCK TABLES `db_instance` WRITE;
/*!40000 ALTER TABLE `db_instance` DISABLE KEYS */;
INSERT INTO `db_instance` VALUES (5,'cb2ff431-c24c-4c11-b092-6564d41df2cb','1.225','mysql','192.168.1.225',3306,'tcp','','root','123456','',0,1,'2026-01-19 14:52:57.481','2026-01-19 14:53:13.050','',0,'',0);
/*!40000 ALTER TABLE `db_instance` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Temporary view structure for view `db_instance_all`
--

DROP TABLE IF EXISTS `db_instance_all`;
/*!50001 DROP VIEW IF EXISTS `db_instance_all`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `db_instance_all` AS SELECT 
 1 AS `id`,
 1 AS `code`,
 1 AS `name`,
 1 AS `db_type`,
 1 AS `sub_type`,
 1 AS `host`,
 1 AS `port`,
 1 AS `username`,
 1 AS `password`,
 1 AS `remark`,
 1 AS `ssh_tunnel_machine_id`,
 1 AS `status`,
 1 AS `create_time`,
 1 AS `update_time`,
 1 AS `creator`,
 1 AS `creator_id`,
 1 AS `modifier`,
 1 AS `modifier_id`,
 1 AS `connection_config`*/;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `db_mongo_instance`
--

DROP TABLE IF EXISTS `db_mongo_instance`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db_mongo_instance` (
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
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db_mongo_instance`
--

LOCK TABLES `db_mongo_instance` WRITE;
/*!40000 ALTER TABLE `db_mongo_instance` DISABLE KEYS */;
/*!40000 ALTER TABLE `db_mongo_instance` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `db_redis_instance`
--

DROP TABLE IF EXISTS `db_redis_instance`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db_redis_instance` (
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
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db_redis_instance`
--

LOCK TABLES `db_redis_instance` WRITE;
/*!40000 ALTER TABLE `db_redis_instance` DISABLE KEYS */;
/*!40000 ALTER TABLE `db_redis_instance` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `db_sql`
--

DROP TABLE IF EXISTS `db_sql`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db_sql` (
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db_sql`
--

LOCK TABLES `db_sql` WRITE;
/*!40000 ALTER TABLE `db_sql` DISABLE KEYS */;
/*!40000 ALTER TABLE `db_sql` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `db_sql_exec`
--

DROP TABLE IF EXISTS `db_sql_exec`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `db_sql_exec` (
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
) ENGINE=InnoDB AUTO_INCREMENT=467 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `db_sql_exec`
--

LOCK TABLES `db_sql_exec` WRITE;
/*!40000 ALTER TABLE `db_sql_exec` DISABLE KEYS */;
INSERT INTO `db_sql_exec` VALUES (1,1,'gin-api','APP_APPLICATION',0,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',2,'数据库不存在',110,'2025-11-29 18:05:04.883','',0),(2,1,'gin-api','APP_APPLICATION',0,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',2,'数据库不存在',112,'2025-11-29 18:05:05.487','',0),(3,1,'gin-api','CMDB_GROUP',0,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',2,'数据库不存在',119,'2025-11-29 18:05:10.402','',0),(4,1,'gin-api','CMDB_GROUP',0,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',2,'数据库不存在',116,'2025-11-29 18:05:10.598','',0),(5,1,'gin-api','CMDB_SQL',0,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',2,'数据库不存在',122,'2025-11-29 18:05:17.424','',0),(6,1,'gin-api','CMDB_SQL',0,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',2,'数据库不存在',136,'2025-11-29 18:05:17.580','',0),(7,1,'gin-api','APP_JENKINS_ENV',0,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',2,'数据库不存在',119,'2025-11-29 18:11:06.550','',0),(8,1,'gin-api','APP_JENKINS_ENV',0,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',2,'数据库不存在',493,'2025-11-29 18:11:06.802','',0),(9,1,'gin-api','APP_JENKINS_ENV',0,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',2,'数据库不存在',127,'2025-11-29 18:11:15.554','',0),(10,1,'gin-api','APP_JENKINS_ENV',0,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',2,'数据库不存在',110,'2025-11-29 18:11:15.687','',0),(11,1,'gin-api','APP_SERVICE_RELEASE_ITEM',1,'SELECT * FROM app_service_release_item LIMIT 20 OFFSET 0','','',1,'执行成功',548,'2025-11-29 18:28:12.401','',0),(12,1,'gin-api','APP_SERVICE_RELEASE_ITEM',1,'SELECT * FROM app_service_release_item LIMIT 20 OFFSET 0','','',1,'执行成功',1083,'2025-11-29 18:28:12.656','',0),(13,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',277,'2025-11-29 18:28:25.668','',0),(14,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',281,'2025-11-29 18:28:25.951','',0),(15,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',308,'2025-11-29 18:31:23.236','',0),(16,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',346,'2025-11-29 18:31:23.606','',0),(17,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',298,'2025-11-29 18:35:52.379','',0),(18,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',303,'2025-11-29 18:35:52.538','',0),(19,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',266,'2025-11-29 18:35:53.254','',0),(20,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',259,'2025-11-29 18:35:53.562','',0),(21,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',283,'2025-11-29 18:37:11.404','',0),(22,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',320,'2025-11-29 18:37:12.000','',0),(23,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',273,'2025-11-29 18:37:11.713','',0),(24,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',299,'2025-11-29 18:37:12.382','',0),(25,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',255,'2025-11-29 18:37:12.911','',0),(26,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',282,'2025-11-29 18:37:13.296','',0),(27,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',248,'2025-11-29 18:37:58.462','',0),(28,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',260,'2025-11-29 18:37:59.326','',0),(29,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',275,'2025-11-29 18:38:46.987','',0),(30,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',277,'2025-11-29 18:38:47.151','',0),(31,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',265,'2025-11-29 18:38:47.925','',0),(32,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',266,'2025-11-29 18:38:48.246','',0),(33,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-11-29 18:39:00.606','',0),(34,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',304,'2025-11-29 18:39:00.775','',0),(35,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',287,'2025-11-29 18:39:01.496','',0),(36,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',263,'2025-11-29 18:39:01.812','',0),(37,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',403,'2025-11-29 18:48:01.028','',0),(38,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',335,'2025-11-29 18:48:01.099','',0),(39,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',10295,'2025-11-29 18:48:15.394','',0),(40,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',2650,'2025-11-29 18:48:14.638','',0),(41,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',6494,'2025-11-29 18:48:15.978','',0),(42,1,'devops','CMDB_HOST',1,'SELECT COUNT(*) as total FROM cmdb_host','','',1,'执行成功',293,'2025-11-29 18:48:16.898','',0),(43,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',3692,'2025-11-29 18:48:12.891','',0),(44,1,'devops','CMDB_HOST',1,'SELECT COUNT(*) as total FROM cmdb_host','','',1,'执行成功',281,'2025-11-29 18:48:21.045','',0),(45,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',340,'2025-11-29 18:50:12.488','',0),(46,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',284,'2025-11-29 18:50:13.429','',0),(47,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',304,'2025-11-29 18:54:02.563','',0),(48,1,'devops','APP_JENKINS_ENV',1,'SELECT COUNT(*) as total FROM app_jenkins_env','','',1,'执行成功',294,'2025-11-29 18:54:03.463','',0),(49,1,'devops','CMDB_GROUP',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',328,'2025-11-29 18:54:07.320','',0),(50,1,'devops','CMDB_GROUP',1,'SELECT COUNT(*) as total FROM cmdb_group','','',1,'执行成功',313,'2025-11-29 18:54:08.231','',0),(51,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',282,'2025-11-29 18:55:28.076','',0),(52,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',280,'2025-11-29 18:55:28.949','',0),(53,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-11-29 18:55:29.997','',0),(54,1,'devops','APP_JENKINS_ENV',1,'SELECT COUNT(*) as total FROM app_jenkins_env','','',1,'执行成功',277,'2025-11-29 18:55:30.867','',0),(55,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',282,'2025-11-29 18:56:33.893','',0),(56,1,'devops','APP_JENKINS_ENV',1,'SELECT COUNT(*) as total FROM app_jenkins_env','','',1,'执行成功',275,'2025-11-29 18:56:34.766','',0),(57,1,'devops','CMDB_GROUP',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',309,'2025-11-29 18:56:37.254','',0),(58,1,'devops','CMDB_GROUP',1,'SELECT COUNT(*) as total FROM cmdb_group','','',1,'执行成功',293,'2025-11-29 18:56:38.144','',0),(59,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',276,'2025-11-29 18:57:16.850','',0),(60,1,'devops','CMDB_HOST',1,'SELECT COUNT(*) as total FROM cmdb_host','','',1,'执行成功',284,'2025-11-29 18:57:17.734','',0),(61,1,'devops','CMDB_SQL_LOG',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',402,'2025-11-29 18:57:49.376','',0),(62,1,'devops','CMDB_SQL_LOG',1,'SELECT COUNT(*) as total FROM cmdb_sql_log','','',1,'执行成功',274,'2025-11-29 18:57:50.233','',0),(63,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',315,'2025-11-29 19:00:49.626','',0),(64,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',330,'2025-11-29 19:00:50.546','',0),(65,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',484,'2025-11-29 19:00:57.485','',0),(66,1,'devops','CMDB_HOST',1,'SELECT COUNT(*) as total FROM cmdb_host','','',1,'执行成功',656,'2025-11-29 19:00:59.337','',0),(67,1,'devops','CMDB_SQL',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',432,'2025-11-29 19:01:33.694','',0),(68,1,'devops','CMDB_SQL',1,'SELECT COUNT(*) as total FROM cmdb_sql','','',1,'执行成功',285,'2025-11-29 19:01:34.585','',0),(69,1,'devops','CONFIG_ACCOUNT',1,'SELECT * FROM config_account LIMIT 20 OFFSET 0','','',1,'执行成功',300,'2025-11-29 19:01:52.297','',0),(70,1,'devops','CONFIG_ACCOUNT',1,'SELECT COUNT(*) as total FROM config_account','','',1,'执行成功',282,'2025-11-29 19:01:53.222','',0),(71,1,'devops','CMDB_SQL_LOG',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',442,'2025-11-29 19:02:12.743','',0),(72,1,'devops','CMDB_SQL_LOG',1,'SELECT COUNT(*) as total FROM cmdb_sql_log','','',1,'执行成功',388,'2025-11-29 19:02:13.732','',0),(73,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',321,'2025-11-29 19:04:24.978','',0),(74,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',291,'2025-11-29 19:04:25.863','',0),(75,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',286,'2025-11-29 19:05:02.134','',0),(76,1,'devops','CMDB_HOST',1,'SELECT COUNT(*) as total FROM cmdb_host','','',1,'执行成功',309,'2025-11-29 19:05:03.008','',0),(77,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',361,'2025-11-29 19:06:09.051','',0),(78,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',291,'2025-11-29 19:06:10.002','',0),(79,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',1109,'2025-11-29 19:06:32.787','',0),(80,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',1586,'2025-11-29 19:06:33.149','',0),(81,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',303,'2025-11-29 19:06:33.300','',0),(82,1,'gin-api','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',552,'2025-11-29 19:06:33.428','',0),(83,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',307,'2025-11-29 19:06:33.766','',0),(84,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',254,'2025-11-29 19:06:34.069','',0),(85,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',553,'2025-11-29 19:06:34.715','',0),(86,1,'gin-api','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',302,'2025-11-29 19:06:35.225','',0),(87,1,'gin-api','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',295,'2025-11-29 19:06:36.173','',0),(88,1,'gin-api','APP_JENKINS_ENV',1,'SELECT COUNT(*) as total FROM app_jenkins_env','','',1,'执行成功',276,'2025-11-29 19:06:37.028','',0),(89,1,'devops','CMDB_SQL',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',275,'2025-11-29 19:08:02.114','',0),(90,1,'devops','CMDB_SQL',1,'SELECT COUNT(*) as total FROM cmdb_sql','','',1,'执行成功',331,'2025-11-29 19:08:03.077','',0),(91,1,'devops','CONFIG_ACCOUNT',1,'SELECT * FROM config_account LIMIT 20 OFFSET 0','','',1,'执行成功',324,'2025-11-29 19:08:59.699','',0),(92,1,'devops','CMDB_SQL_RECORDS',1,'SELECT * FROM cmdb_sql_records LIMIT 20 OFFSET 0','','',1,'执行成功',299,'2025-11-29 19:08:59.964','',0),(93,1,'devops','CMDB_SQL_RECORDS',1,'SELECT * FROM cmdb_sql_records LIMIT 20 OFFSET 0','','',1,'执行成功',275,'2025-11-29 19:09:00.054','',0),(94,1,'devops','CMDB_SQL_RECORDS',1,'SELECT * FROM cmdb_sql_records LIMIT 20 OFFSET 0','','',1,'执行成功',259,'2025-11-29 19:09:00.169','',0),(95,1,'devops','CONFIG_ACCOUNT',1,'SELECT * FROM config_account LIMIT 20 OFFSET 0','','',1,'执行成功',360,'2025-11-29 19:08:59.813','',0),(96,1,'devops','CONFIG_ACCOUNT',1,'SELECT COUNT(*) as total FROM config_account','','',1,'执行成功',292,'2025-11-29 19:09:00.602','',0),(97,1,'devops','CMDB_SQL_RECORDS',1,'SELECT COUNT(*) as total FROM cmdb_sql_records','','',1,'执行成功',303,'2025-11-29 19:09:00.855','',0),(98,1,'devops','CONFIG_ACCOUNT',1,'SELECT COUNT(*) as total FROM config_account','','',1,'执行成功',326,'2025-11-29 19:09:01.089','',0),(99,1,'devops','CMDB_SQL_RECORDS',1,'SELECT COUNT(*) as total FROM cmdb_sql_records','','',1,'执行成功',295,'2025-11-29 19:09:01.217','',0),(100,1,'devops','CMDB_SQL_RECORDS',1,'SELECT COUNT(*) as total FROM cmdb_sql_records','','',1,'执行成功',334,'2025-11-29 19:09:01.611','',0),(101,1,'devops','CMDB_SQL',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',280,'2025-11-29 19:09:14.451','',0),(102,1,'devops','CMDB_SQL',1,'SELECT COUNT(*) as total FROM cmdb_sql','','',1,'执行成功',278,'2025-11-29 19:09:15.298','',0),(103,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',1029,'2025-11-29 19:09:29.074','',0),(104,1,'devops','CMDB_HOST',1,'SELECT COUNT(*) as total FROM cmdb_host','','',1,'执行成功',6538,'2025-11-29 19:09:36.486','',0),(105,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',264,'2025-11-29 19:23:51.118','',0),(106,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',296,'2025-11-29 19:23:51.990','',0),(107,1,'devops','CMDB_SQL',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',295,'2025-11-29 19:23:59.137','',0),(108,1,'devops','CMDB_SQL',1,'SELECT COUNT(*) as total FROM cmdb_sql','','',1,'执行成功',279,'2025-11-29 19:24:00.023','',0),(109,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 100;','','',1,'执行成功',280,'2025-11-29 19:25:10.430','',0),(110,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',284,'2025-11-29 19:49:42.427','',0),(111,1,'devops','APP_APPLICATION',1,'SELECT COUNT(*) as total FROM app_application','','',1,'执行成功',287,'2025-11-29 19:49:43.313','',0),(112,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',239,'2025-11-29 19:57:17.599','',0),(113,1,'devops','CMDB_SQL_LOG',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',213,'2025-11-29 19:57:40.935','',0),(114,1,'devops','CMDB_SQL',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',213,'2025-11-29 19:57:40.616','',0),(115,1,'devops','CMDB_SQL_LOG',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',212,'2025-11-29 19:57:40.797','',0),(116,1,'devops','CMDB_SQL_LOG',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',210,'2025-11-29 19:57:59.070','',0),(117,1,'devops','CMDB_SQL_RECORDS',1,'SELECT * FROM cmdb_sql_records LIMIT 20 OFFSET 0','','',1,'执行成功',248,'2025-11-29 19:58:15.888','',0),(118,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',307,'2025-11-29 19:58:47.223','',0),(119,1,'devops','CMDB_HOST;',1,'SELECT * FROM cmdb_host;','','',1,'执行成功',212,'2025-11-29 20:00:09.885','',0),(120,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',263,'2025-11-29 20:03:26.517','',0),(121,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',213,'2025-11-29 20:03:44.248','',0),(122,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 100;','','',1,'执行成功',220,'2025-11-29 20:04:04.538','',0),(123,1,'devops','TASK_WORK',1,'SELECT * FROM task_work LIMIT 20 OFFSET 0','','',1,'执行成功',234,'2025-11-29 20:04:29.401','',0),(124,1,'devops','K8S_CLUSTER',1,'SELECT * FROM k8s_cluster LIMIT 20 OFFSET 0','','',1,'执行成功',247,'2025-11-29 20:05:20.336','',0),(125,1,'devops','CONFIG_ECSAUTH',1,'SELECT * FROM config_ecsauth LIMIT 20 OFFSET 0','','',1,'执行成功',515,'2025-11-29 20:07:54.567','',0),(126,1,'devops','CONFIG_ACCOUNT',1,'SELECT * FROM config_account LIMIT 20 OFFSET 0','','',1,'执行成功',219,'2025-11-29 20:08:12.059','',0),(127,1,'devops','CONFIG_ECSAUTH',1,'SELECT * FROM config_ecsauth LIMIT 20 OFFSET 0','','',1,'执行成功',386,'2025-11-29 20:08:56.093','',0),(128,1,'devops','CONFIG_ACCOUNT',1,'SELECT * FROM config_account LIMIT 20 OFFSET 0','','',1,'执行成功',212,'2025-11-29 20:09:15.428','',0),(129,1,'devops','CONFIG_ACCOUNT',1,'SELECT * FROM config_account LIMIT 20 OFFSET 0','','',1,'执行成功',241,'2025-11-29 20:09:18.899','',0),(130,1,'devops','SYS_MENU',1,'SELECT * FROM sys_menu LIMIT 20 OFFSET 0','','',1,'执行成功',211,'2025-11-29 20:09:31.833','',0),(131,1,'devops','CONFIG_KEYMANAGE',1,'SELECT * FROM config_keymanage LIMIT 20 OFFSET 0','','',1,'执行成功',218,'2025-11-29 20:10:16.736','',0),(132,1,'gin-api','APP_SERVICE_RELEASE',1,'SELECT * FROM app_service_release LIMIT 20 OFFSET 0','','',1,'执行成功',229,'2025-11-29 20:10:41.975','',0),(133,1,'gin-api','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',321,'2025-11-29 20:11:33.117','',0),(134,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',213,'2025-11-29 20:14:46.781','',0),(135,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',213,'2025-11-29 20:17:07.095','',0),(136,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',211,'2025-11-29 20:17:20.352','',0),(137,1,'devops','CMDB_GROUP',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',240,'2025-11-29 20:27:29.297','',0),(138,1,'devops','CMDB_SQL',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',258,'2025-11-29 20:28:55.241','',0),(139,1,'devops','CMDB_SQL',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',579,'2025-11-29 20:28:55.466','',0),(140,1,'devops','CMDB_GROUP',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',205,'2025-11-29 20:30:47.480','',0),(141,1,'devops','CMDB_HOST',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',199,'2025-11-29 20:30:58.495','',0),(142,1,'devops','CMDB_GROUP',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-11-29 20:34:21.225','',0),(143,1,'devops','CMDB_GROUP',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',4246,'2025-11-29 20:45:39.121','',0),(144,1,'devops','CMDB_GROUP',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',582,'2025-11-29 20:49:42.415','',0),(145,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',212,'2025-11-29 20:59:18.272','',0),(146,1,'devops','APP_APPLICATION',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-11-29 21:02:11.486','',0),(147,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',242,'2025-11-29 21:05:56.924','',0),(148,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',269,'2025-11-29 21:06:38.995','',0),(149,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',327,'2025-11-29 21:08:52.177','',0),(150,1,'devops','APP_JENKINS_ENV',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',646,'2025-11-29 21:10:07.252','',0),(151,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 100;','','',1,'执行成功',293,'2025-11-29 21:18:18.161','',0),(152,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',320,'2025-11-29 21:21:01.619','',0),(153,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',297,'2025-11-29 21:21:10.168','',0),(154,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',326,'2025-11-29 21:21:27.886','',0),(155,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',283,'2025-11-29 21:22:09.300','',0),(156,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',283,'2025-11-29 21:22:19.066','',0),(157,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',276,'2025-11-29 21:22:21.160','',0),(158,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',264,'2025-11-29 21:34:45.094','',0),(159,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',1520,'2025-11-29 21:35:20.063','',0),(160,1,'sys','sys_config',1,'SELECT * FROM sys_config LIMIT 20 OFFSET 0','','',1,'执行成功',293,'2025-11-29 21:36:51.862','',0),(161,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-11-29 21:37:07.653','',0),(162,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',292,'2025-11-29 21:37:26.201','',0),(163,1,'devops','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',298,'2025-11-29 21:37:48.390','',0),(164,1,'devops','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',277,'2025-11-29 21:38:13.100','',0),(165,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',282,'2025-11-29 21:40:52.663','',0),(166,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',283,'2025-11-29 21:42:31.577','',0),(167,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',283,'2025-11-29 21:45:10.707','',0),(168,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',285,'2025-11-29 21:45:23.126','',0),(169,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',270,'2025-11-29 21:45:42.521','',0),(170,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',491,'2025-11-29 21:46:03.897','',0),(171,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',295,'2025-11-29 21:50:35.944','',0),(172,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',270,'2025-11-29 21:50:58.532','',0),(173,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',258,'2025-11-29 21:51:00.879','',0),(174,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',266,'2025-11-29 21:54:42.129','',0),(175,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',299,'2025-11-29 21:57:35.040','',0),(176,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',408,'2025-11-29 21:57:38.152','',0),(177,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',279,'2025-11-29 21:57:43.563','',0),(178,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',341,'2025-11-29 21:57:53.234','',0),(179,1,'devops','config_ecsauth',1,'SELECT * FROM config_ecsauth LIMIT 20 OFFSET 0','','',1,'执行成功',309,'2025-11-29 21:58:20.086','',0),(180,1,'devops','config_ecsauth',1,'SELECT * FROM config_ecsauth LIMIT 20 OFFSET 0','','',1,'执行成功',579,'2025-11-29 21:58:20.467','',0),(181,1,'devops','config_ecsauth',1,'SELECT * FROM config_ecsauth LIMIT 20 OFFSET 0','','',1,'执行成功',299,'2025-11-29 21:58:28.291','',0),(182,1,'devops','monitor_agent',1,'SELECT * FROM monitor_agent LIMIT 20 OFFSET 0','','',1,'执行成功',277,'2025-11-29 21:58:36.381','',0),(183,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',330,'2025-11-29 21:59:23.190','',0),(184,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',450,'2025-11-29 21:59:44.784','',0),(185,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',295,'2025-11-29 22:10:51.726','',0),(186,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',266,'2025-11-29 22:10:54.896','',0),(187,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',323,'2025-11-29 22:11:01.210','',0),(188,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',288,'2025-11-29 22:11:01.374','',0),(189,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',319,'2025-11-29 22:14:15.745','',0),(190,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',508,'2025-11-29 22:14:16.040','',0),(191,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',261,'2025-11-29 22:14:22.801','',0),(192,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',278,'2025-11-29 22:14:29.071','',0),(193,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',587,'2025-11-29 22:17:49.851','',0),(194,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',273,'2025-11-29 22:21:41.845','',0),(195,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',297,'2025-11-29 22:21:41.996','',0),(196,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',338,'2025-11-29 22:21:41.638','',0),(197,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',935,'2025-11-29 22:21:42.359','',0),(198,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',280,'2025-11-29 22:21:45.552','',0),(199,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',290,'2025-11-29 22:25:38.566','',0),(200,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',262,'2025-11-29 22:26:39.755','',0),(201,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',299,'2025-11-29 22:29:18.342','',0),(202,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',288,'2025-11-29 22:30:54.805','',0),(203,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',282,'2025-11-29 22:31:17.619','',0),(204,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',289,'2025-11-29 22:32:34.668','',0),(205,1,'devops','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',272,'2025-11-29 22:32:43.903','',0),(206,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',312,'2025-11-29 22:32:59.875','',0),(207,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',268,'2025-11-29 22:38:19.169','',0),(208,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',285,'2025-11-29 22:42:46.982','',0),(209,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',283,'2025-11-29 22:48:01.180','',0),(210,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',279,'2025-11-29 22:53:17.671','',0),(211,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',271,'2025-11-29 22:57:40.841','',0),(212,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',262,'2025-11-29 22:59:48.099','',0),(213,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',278,'2025-11-29 23:00:34.141','',0),(214,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',569,'2025-11-29 23:04:04.638','',0),(215,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',542,'2025-11-29 23:10:59.541','',0),(216,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',309,'2025-11-29 23:13:35.399','',0),(217,1,'gin-api','app_sh_release',1,'SELECT * FROM app_sh_release LIMIT 20 OFFSET 0','','',1,'执行成功',316,'2025-11-29 23:17:36.816','',0),(218,1,'gin-api','app_sh_release',1,'SELECT * FROM app_sh_release LIMIT 20 OFFSET 0','','',1,'执行成功',713,'2025-11-29 23:17:37.046','',0),(219,1,'gin-api','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',311,'2025-11-29 23:17:41.461','',0),(220,1,'gin-api','config_ecsauth',1,'SELECT * FROM config_ecsauth LIMIT 20 OFFSET 0','','',1,'执行成功',451,'2025-11-29 23:18:01.612','',0),(221,1,'gin-api','config_keymanage',1,'SELECT * FROM config_keymanage LIMIT 20 OFFSET 0','','',1,'执行成功',300,'2025-11-29 23:18:03.029','',0),(222,1,'gin-api','config_sync_schedule',1,'SELECT * FROM config_sync_schedule LIMIT 20 OFFSET 0','','',1,'执行成功',305,'2025-11-29 23:18:04.708','',0),(223,1,'gin-api','sys_operation_log',1,'SELECT * FROM sys_operation_log LIMIT 20 OFFSET 0','','',1,'执行成功',303,'2025-11-29 23:18:07.975','',0),(224,1,'gin-api','sys_operation_log',1,'SELECT * FROM sys_operation_log LIMIT 20 OFFSET 0','','',1,'执行成功',334,'2025-11-29 23:18:13.728','',0),(225,1,'gin-api','app_service_release',1,'SELECT * FROM app_service_release LIMIT 20 OFFSET 0','','',1,'执行成功',304,'2025-11-29 23:26:31.908','',0),(226,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',420,'2025-11-29 23:30:01.262','',0),(227,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',331,'2025-11-29 23:35:23.442','',0),(228,1,'devops','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',306,'2025-11-29 23:36:11.800','',0),(229,1,'devops','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',293,'2025-11-29 23:36:35.095','',0),(230,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',297,'2025-11-29 23:39:22.251','',0),(231,1,'gin-api','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',340,'2025-11-29 23:39:24.638','',0),(232,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',289,'2025-11-29 23:39:32.922','',0),(233,1,'devops','',5,'RENAME TABLE app_jenkins_env_copy_20251129233940 TO app_jenkins_env_copy_20251129233940_new111','','',1,'执行成功',298,'2025-11-29 23:39:51.148','',0),(234,1,'devops','app_jenkins_env_copy_20251129233940_new111',1,'SELECT * FROM app_jenkins_env_copy_20251129233940_new111 LIMIT 20 OFFSET 0','','',1,'执行成功',563,'2025-11-29 23:40:20.699','',0),(235,1,'devops','',5,'DROP TABLE app_jenkins_env_copy_20251129233940_new111','','',1,'执行成功',291,'2025-11-29 23:40:30.953','',0),(236,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',434,'2025-11-29 23:44:31.265','',0),(237,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',466,'2025-11-29 23:45:21.575','',0),(238,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',288,'2025-11-29 23:52:27.278','',0),(239,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',302,'2025-11-29 23:55:28.566','',0),(240,1,'gin-api','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',319,'2025-11-29 23:58:06.595','',0),(241,2,'postgres','',5,'CREATE DATABASE database_name;','','',1,'执行成功',599,'2025-11-30 00:56:01.163','',0),(242,2,'database_name','',5,'CREATE TABLE users (\n    id SERIAL PRIMARY KEY,\n    name VARCHAR(100) NOT NULL,\n    email VARCHAR(150) UNIQUE NOT NULL\n);','','',1,'执行成功',3259,'2025-11-30 01:10:07.720','',0),(243,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',4965,'2025-11-30 01:10:34.369','',0),(244,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',198,'2025-11-30 01:10:34.867','',0),(245,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',201,'2025-11-30 01:10:35.037','',0),(246,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',208,'2025-11-30 01:10:35.181','',0),(247,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',208,'2025-11-30 01:10:35.442','',0),(248,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',192,'2025-11-30 01:10:35.755','',0),(249,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',205,'2025-11-30 01:10:35.988','',0),(250,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',202,'2025-11-30 01:10:39.921','',0),(251,2,'database_name','users',2,'INSERT INTO users (name, email) VALUES (\'张三\', \'zhangsan@123.com\')','','',1,'执行成功',376,'2025-11-30 01:11:20.911','',0),(252,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',299,'2025-11-30 01:11:21.506','',0),(253,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',299,'2025-11-30 11:38:28.572','',0),(254,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',275,'2025-11-30 11:38:44.346','',0),(255,1,'gin-api','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',272,'2025-11-30 11:38:53.975','',0),(256,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',342,'2025-11-30 11:39:11.611','',0),(257,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',277,'2025-11-30 11:39:53.115','',0),(258,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',284,'2025-11-30 11:40:04.010','',0),(259,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',273,'2025-11-30 14:35:46.125','',0),(260,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',219,'2025-11-30 15:43:25.630','',0),(261,1,'RECOVER_YOUR_DATA','',5,'CREATE DATABASE IF NOT EXISTS test \n  DEFAULT CHARACTER SET utf8mb4;','','',1,'执行成功',762,'2025-11-30 23:49:38.276','',0),(262,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',345,'2025-11-30 23:50:09.029','',0),(263,1,'devops','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',304,'2025-11-30 23:50:24.695','',0),(264,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',311,'2025-11-30 23:50:29.549','',0),(265,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',285,'2025-11-30 23:51:51.702','',0),(266,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',308,'2025-11-30 23:53:56.382','',0),(267,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',291,'2025-11-30 23:54:23.552','',0),(268,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',292,'2025-11-30 23:54:33.359','',0),(269,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',303,'2025-11-30 23:54:38.550','',0),(270,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',333,'2025-11-30 23:54:46.965','',0),(271,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',319,'2025-11-30 23:54:58.898','',0),(272,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',314,'2025-11-30 23:54:59.768','',0),(273,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',315,'2025-11-30 23:55:02.322','',0),(274,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',290,'2025-11-30 23:55:11.683','',0),(275,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',282,'2025-11-30 23:55:36.034','',0),(276,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',261,'2025-11-30 23:55:40.374','',0),(277,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 100;','','',1,'执行成功',288,'2025-11-30 23:56:00.161','',0),(278,1,'devops','cmdb_group;',1,'SELECT * FROM cmdb_group;','','',1,'执行成功',272,'2025-11-30 23:56:57.273','',0),(279,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',279,'2025-11-30 23:57:18.128','',0),(280,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',286,'2025-11-30 23:57:32.189','',0),(281,1,'devops','cmdb_group',2,'INSERT INTO cmdb_group (parent_id, name, create_time, remark, update_time) VALUES (\'0\', \'test\', \'\', \'123\', \'\')','','',2,'Error 1292 (22007): Incorrect datetime value: \'\' for column \'create_time\' at row 1',234,'2025-11-30 23:58:10.839','',0),(282,1,'devops','cmdb_host_copy_20251130235846',1,'SELECT * FROM cmdb_host_copy_20251130235846 LIMIT 20 OFFSET 0','','',1,'执行成功',286,'2025-11-30 23:58:49.791','',0),(283,1,'devops','',5,'DROP TABLE cmdb_host_copy_20251130235846','','',1,'执行成功',295,'2025-11-30 23:58:54.686','',0),(284,1,'devops','',5,'RENAME TABLE cmdb_host_copy_20251130235914 TO cmdb_host_copy_123','','',1,'执行成功',274,'2025-11-30 23:59:22.577','',0),(285,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',207,'2025-11-30 23:59:58.611','',0),(286,2,'database_name','users',2,'INSERT INTO users (name, email) VALUES (\'李四\', \'lisi@123.com\')','','',1,'执行成功',285,'2025-12-01 00:00:30.571','',0),(287,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',214,'2025-12-01 00:00:31.076','',0),(288,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',280,'2025-12-01 00:05:29.288','',0),(289,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',295,'2025-12-01 00:06:23.704','',0),(290,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',1184,'2025-12-01 00:06:54.903','',0),(291,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',611,'2025-12-01 00:15:50.948','',0),(292,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',291,'2025-12-01 00:17:04.651','',0),(293,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',303,'2025-12-01 00:19:18.036','',0),(294,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',417,'2025-12-01 00:19:22.279','',0),(295,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',280,'2025-12-01 00:20:01.970','',0),(296,1,'devops','cmdb_host',3,'UPDATE cmdb_host SET host_name = \'虚拟机001111\', group_id = 12, private_ip = \'172.16.226.16\', public_ip = \'120.231.244.158\', ssh_name = \'root\', ssh_key_id = 13, ssh_port = 22, remark = \'123\', vendor = 1, region = \'\', instance_id = \'\', os = \'CentOSLinux7(Core)\', status = 1, cpu = \'2\', memory = \'4\', disk = \'17\', billing_type = \'\', create_time = \'2025-11-23 23:44:35\', expire_time = NULL, update_time = \'2025-11-23 23:45:32\', ssh_ip = \'172.16.226.16\', name = \'jenkins\', ssh_gateway_id = NULL WHERE id = 511;','','',1,'执行成功',254,'2025-12-01 00:20:08.106','',0),(297,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',292,'2025-12-01 00:20:08.651','',0),(298,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',301,'2025-12-01 00:20:11.764','',0),(299,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 100;','','',1,'执行成功',293,'2025-12-01 00:20:31.016','',0),(300,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',302,'2025-12-01 00:21:51.502','',0),(301,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 100;','','',1,'执行成功',569,'2025-12-01 00:21:57.257','',0),(302,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',367,'2025-12-01 00:23:39.693','',0),(303,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 100;','','',1,'执行成功',540,'2025-12-01 00:23:47.277','',0),(304,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',217,'2025-12-01 00:24:18.173','',0),(305,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',289,'2025-12-01 00:25:05.373','',0),(306,2,'database_name','users',2,'INSERT INTO users (name, email) VALUES (\'test\', \'123@123.com\')','','',1,'执行成功',553,'2025-12-01 00:30:03.796','',0),(307,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',205,'2025-12-01 00:30:04.267','',0),(308,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-01 00:30:24.140','',0),(309,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',452,'2025-12-01 00:37:20.181','',0),(310,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',235,'2025-12-01 00:37:27.599','',0),(311,2,'database_name','users_copy_20251201003823',1,'SELECT * FROM users_copy_20251201003823 LIMIT 20 OFFSET 0','','',1,'执行成功',197,'2025-12-01 00:38:47.198','',0),(312,2,'database_name','users_copy_20251201003823',1,'SELECT * FROM users_copy_20251201003823 LIMIT 100;','','',1,'执行成功',227,'2025-12-01 00:38:56.092','',0),(313,1,'RECOVER_YOUR_DATA','',5,'CREATE DATABASE IF NOT EXISTS ops \n  DEFAULT CHARACTER SET utf8mb4;','','',1,'执行成功',729,'2025-12-01 00:43:24.905','',0),(314,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',296,'2025-12-01 00:43:58.889','',0),(315,1,'devops','cmdb_host_copy_123',1,'SELECT * FROM cmdb_host_copy_123 LIMIT 20 OFFSET 0','','',1,'执行成功',351,'2025-12-01 00:44:00.240','',0),(316,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',320,'2025-12-01 00:44:02.951','',0),(317,1,'devops','cmdb_host',3,'UPDATE cmdb_host SET host_name = \'虚拟机01\', group_id = 12, private_ip = \'172.16.226.16\', public_ip = \'120.231.244.158\', ssh_name = \'root\', ssh_key_id = 13, ssh_port = 22, remark = \'123\', vendor = 1, region = \'\', instance_id = \'\', os = \'CentOSLinux7(Core)\', status = 1, cpu = \'2\', memory = \'4\', disk = \'17\', billing_type = \'\', create_time = \'2025-11-23 23:44:35\', expire_time = NULL, update_time = \'2025-11-23 23:45:32\', ssh_ip = \'172.16.226.16\', name = \'jenkins\', ssh_gateway_id = NULL WHERE id = 511;','','',1,'执行成功',722,'2025-12-01 00:44:23.250','',0),(318,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',333,'2025-12-01 00:44:26.196','',0),(319,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',297,'2025-12-01 00:44:31.613','',0),(320,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',316,'2025-12-01 00:44:46.505','',0),(321,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',475,'2025-12-01 00:46:30.077','',0),(322,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',326,'2025-12-01 00:46:48.843','',0),(323,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',290,'2025-12-01 00:46:53.212','',0),(324,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',296,'2025-12-01 00:46:57.044','',0),(325,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',306,'2025-12-01 00:47:00.548','',0),(326,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',296,'2025-12-01 00:47:09.235','',0),(327,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',327,'2025-12-01 00:47:22.518','',0),(328,1,'devops','cmdb_host;',1,'SELECT * FROM  cmdb_host;','','',1,'执行成功',297,'2025-12-01 00:48:12.389','',0),(329,1,'devops','',5,'DROP TABLE cmdb_host_copy_123','','',1,'执行成功',578,'2025-12-01 00:49:01.428','',0),(330,1,'devops','cmdb_host_copy_20251201004911',1,'SELECT * FROM cmdb_host_copy_20251201004911 LIMIT 20 OFFSET 0','','',1,'执行成功',321,'2025-12-01 00:49:15.842','',0),(331,1,'devops','',5,'RENAME TABLE cmdb_host_copy_20251201004911 TO cmdb_host_copy_123','','',1,'执行成功',253,'2025-12-01 00:49:25.590','',0),(332,1,'devops','',5,'DROP TABLE cmdb_host_copy_123','','',1,'执行成功',259,'2025-12-01 00:49:43.125','',0),(333,2,'database_name','',5,'CREATE TABLE students (\n    id SERIAL PRIMARY KEY,                -- 学生ID，自增主键\n    student_id VARCHAR(20) UNIQUE NOT NULL, -- 学号，唯一且非空\n    name VARCHAR(50) NOT NULL,            -- 姓名\n    gender CHAR(1) CHECK (gender IN (\'M\', \'F\')), -- 性别：M 男，F 女\n    birth_date DATE,                      -- 出生日期\n    email VARCHAR(100) UNIQUE,            -- 邮箱，唯一\n    phone VARCHAR(20),                    -- 电话\n    enrollment_date DATE DEFAULT CURRENT_DATE, -- 入学日期，默认为当前日期\n    major VARCHAR(100),                   -- 专业\n    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- 记录创建时间\n);','','',1,'执行成功',239,'2025-12-01 00:50:33.682','',0),(334,2,'database_name','students',1,'SELECT * FROM students LIMIT 20 OFFSET 0','','',1,'执行成功',222,'2025-12-01 00:50:38.712','',0),(335,2,'database_name','students',2,'INSERT INTO students (student_id, name, gender, birth_date, email, phone, enrollment_date, major) VALUES (\'1\', \'张三\', \'m\', \'2004-05-15\', \'123@456.com\', \'12345678911\', \'2004-05-15\', \'软件工程\')','','',2,'pq: new row for relation \"students\" violates check constraint \"students_gender_check\"',186,'2025-12-01 00:52:04.216','',0),(336,2,'database_name','students',2,'INSERT INTO students (student_id, name, gender, birth_date, email, phone, enrollment_date, major) VALUES (\'20230001\', \'张三\', \'m\', \'2004-05-15\', \'123@456.com\', \'13800138001\', \'2004-05-15\', \'软件工程\')','','',2,'pq: new row for relation \"students\" violates check constraint \"students_gender_check\"',182,'2025-12-01 00:52:31.225','',0),(337,2,'database_name','students',2,'INSERT INTO students (student_id, name, gender, birth_date, email, phone, enrollment_date, major) VALUES (\'20230001\', \'张三\', \'\'\'M\'\'\', \'2004-05-15\', \'123@456.com\', \'13800138001\', \'2004-05-15\', \'软件工程\')','','',2,'pq: value too long for type character(1)',214,'2025-12-01 00:52:54.401','',0),(338,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',542,'2025-12-01 00:52:58.529','',0),(339,2,'database_name','users',2,'INSERT INTO users (name, email) VALUES (\'test123\', \'123@123.com\')','','',1,'执行成功',183,'2025-12-01 00:53:22.992','',0),(340,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',253,'2025-12-01 00:53:23.494','',0),(341,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',207,'2025-12-01 00:53:33.443','',0),(342,2,'database_name','users',1,'SELECT * FROM users LIMIT 100;','','',1,'执行成功',196,'2025-12-01 00:53:54.378','',0),(343,2,'database_name','users_copy_20251201003823',1,'SELECT * FROM users_copy_20251201003823 LIMIT 20 OFFSET 0','','',1,'执行成功',231,'2025-12-01 00:53:59.552','',0),(344,2,'database_name','',5,'DROP TABLE users_copy_20251201003823','','',1,'执行成功',184,'2025-12-01 00:54:05.164','',0),(345,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',286,'2025-12-01 10:18:58.108','',0),(346,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 100;','','',1,'执行成功',257,'2025-12-01 10:19:34.501','',0),(347,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-12-01 12:05:45.106','',0),(348,1,'RECOVER_YOUR_DATA','RECOVER_YOUR_DATA',1,'SELECT * FROM RECOVER_YOUR_DATA LIMIT 20 OFFSET 0','','',1,'执行成功',288,'2025-12-01 14:07:15.387','',0),(349,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-01 14:07:23.511','',0),(350,1,'gin-api','app_sh_release',1,'SELECT * FROM app_sh_release LIMIT 20 OFFSET 0','','',1,'执行成功',254,'2025-12-01 17:10:51.225','',0),(351,1,'gin-api','app_service_release',1,'SELECT * FROM app_service_release LIMIT 20 OFFSET 0','','',1,'执行成功',245,'2025-12-01 17:11:07.319','',0),(352,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',253,'2025-12-01 17:11:39.058','',0),(353,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',248,'2025-12-01 17:11:57.884','',0),(354,1,'gin-api','quick_deployment_tasks',1,'SELECT * FROM quick_deployment_tasks LIMIT 20 OFFSET 0','','',1,'执行成功',280,'2025-12-01 17:12:31.887','',0),(355,1,'gin-api','quick_deployments',1,'SELECT * FROM quick_deployments LIMIT 20 OFFSET 0','','',1,'执行成功',254,'2025-12-01 17:12:34.351','',0),(356,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',248,'2025-12-01 19:50:46.392','',0),(357,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',294,'2025-12-02 11:13:02.970','',0),(358,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-12-02 11:34:17.783','',0),(359,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-02 11:35:41.192','',0),(360,1,'gin-api','sys_operation_log',1,'SELECT * FROM sys_operation_log LIMIT 20 OFFSET 0','','',1,'执行成功',292,'2025-12-02 12:01:05.583','',0),(361,1,'gin-api','sys_operation_log',1,'SELECT * FROM sys_operation_log LIMIT 100;','','',1,'执行成功',555,'2025-12-02 12:06:45.939','',0),(362,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',257,'2025-12-02 12:08:01.959','',0),(363,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',250,'2025-12-02 12:25:37.979','',0),(364,1,'devops','app_application',1,'SELECT *\nFROM  app_application\nLIMIT  100;','','',1,'执行成功',252,'2025-12-02 12:25:58.932','',0),(365,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',277,'2025-12-02 12:30:05.820','',0),(366,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',253,'2025-12-02 12:35:09.666','',0),(367,2,'database_name','students',1,'SELECT * FROM students LIMIT 20 OFFSET 0','','',1,'执行成功',180,'2025-12-02 12:35:11.544','',0),(368,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',262,'2025-12-02 12:38:20.200','',0),(369,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',251,'2025-12-02 12:38:39.691','',0),(370,1,'gin-api','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',250,'2025-12-02 14:59:49.691','',0),(371,1,'gin-api','db',1,'SELECT * FROM db LIMIT 20 OFFSET 0','','',1,'执行成功',261,'2025-12-02 15:00:14.754','',0),(372,1,'gin-api','db_instance',1,'SELECT * FROM db_instance LIMIT 20 OFFSET 0','','',1,'执行成功',258,'2025-12-02 15:01:14.757','',0),(373,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',255,'2025-12-02 15:09:10.006','',0),(374,1,'gin-api','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',247,'2025-12-02 15:41:50.960','',0),(375,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',251,'2025-12-02 15:42:08.133','',0),(376,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',252,'2025-12-02 15:42:47.779','',0),(377,1,'RECOVER_YOUR_DATA','RECOVER_YOUR_DATA',1,'SELECT * FROM RECOVER_YOUR_DATA LIMIT 20 OFFSET 0','','',1,'执行成功',284,'2025-12-02 16:00:08.047','',0),(378,1,'mayfly-go','t_db_backup_history',1,'SELECT * FROM t_db_backup_history LIMIT 20 OFFSET 0','','',1,'执行成功',284,'2025-12-02 16:00:14.480','',0),(379,1,'mayfly-go','t_db_restore',1,'SELECT * FROM t_db_restore LIMIT 20 OFFSET 0','','',1,'执行成功',283,'2025-12-02 16:00:16.976','',0),(380,3,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-02 16:59:07.917','',0),(381,3,'gin-api','app_service_release_item',1,'SELECT * FROM app_service_release_item LIMIT 20 OFFSET 0','','',1,'执行成功',261,'2025-12-02 16:59:08.689','',0),(382,1,'gin-api','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-12-03 10:26:41.230','',0),(383,1,'gin-api','app_service_release_item',1,'SELECT * FROM app_service_release_item LIMIT 20 OFFSET 0','','',1,'执行成功',323,'2025-12-03 10:27:20.235','',0),(384,1,'gin-api','app_sh_release',1,'SELECT * FROM app_sh_release LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-03 10:27:21.025','',0),(385,1,'gin-api','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',548,'2025-12-03 10:31:37.487','',0),(386,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',263,'2025-12-03 10:36:57.065','',0),(387,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',340,'2025-12-03 10:37:02.051','',0),(388,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',266,'2025-12-03 10:40:18.048','',0),(389,1,'devops','cmdb_host_1',2,'INSERT INTO cmdb_host_1 (id, host_name, group_id, private_ip, public_ip, ssh_name, ssh_key_id, ssh_port, remark, vendor, region, instance_id, os, status, cpu, memory, disk, billing_type, create_time, expire_time, update_time, ssh_ip, name, ssh_gateway_id) VALUES (506, \'华为云ops\', 4, \'172.31.6.35\', \'139.9.205.38\', \'root\', 22, 22, \'123\', 5, \'\', \'\', \'Ubuntu24.04.2\', 1, \'2\', \'2\', \'40\', \'\', \'2025-11-11 17:24:36\', NULL, \'2025-11-26 17:22:07\', \'139.9.205.38\', \'hw-ops\', NULL);','','',2,'Error 1146 (42S02): Table \'devops.cmdb_host_1\' doesn\'t exist',202,'2025-12-03 10:43:12.842','',0),(390,1,'devops','cmdb_host_1',2,'INSERT INTO cmdb_host_1 (id, host_name, group_id, private_ip, public_ip, ssh_name, ssh_key_id, ssh_port, remark, vendor, region, instance_id, os, status, cpu, memory, disk, billing_type, create_time, expire_time, update_time, ssh_ip, name, ssh_gateway_id) VALUES (506, \'华为云ops\', 4, \'172.31.6.35\', \'139.9.205.38\', \'root\', 22, 22, \'123\', 5, \'\', \'\', \'Ubuntu24.04.2\', 1, \'2\', \'2\', \'40\', \'\', \'2025-11-11 17:24:36\', NULL, \'2025-11-26 17:22:07\', \'139.9.205.38\', \'hw-ops\', NULL);','','',2,'Error 1146 (42S02): Table \'devops.cmdb_host_1\' doesn\'t exist',216,'2025-12-03 10:43:14.148','',0),(391,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',785,'2025-12-03 11:09:19.304','',0),(392,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-12-03 11:24:14.607','',0),(393,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',275,'2025-12-03 11:35:15.655','',0),(394,1,'mayfly-go','t_db',1,'SELECT * FROM t_db LIMIT 20 OFFSET 0','','',1,'执行成功',266,'2025-12-03 11:40:08.029','',0),(395,1,'mayfly-go','t_db_backup',1,'SELECT * FROM t_db_backup LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-12-03 11:40:10.424','',0),(396,1,'mayfly-go','t_db_data_sync_log',1,'SELECT * FROM t_db_data_sync_log LIMIT 20 OFFSET 0','','',1,'执行成功',416,'2025-12-03 11:41:35.563','',0),(397,1,'mayfly-go','t_db_instance',1,'SELECT * FROM t_db_instance LIMIT 20 OFFSET 0','','',1,'执行成功',259,'2025-12-03 11:41:38.235','',0),(398,1,'gin-api','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-03 11:41:41.904','',0),(399,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',257,'2025-12-03 11:41:59.928','',0),(400,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',269,'2025-12-03 11:43:46.442','',0),(401,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',317,'2025-12-03 11:45:13.925','',0),(402,1,'devops','',5,'CREATE TABLE `app_jenkins_env123` ( `id` bigint(20) NOT NULL AUTO_INCREMENT, `app_id` bigint(20) NOT NULL, `env_name` varchar(50) NOT NULL, `jenkins_server_id` bigint(20), `job_name` varchar(255), `job_url` varchar(500), `build_params` json, `deploy_config` json, `notification` json, `is_active` tinyint(3) DEFAULT 1, `created_at` datetime, `updated_at` datetime, `deleted_at` datetime, PRIMARY KEY (id) ); ALTER TABLE `app_jenkins_env` ADD INDEX `idx_app_jenkins_env_app_id`(`app_id`) USING BTREE; ALTER TABLE `app_jenkins_env` ADD INDEX `idx_app_jenkins_env_deleted_at`(`deleted_at`) USING BTREE','','',2,'Error 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near \'ALTER TABLE `app_jenkins_env` ADD INDEX `idx_app_jenkins_env_app_id`(`app_id`) U\' at line 1',204,'2025-12-03 11:47:45.166','',0),(403,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',270,'2025-12-03 11:48:51.742','',0),(404,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-12-03 11:52:55.287','',0),(405,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',257,'2025-12-03 11:52:58.121','',0),(406,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',260,'2025-12-03 11:53:00.314','',0),(407,1,'gin-api','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',295,'2025-12-03 11:56:02.107','',0),(408,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-03 12:04:05.208','',0),(409,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',254,'2025-12-03 12:07:28.099','',0),(410,1,'gin-api','app_sh_release',1,'SELECT * FROM app_sh_release LIMIT 20 OFFSET 0','','',1,'执行成功',261,'2025-12-03 12:08:37.417','',0),(411,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',327,'2025-12-03 12:08:52.889','',0),(412,1,'devops','cmdb_group',1,'SELECT * FROM cmdb_group LIMIT 20 OFFSET 0','','',1,'执行成功',258,'2025-12-03 12:14:02.575','',0),(413,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',342,'2025-12-03 12:16:54.823','',0),(414,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',261,'2025-12-03 12:19:56.646','',0),(415,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',257,'2025-12-03 12:23:13.577','',0),(416,1,'devops','cmdb_sql',1,'SELECT * FROM cmdb_sql LIMIT 20 OFFSET 0','','',1,'执行成功',261,'2025-12-03 12:26:28.689','',0),(417,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',253,'2025-12-03 12:28:20.840','',0),(418,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',244,'2025-12-03 12:29:04.357','',0),(419,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',395,'2025-12-03 12:32:00.866','',0),(420,1,'devops','',5,'CREATE TABLE `cmdb_group123` (\n `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT \'\'\'主键\'\'\',\n `parent_id` bigint(20) NOT NULL DEFAULT 0 COMMENT \'\'\'父级分组ID\'\'\',\n `name` longtext NOT NULL COMMENT \'\'\'分组名称\'\'\',\n `create_time` datetime NOT NULL COMMENT \'\'\'创建时间\'\'\',\n `remark` longtext COMMENT \'\'\'备注\'\'\',\n `update_time` datetime COMMENT \'\'\'更新时间\'\'\', \nPRIMARY KEY (id)\n)','','',1,'执行成功',244,'2025-12-03 12:42:43.341','',0),(421,1,'devops','cmdb_group123',1,'SELECT * FROM cmdb_group123 LIMIT 20 OFFSET 0','','',1,'执行成功',262,'2025-12-03 12:42:51.081','',0),(422,1,'devops','',5,'DROP TABLE cmdb_group123','','',1,'执行成功',228,'2025-12-03 12:43:02.777','',0),(423,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',265,'2025-12-03 12:46:17.553','',0),(424,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',278,'2025-12-03 15:54:52.850','',0),(425,1,'devops','cmdb_sql_log',1,'SELECT * FROM cmdb_sql_log LIMIT 20 OFFSET 0','','',1,'执行成功',252,'2025-12-03 15:55:18.939','',0),(426,1,'gin-api','sys_operation_log',1,'SELECT * FROM sys_operation_log LIMIT 20 OFFSET 0','','',1,'执行成功',254,'2025-12-03 15:55:30.381','',0),(427,1,'gin-api','sys_operation_log',1,'SELECT *\nFROM  sys_operation_log\nLIMIT  100;','','',1,'执行成功',266,'2025-12-03 15:55:49.290','',0),(428,1,'gin-api','sys_admin',1,'SELECT *\nFROM  sys_admin\nLIMIT  100;','','',1,'执行成功',250,'2025-12-03 15:57:53.622','',0),(429,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',256,'2025-12-05 20:29:40.131','',0),(430,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',262,'2025-12-05 20:32:28.584','',0),(431,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',392,'2025-12-05 20:33:31.694','',0),(432,1,'devops','app_application',1,'SELECT *\nFROM  app_application\nLIMIT  100;','','',1,'执行成功',308,'2025-12-05 20:33:51.765','',0),(433,1,'devops','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',424,'2025-12-05 20:34:23.519','',0),(434,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',273,'2025-12-05 20:34:30.817','',0),(435,1,'devops','cmdb_host',3,'UPDATE cmdb_host SET host_name = \'虚拟机2025\', group_id = 4, private_ip = \'172.16.226.13\', public_ip = \'120.231.244.158\', ssh_name = \'root\', ssh_key_id = 13, ssh_port = 22, remark = \'123\', vendor = 1, region = \'\', instance_id = \'\', os = \'Ubuntu20.04.2\', status = 1, cpu = \'2\', memory = \'3\', disk = \'19\', billing_type = \'\', create_time = \'2025-11-23 23:46:03\', expire_time = NULL, update_time = \'2025-11-23 23:46:07\', ssh_ip = \'172.16.226.13\', name = \'k8s-node02\', ssh_gateway_id = NULL WHERE id = 512;','','',1,'执行成功',436,'2025-12-05 20:34:45.595','',0),(436,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-12-05 20:34:46.094','',0),(437,1,'devops','cmdb_host',1,'SELECT * FROM cmdb_host LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-12-05 20:34:49.893','',0),(438,1,'devops','',5,'CREATE DATABASE IF NOT EXISTS test1\n  DEFAULT CHARACTER SET utf8mb4;','','',1,'执行成功',237,'2025-12-05 20:35:34.369','',0),(439,1,'test1','',5,'CREATE TABLE test1.your_table_name (\n  id BIGINT PRIMARY KEY AUTO_INCREMENT,\n  -- your columns here\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;','','',2,'Error 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near \') ENGINE=InnoDB DEFAULT CHARSET=utf8mb4\' at line 4',219,'2025-12-05 20:35:52.483','',0),(440,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',217,'2025-12-05 20:36:57.655','',0),(441,2,'database_name','users',2,'INSERT INTO users (name, email) VALUES (\'王五\', \'xxxx@123.com\')','','',1,'执行成功',191,'2025-12-05 20:37:14.770','',0),(442,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',205,'2025-12-05 20:37:15.207','',0),(443,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-12-08 10:52:11.033','',0),(444,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',288,'2025-12-08 23:01:38.354','',0),(445,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-12-08 23:01:38.638','',0),(446,1,'gin-api','sys_config',1,'SELECT * FROM sys_config LIMIT 20 OFFSET 0','','',1,'执行成功',259,'2025-12-09 13:21:51.379','',0),(447,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',280,'2025-12-09 23:45:28.566','',0),(448,1,'gin-api','app_jenkins_env',1,'SELECT * FROM app_jenkins_env LIMIT 20 OFFSET 0','','',1,'执行成功',268,'2025-12-09 23:48:06.048','',0),(449,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',267,'2025-12-10 11:22:14.627','',0),(450,1,'gin-api','sys_admin',1,'SELECT * FROM sys_admin LIMIT 20 OFFSET 0','','',1,'执行成功',268,'2025-12-10 17:17:50.042','',0),(451,1,'gin-api','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',276,'2025-12-11 18:04:01.397','',0),(452,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',314,'2025-12-13 00:03:02.341','',0),(453,2,'database_name','users',1,'SELECT * FROM users LIMIT 20 OFFSET 0','','',1,'执行成功',287,'2025-12-13 00:03:27.273','',0),(454,2,'database_name','students',1,'SELECT * FROM students LIMIT 20 OFFSET 0','','',1,'执行成功',257,'2025-12-13 00:03:29.263','',0),(455,1,'devops','app_application',1,'SELECT * FROM app_application LIMIT 20 OFFSET 0','','',1,'执行成功',303,'2025-12-13 16:15:43.928','',0),(456,5,'prometheusalert','alert_record',1,'SELECT * FROM alert_record LIMIT 20 OFFSET 0','','',1,'执行成功',9,'2026-01-19 14:53:49.871','',0),(457,5,'prometheusalert','prometheus_alert_d_b',1,'SELECT * FROM prometheus_alert_d_b LIMIT 20 OFFSET 0','','',1,'执行成功',8,'2026-01-19 14:54:30.837','',0),(458,5,'prometheusalert','prometheus_alert_d_b',1,'SELECT * FROM prometheus_alert_d_b LIMIT 20 OFFSET 0','','',1,'执行成功',9,'2026-01-19 14:54:33.234','',0),(459,5,'prometheusalert','prometheus_alert_d_b',1,'SELECT * FROM prometheus_alert_d_b LIMIT 100;','','',1,'执行成功',10,'2026-01-19 14:54:55.874','',0),(460,5,'prometheusalert','alert_router',1,'SELECT * FROM alert_router LIMIT 20 OFFSET 0','','',1,'执行成功',7,'2026-01-19 14:56:14.050','',0),(461,5,'kafori','exp_class',1,'SELECT * FROM exp_class LIMIT 20 OFFSET 0','','',1,'执行成功',112,'2026-01-20 17:50:24.858','',0),(462,5,'kafori','experiment',1,'SELECT * FROM experiment LIMIT 20 OFFSET 0','','',1,'执行成功',38,'2026-01-20 17:50:29.117','',0),(463,5,'kafori','gene_express_counts',1,'SELECT * FROM gene_express_counts LIMIT 20 OFFSET 0','','',1,'执行成功',77,'2026-01-20 17:50:29.725','',0),(464,5,'kafori','gene_express_tpm',1,'SELECT * FROM gene_express_tpm LIMIT 20 OFFSET 0','','',1,'执行成功',75,'2026-01-20 17:50:30.854','',0),(465,5,'kafori','sample',1,'SELECT * FROM sample LIMIT 20 OFFSET 0','','',1,'执行成功',80,'2026-01-20 17:50:31.335','',0),(466,5,'kafori','user',1,'SELECT * FROM user LIMIT 20 OFFSET 0','','',1,'执行成功',62,'2026-01-20 17:50:31.939','',0);
/*!40000 ALTER TABLE `db_sql_exec` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `k8s_cluster`
--

DROP TABLE IF EXISTS `k8s_cluster`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_cluster` (
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
) ENGINE=InnoDB AUTO_INCREMENT=37 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_cluster`
--

LOCK TABLES `k8s_cluster` WRITE;
/*!40000 ALTER TABLE `k8s_cluster` DISABLE KEYS */;
INSERT INTO `k8s_cluster` VALUES (36,'k8s','v1.34.3',2,'apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURtakNDQW9LZ0F3SUJBZ0lVWUFjVzMvSjJLTFhVbnBDYkRUT3Q0REpFSjljd0RRWUpLb1pJaHZjTkFRRUwKQlFBd1pERUxNQWtHQTFVRUJoTUNRMDR4RVRBUEJnTlZCQWdUQ0VoaGJtZGFhRzkxTVFzd0NRWURWUVFIRXdKWQpVekVNTUFvR0ExVUVDaE1EYXpoek1ROHdEUVlEVlFRTEV3WlRlWE4wWlcweEZqQVVCZ05WQkFNVERXdDFZbVZ5CmJtVjBaWE10WTJFd0lCY05Nall3TXpBek1ERTBNREF3V2hnUE1qRXlOakF5TURjd01UUXdNREJhTUdReEN6QUoKQmdOVkJBWVRBa05PTVJFd0R3WURWUVFJRXdoSVlXNW5XbWh2ZFRFTE1Ba0dBMVVFQnhNQ1dGTXhEREFLQmdOVgpCQW9UQTJzNGN6RVBNQTBHQTFVRUN4TUdVM2x6ZEdWdE1SWXdGQVlEVlFRREV3MXJkV0psY201bGRHVnpMV05oCk1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBc1psZlAzOWtyUW1OblN3RUVOd28KdEFqTTVNQ242dFp1VUtGOEVzWnhzWk5QSlNZTDY4aUIrY0xqTWtLdmFjNDl1V2JMdy9ab24vU2VtZTVVdGZ0RQp6QmhUcTltZ2hqenJFb0duREh6QjNNYTR6UGpEeFFMUngzVVduWVBRZWpPMVBQYlM4cGM0M3RvSTkzL0w2Q0RGClBpR3pBVlIrVDNWc3VIbkF1MG1NVjJkRTRuSEdPOXBsSjd5YXZEV0pLMzV5RUdBTndIY29FTElDaEhMVXYyOVcKQStjMlphcDhyUXltdTBPS01qV3ZpSTJPWHV6bjRWNDV1VzdHYlRMNVIzNnFtMnBaeEVqVDhTVzdvLzVVc1BEUApiVEtxYlZacFRLanNKeGJjTEdtZnRnTmxBOE9VWmp5UjRLbDdJang2S3g2RG81eHJTZEtwd1BFVXAvZUpYeUV5Cm9RSURBUUFCbzBJd1FEQU9CZ05WSFE4QkFmOEVCQU1DQVFZd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlYKSFE0RUZnUVVjNTB2WENqOVJFNUhYWnNoYUtya0s1S2Q4ZmN3RFFZSktvWklodmNOQVFFTEJRQURnZ0VCQUdDOQoxd2dSYzZXTWQ0WjBJUHNMVHRoOHk4UXMvWkpuWGNlMkhlSkRmY0g3SmpLL056R0xJWmVyZXhVdndxRHQzTFFzCjlHMkJlZTlYZS8rNkVDVXNOM0xGQWlhTEZ3K2ZtNGxOM1E5R2J2Um9wRHVleGhxUVhtZFNGR3BEQlR1bHYyTlkKV0NaUk1GWTFrYkJldXlXOHdQQ05rY0pmd05mV0NYL25YNnh2MFBHdzNTRHplVFBNQ1pnTmRvekdWZVNDeHg1dQpLSFQ2ZTVWb1dMSW1IdkM5bTluejBFM1VBWHl6WUdUTFZtelNoem1pR2NJOC92aDVoNDFtTUNCK3BhZmNJVmh2ClRFVDc3bFpLTkozSUhsZEJ1ZGFaT1d2Um0zbkdOMStGdTVSWVZtR3NPOThFeDhnT0VSTDJQcjJwbVB6Q1RVbXQKbEs1eUpEUjd0dmFMUHhDa2Jvdz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=\n    server: https://192.168.0.51:6443\n  name: cluster1\ncontexts:\n- context:\n    cluster: cluster1\n    user: admin\n  name: context-cluster1\ncurrent-context: context-cluster1\nkind: Config\nusers:\n- name: admin\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUQyakNDQXNLZ0F3SUJBZ0lVUE1JRVJKMmhYY01uR1ppNTcwQ1Q3d1FBQmxnd0RRWUpLb1pJaHZjTkFRRUwKQlFBd1pERUxNQWtHQTFVRUJoTUNRMDR4RVRBUEJnTlZCQWdUQ0VoaGJtZGFhRzkxTVFzd0NRWURWUVFIRXdKWQpVekVNTUFvR0ExVUVDaE1EYXpoek1ROHdEUVlEVlFRTEV3WlRlWE4wWlcweEZqQVVCZ05WQkFNVERXdDFZbVZ5CmJtVjBaWE10WTJFd0lCY05Nall3TkRBMk1UUXhOakF3V2hnUE1qQTNOakF6TWpReE5ERTJNREJhTUdjeEN6QUoKQmdOVkJBWVRBa05PTVJFd0R3WURWUVFJRXdoSVlXNW5XbWh2ZFRFTE1Ba0dBMVVFQnhNQ1dGTXhGekFWQmdOVgpCQW9URG5ONWMzUmxiVHB0WVhOMFpYSnpNUTh3RFFZRFZRUUxFd1pUZVhOMFpXMHhEakFNQmdOVkJBTVRCV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXpuMDRpT3FsbEY2c01IeVYKNXBMT0lpN25NbkZOeHhhdi9QN2V6SldlQXhCTkY5aG40eE5PVC9Hc2o4eC8zSVprZEoySncwempkVlc3TzBnMgorUitTNTNlVkRHWkxLY0lDK2NFNHJheUp2dXJvSUdrY3RmK2VWTm1SVWgzWTFpM05VeUxIVXZGT0JkY0pTV0ExCjMwckt3LzBuYVE3VjM4dmIvMnVqSTFteFZWNU9HVkh6Z2pITkFCZVd0SHpncnQ0US90QmtwM1FZMXowVzU3dC8KSXBURnp3TGZJeVN0NHk0Q0RxWWdvS0F0VnhveW5SWVBlQUZVanJURkR4aUpqMC8wN3BCL2YxbEQ4WitjbGxNUQpoRlhsZ3RNTDd4TVlLVnBmbFdqZ1UwMGJodEZ6dHFUSmxScWM0RElpK2dOeU5HdXBTR2ZweEVBVkNzVGZzR0tnCmlNREdOd0lEQVFBQm8zOHdmVEFPQmdOVkhROEJBZjhFQkFNQ0JhQXdIUVlEVlIwbEJCWXdGQVlJS3dZQkJRVUgKQXdFR0NDc0dBUVVGQndNQ01Bd0dBMVVkRXdFQi93UUNNQUF3SFFZRFZSME9CQllFRkFrdnpWVWUwSm8vbFJYawoyM1lxU3YrQnhaVjhNQjhHQTFVZEl3UVlNQmFBRkhPZEwxd28vVVJPUjEyYklXaXE1Q3VTbmZIM01BMEdDU3FHClNJYjNEUUVCQ3dVQUE0SUJBUUJlTXRwQVBQUlhBWlpnb3lMSUxFNXFDVXA0REtFVTdTNzRaY1hRS2FkaVhnVGYKZlRXN29VbG1YUXplVW53emltTEdkRy9NR3IzL096Nml3RW1JWVJIbDIyS3JqbzNRaHJaRWlQVFZySGZCOHFiawpLY25UMjNkRklwVjdxNkZWZ3pTT1VyQWoyQlhUYTd4SkplMUZhdDVxL3d4dmNEb0NEdHBVaG9venBQMlUzaDAyCnAxQndEMmkybjlQT2VFRHhLcHBOcS9id0psdHh1TWlDZFQzckJaTDE2WnRqL21SVDZVTnNkM0lWai9GY1ZyMVcKSXNYZ1BJVTNrSFlvZVVjTFZNd0diWXZJWmMzcFZtd3Nna0ZZZXZVdno3R1ArTDVMVXRVWXpUanUzeUVoWURpNQp3N2dvVXpJT0tiRUxjZFFUYlVnMVNTVnZ6aStRc0xHS2lScEozNVBWCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBem4wNGlPcWxsRjZzTUh5VjVwTE9JaTduTW5GTnh4YXYvUDdlekpXZUF4Qk5GOWhuCjR4Tk9UL0dzajh4LzNJWmtkSjJKdzB6amRWVzdPMGcyK1IrUzUzZVZER1pMS2NJQytjRTRyYXlKdnVyb0lHa2MKdGYrZVZObVJVaDNZMWkzTlV5TEhVdkZPQmRjSlNXQTEzMHJLdy8wbmFRN1YzOHZiLzJ1akkxbXhWVjVPR1ZIegpnakhOQUJlV3RIemdydDRRL3RCa3AzUVkxejBXNTd0L0lwVEZ6d0xmSXlTdDR5NENEcVlnb0tBdFZ4b3luUllQCmVBRlVqclRGRHhpSmowLzA3cEIvZjFsRDhaK2NsbE1RaEZYbGd0TUw3eE1ZS1ZwZmxXamdVMDBiaHRGenRxVEoKbFJxYzRESWkrZ055Tkd1cFNHZnB4RUFWQ3NUZnNHS2dpTURHTndJREFRQUJBb0lCQVFEQVVzK0NieE1UODIzcgpwMDZ6T0k4NE5YbHZlU3JaUVI5ZnhEL2VTNGltRDl2T1Z1akVEbXBGZWJLaEtQYjZmazQ0YzdjcHFZYTk5Y0R6CkFWcjZoRmIrc2xtbjkrc1FGTDFTeEk0MjdOcExjVjROT2ZuYzVoNGhwNWQ3NVlMZVVrRElxejc5MlBVejZVcWsKQmpHSExaRllKSVU5aSt6V2luajVHTmNWQ09ocnVrTVRrNUtXeFZSaUo5NWJBWThEdE11M2xpWE9ibnp0UVhmSwpwVnFhd1IrbDdDOUlHZVAxaXBxUHYxdkVndUpJZkhsUTZBVXRHRWhoWnk2SGRIb2dGREd5UjNzUmxHMDNYcDlQCnpud3ZSL01obFNUbmlIZlZLOHhVUFRPbUIwNVRvUlZ2N2RkWlNhM1Y1RnlwaCtqaTNXUWZKUFZ1OHNOMFFuY2YKMzBoZi9mdFJBb0dCQU92VDJuejdTam11cFU0UFVaUEtYZ0xQT2JjaXdSUkJ2eUZBRmNDZU4wb2tnWUpZUjluYgo4L0xCQlZ4SmJqMDZwOGhYdEhYdnV4SWJqRXF3Zkw3UVE1Tlh0QkpkeDQ4SWRsUUwwaHdlbjJkbE9LZitNeHdoCjBxZWlIM0FzU3RJeGN1SzFEN1BPTWpqc2RqTnpqOUJFTjNmTlhYd1REb1BMVEJqT3VVdE1zdXVaQW9HQkFPQW0KNm1FaDV6YlIwUWUvMERpbTFYaUpseDNVSVd6Ui9kbzU3UXN2Y2FpQzRMb05FUnZ5bVV4MklQd25VcmozMS9SYgpuSmxseTJGanV3ZXZSQUxidmVCaHJuNURVZEppNmtWMDZ5Y1lITTRydWh4c2RJRjlRanh2Z1B1TVlERE51V3N0CldEV2JzSUtuTzNsZDNGY0pRTDNTdU9uUzNJZTEvZWIrSnpNR0ZlSlBBb0dBSFlSLzVZTGlrSU8rcHgyZHZWem8KRlh2d29tNlVNai9rZDNuNC93b09xNVlVSzhkMi90cmNGdmQzLzB3bG96NVRQKzFTWml5aWdxcTJEYWMzaGY2MQpacHprcWlQWTFadGVqRGlLalFCOFBVbmJKSlBadTl5ejdFMkxsOVBEYmJuSXduMmRRSWsvbWdabTZSdSsxWGVrClpiZHJ5eHJyaHkyTUVkTFdPSW1hSDRrQ2dZQTZHSndDMU9DeWlxc2MwUkJNdnFEeGZ1ZlY2VmRJZHh6T1pOa1YKWHhTY1VsK2dtU1pvRWZhOXBKaGtBbVVrWDhodkl4ZHhncnExNFQxbWZueW9LUUFMbWdXNTRBVGkvSlF2c0dBSwpYK3VIWERuK2gvV2lZaTY2cjBQRTd0czNpdnFWMXNqWURDUXhtTnFIV0ZaMWtJWmhMSzZVbS94Z2lFRG9qOUxsCkhKeXJiUUtCZ1FEa0c2UFRJODdrUXI3TUkrZW1NTGUwTWU4ZmI2c0o4L053b2NNZ0lFT1Y3U2dYUzRQQmtKTFQKMTRJZ0JNa01mUEZrQStoWXZwcTdrdmdFZmVvZHpHa242UEJrUitXb1M3aW1FTmI4aDNsck52MjRWSmpXR0lRTQpMZXhWcndVWTVJRnRDbkFqWGpxdkl2eWZvcm1LVmMxL0h5MUtOWG5pM3lGUmpONGpjUzlZSlE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=\n','',2,'2026-04-07 10:28:35.059','2026-06-26 02:08:10.526',3,3,0,3,'2026-06-26 02:08:10.526');
/*!40000 ALTER TABLE `k8s_cluster` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `k8s_group_permission`
--

DROP TABLE IF EXISTS `k8s_group_permission`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_group_permission` (
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
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s用户组权限表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_group_permission`
--

LOCK TABLES `k8s_group_permission` WRITE;
/*!40000 ALTER TABLE `k8s_group_permission` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_group_permission` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `k8s_permission`
--

DROP TABLE IF EXISTS `k8s_permission`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_permission` (
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
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s权限管理表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_permission`
--

LOCK TABLES `k8s_permission` WRITE;
/*!40000 ALTER TABLE `k8s_permission` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_permission` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `k8s_rbac_binding`
--

DROP TABLE IF EXISTS `k8s_rbac_binding`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_rbac_binding` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL,
  `namespace` varchar(255) DEFAULT '',
  `role_id` bigint unsigned NOT NULL,
  `subject_type` varchar(32) NOT NULL,
  `subject_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=23 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_rbac_binding`
--

LOCK TABLES `k8s_rbac_binding` WRITE;
/*!40000 ALTER TABLE `k8s_rbac_binding` DISABLE KEYS */;
INSERT INTO `k8s_rbac_binding` VALUES (22,36,'monitor',15,'User',106,'2026-05-31 23:48:38.625','2026-05-31 23:55:27.402');
/*!40000 ALTER TABLE `k8s_rbac_binding` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `k8s_rbac_role`
--

DROP TABLE IF EXISTS `k8s_rbac_role`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_rbac_role` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL,
  `namespace` varchar(255) DEFAULT '',
  `name` varchar(255) NOT NULL,
  `rules` json NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_rbac_role`
--

LOCK TABLES `k8s_rbac_role` WRITE;
/*!40000 ALTER TABLE `k8s_rbac_role` DISABLE KEYS */;
INSERT INTO `k8s_rbac_role` VALUES (15,36,'monitor','test','[{\"verbs\": [\"get\", \"list\", \"watch\"], \"apiGroups\": [\"\", \"rbac.authorization.k8s.io\"], \"resources\": [\"namespaces\", \"deployments\", \"pods\", \"configmaps\", \"secrets\", \"services\", \"ingresses\", \"*\"]}]','2026-05-31 23:47:30.193','2026-05-31 23:56:12.347');
/*!40000 ALTER TABLE `k8s_rbac_role` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `k8s_user_group`
--

DROP TABLE IF EXISTS `k8s_user_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_user_group` (
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
) ENGINE=InnoDB AUTO_INCREMENT=22 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s用户组表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_user_group`
--

LOCK TABLES `k8s_user_group` WRITE;
/*!40000 ALTER TABLE `k8s_user_group` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_user_group` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `k8s_user_group_member`
--

DROP TABLE IF EXISTS `k8s_user_group_member`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_user_group_member` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `group_id` int unsigned NOT NULL COMMENT '用户组ID(k8s_user_group.id)',
  `user_id` int unsigned NOT NULL COMMENT '用户ID(sys_admin.id)',
  `created_at` datetime(3) NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_group_user` (`group_id`,`user_id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='K8s用户组成员关系表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_user_group_member`
--

LOCK TABLES `k8s_user_group_member` WRITE;
/*!40000 ALTER TABLE `k8s_user_group_member` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_user_group_member` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_agent`
--

DROP TABLE IF EXISTS `monitor_agent`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_agent` (
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
) ENGINE=InnoDB AUTO_INCREMENT=87 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_agent`
--

LOCK TABLES `monitor_agent` WRITE;
/*!40000 ALTER TABLE `monitor_agent` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_agent` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_config`
--

DROP TABLE IF EXISTS `monitor_alert_config`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_config` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `conf_key` varchar(255) DEFAULT NULL,
  `conf_value` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_monitor_alert_config_conf_key` (`conf_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_config`
--

LOCK TABLES `monitor_alert_config` WRITE;
/*!40000 ALTER TABLE `monitor_alert_config` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_alert_config` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_group_rule`
--

DROP TABLE IF EXISTS `monitor_alert_group_rule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_group_rule` (
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
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_group_rule`
--

LOCK TABLES `monitor_alert_group_rule` WRITE;
/*!40000 ALTER TABLE `monitor_alert_group_rule` DISABLE KEYS */;
INSERT INTO `monitor_alert_group_rule` VALUES (8,'2026-04-08 15:33:08.729','2026-04-08 20:02:58.403',NULL,1,'node-system-usage','apiVersion: monitoring.coreos.com/v1\nkind: PrometheusRule\nmetadata:\n    name: node-cpu-usage\n    namespace: monitor\n    labels:\n        release: prometheus\nspec:\n    groups:\n        - name: node.cpu.usage.rules\n          rules: []\n','{\"cluster\": \"beijing-core\"}'),(9,'2026-04-08 17:58:52.094','2026-05-10 10:50:36.347',NULL,1,'node-cpu-usage','apiVersion: monitoring.coreos.com/v1\nkind: PrometheusRule\nmetadata:\n    name: node-cpu-usage\n    namespace: monitor\n    labels:\n        release: prometheus\nspec:\n    groups:\n        - name: node.cpu.usage.rules\n          rules:\n            - alert: HighCPUUsage\n              expr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{instance=\"192.168.0.51:9100\",mode=\"idle\"}[5m])) * 100) > 90\n              for: 0m\n              labels:\n                cluster: default\n                severity: warning\n              annotations:\n                description: CPU 使用率持续 > 90%。\n                summary: CPU 使用率高\n            - alert: HighCPUUsageCritical\n              expr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 95\n              for: 3m\n              labels:\n                cluster: default\n                severity: critical\n              annotations:\n                description: CPU 使用率持续 > 95%。\n                summary: CPU 使用率高\n            - alert: HighMemoryUsage\n              expr: (1 - (node_memory_MemAvailable_bytes{instance=\"192.168.0.51:9100\"} / node_memory_MemTotal_bytes{instance=\"192.168.0.51:9100\"})) * 100 > 90\n              for: 0m\n              labels:\n                cluster: default\n                severity: warning\n                team: devops\n              annotations:\n                description: 内存 使用率持续 > 90%。\n                summary: 内存告警\n','{\"cluster\": \"default\"}'),(10,'2026-04-08 18:34:54.237','2026-06-27 00:00:27.658',NULL,1,'test','apiVersion: \"\"\nkind: \"\"\nmetadata:\n    name: \"\"\n    namespace: \"\"\nspec:\n    groups:\n        - name: test\n          rules:\n            - alert: domain_test\n              expr: remaining_days{env=\"prod\"} <= 40\n              for: 0m\n              labels:\n                cluster: test\n                severity: warning\n                team: devops\n','{\"cluster\":\"test\"}'),(13,'2026-04-08 21:18:26.620','2026-04-08 21:18:26.620',NULL,1,'ddd','apiVersion: monitoring.coreos.com/v1\nkind: PrometheusRule\nmetadata:\n  name: default-rules\nspec:\n  groups:\n    - name: default.rules\n      rules: []','{\"test\": \"ddd\"}');
/*!40000 ALTER TABLE `monitor_alert_group_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_record`
--

DROP TABLE IF EXISTS `monitor_alert_record`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_record` (
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
) ENGINE=InnoDB AUTO_INCREMENT=102 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_record`
--

LOCK TABLES `monitor_alert_record` WRITE;
/*!40000 ALTER TABLE `monitor_alert_record` DISABLE KEYS */;
INSERT INTO `monitor_alert_record` VALUES (14,'CPU负载过高','critical','{\"alertname\":\"CPU使用率过高\",\"instance\":\"192.168.10.100\",\"severity\":\"critical\"}','192168.0.51:9100',NULL,NULL,NULL,NULL,'resolved',NULL,NULL,NULL),(15,'CPU使用率过高','','{\"alertname\":\"CPU使用率过高\",\"instance\":\"192.168.10.100\",\"severity\":\"critical\"}','192.168.10.100','2026-04-06T10:00:00Z','0001-01-01T00:00:00Z','','服务器 CPU 使用率当前已达到 95% 以上','firing','2026-04-09 15:09:47.857','','0000-00-00 00:00:00.000'),(16,'CPU使用率过高','','{\"alertname\":\"CPU使用率过高\",\"instance\":\"192.168.1.1\",\"severity\":\"critical\"}','192.168.1.1','','','','','firing','2026-04-09 15:31:33.128','','0000-00-00 00:00:00.000'),(17,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T15:37:38+08:00','','内存告警','','resolved','2026-04-09 15:40:45.875','','0000-00-00 00:00:00.000'),(18,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T15:37:38+08:00','','内存告警','','resolved','2026-04-09 15:40:45.875','','0000-00-00 00:00:00.000'),(19,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T15:37:38+08:00','','内存告警','','resolved','2026-04-09 15:40:45.886','','0000-00-00 00:00:00.000'),(20,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T15:41:14+08:00','','内存告警','','firing','2026-04-09 15:41:14.112','','0000-00-00 00:00:00.000'),(21,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T15:41:14+08:00','','内存告警','','firing','2026-04-09 15:41:14.113','','0000-00-00 00:00:00.000'),(22,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T15:41:14+08:00','','内存告警','','firing','2026-04-09 15:41:14.133','','0000-00-00 00:00:00.000'),(23,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T15:41:14+08:00','','内存告警','','resolved','2026-04-09 15:52:40.654','','0000-00-00 00:00:00.000'),(24,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T15:41:14+08:00','','内存告警','','resolved','2026-04-09 15:52:40.668','','0000-00-00 00:00:00.000'),(25,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T15:41:14+08:00','','内存告警','','resolved','2026-04-09 15:52:40.684','','0000-00-00 00:00:00.000'),(26,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T15:53:08+08:00','','内存告警','','firing','2026-04-09 15:53:08.831','','0000-00-00 00:00:00.000'),(27,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T15:53:08+08:00','','内存告警','','firing','2026-04-09 15:53:08.842','','0000-00-00 00:00:00.000'),(28,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T15:53:08+08:00','','内存告警','','firing','2026-04-09 15:53:08.848','','0000-00-00 00:00:00.000'),(29,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T15:53:08+08:00','','内存告警','','resolved','2026-04-09 15:54:33.225','','0000-00-00 00:00:00.000'),(30,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T15:53:08+08:00','','内存告警','','resolved','2026-04-09 15:54:33.226','','0000-00-00 00:00:00.000'),(31,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T15:53:08+08:00','','内存告警','','resolved','2026-04-09 15:54:33.236','','0000-00-00 00:00:00.000'),(32,'HighMem','','{\"alertname\":\"HighMem\",\"instance\":\"local\",\"severity\":\"warning\"}','local','2023-01-01T00:00:00Z','','','Resolved','resolved','2026-04-09 16:03:06.529','','0000-00-00 00:00:00.000'),(33,'HighMem','','{\"alertname\":\"HighMem\",\"instance\":\"local\",\"severity\":\"warning\"}','local','2023-01-01T00:00:00Z','2026-01-01T00:00:00+08:00','','Resolved','resolved','2026-04-09 16:06:12.849','','0000-00-00 00:00:00.000'),(34,'HighMem','','{\"alertname\":\"HighMem\",\"instance\":\"local\",\"severity\":\"warning\",\"team\":\"devops\"}','local','2023-01-01T00:00:00Z','2026-01-01T00:00:00+08:00','','Resolved','resolved','2026-04-09 16:07:59.001','','0000-00-00 00:00:00.000'),(35,'HighMem','','{\"alertname\":\"HighMem\",\"instance\":\"local\",\"severity\":\"warning\",\"team\":\"devops\"}','local','2023-01-01T00:00:00Z','','','Resolved','resolved','2026-04-09 16:08:17.437','','0000-00-00 00:00:00.000'),(36,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T16:10:52+08:00','','内存告警','','firing','2026-04-09 16:10:52.358','','0000-00-00 00:00:00.000'),(37,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T16:10:52+08:00','','内存告警','','firing','2026-04-09 16:10:52.358','','0000-00-00 00:00:00.000'),(38,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T16:10:52+08:00','','内存告警','','firing','2026-04-09 16:10:52.377','','0000-00-00 00:00:00.000'),(39,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T16:10:52+08:00','2026-04-09T16:11:20+08:00','内存告警','','resolved','2026-04-09 16:11:20.533','','0000-00-00 00:00:00.000'),(40,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T16:10:52+08:00','2026-04-09T16:11:20+08:00','内存告警','','resolved','2026-04-09 16:11:20.532','','0000-00-00 00:00:00.000'),(41,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T16:10:52+08:00','2026-04-09T16:11:20+08:00','内存告警','','resolved','2026-04-09 16:11:20.544','','0000-00-00 00:00:00.000'),(42,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"cluster\":\"default\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"warning\"}','192.168.0.53:9100','2026-04-09T17:00:20+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:01:45.420','','0000-00-00 00:00:00.000'),(43,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"cluster\":\"default\",\"instance\":\"192.168.0.51:9100\",\"severity\":\"warning\"}','192.168.0.51:9100','2026-04-09T17:00:20+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:01:45.421','','0000-00-00 00:00:00.000'),(44,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"cluster\":\"default\",\"instance\":\"192.168.0.52:9100\",\"severity\":\"warning\"}','192.168.0.52:9100','2026-04-09T17:00:20+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:01:45.424','','0000-00-00 00:00:00.000'),(45,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:10:15+08:00','','内存告警','','firing','2026-04-09 17:10:15.525','','0000-00-00 00:00:00.000'),(46,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T17:10:15+08:00','','内存告警','','firing','2026-04-09 17:10:15.527','','0000-00-00 00:00:00.000'),(47,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T17:10:15+08:00','','内存告警','','firing','2026-04-09 17:10:15.566','','0000-00-00 00:00:00.000'),(48,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T17:10:15+08:00','2026-04-09T17:11:11+08:00','内存告警','','resolved','2026-04-09 17:11:11.777','','0000-00-00 00:00:00.000'),(49,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:10:15+08:00','2026-04-09T17:11:11+08:00','内存告警','','resolved','2026-04-09 17:11:11.779','','0000-00-00 00:00:00.000'),(50,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T17:10:15+08:00','2026-04-09T17:11:11+08:00','内存告警','','resolved','2026-04-09 17:11:11.811','','0000-00-00 00:00:00.000'),(51,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"warning\"}','192.168.0.53:9100','2026-04-09T10:31:44+08:00','2026-04-09T17:18:43+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:18:43.813','','0000-00-00 00:00:00.000'),(52,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.52:9100\",\"severity\":\"warning\"}','192.168.0.52:9100','2026-04-09T10:31:44+08:00','2026-04-09T17:18:43+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:18:43.813','','0000-00-00 00:00:00.000'),(53,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.51:9100\",\"severity\":\"warning\"}','192.168.0.51:9100','2026-04-09T10:31:44+08:00','2026-04-09T17:18:43+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:18:43.818','','0000-00-00 00:00:00.000'),(54,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.52:9100\",\"severity\":\"warning\"}','192.168.0.52:9100','2026-04-09T17:19:11+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:19:12.003','','0000-00-00 00:00:00.000'),(55,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"warning\"}','192.168.0.53:9100','2026-04-09T17:19:11+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:19:12.007','','0000-00-00 00:00:00.000'),(56,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.51:9100\",\"severity\":\"warning\"}','192.168.0.51:9100','2026-04-09T17:19:11+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:19:12.009','','0000-00-00 00:00:00.000'),(57,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"critical\"}','192.168.0.53:9100','2026-04-09T17:19:11+08:00','2026-04-09T17:21:04+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:21:04.446','','0000-00-00 00:00:00.000'),(58,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.52:9100\",\"severity\":\"critical\"}','192.168.0.52:9100','2026-04-09T17:19:11+08:00','2026-04-09T17:21:04+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:21:04.447','','0000-00-00 00:00:00.000'),(59,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.51:9100\",\"severity\":\"critical\"}','192.168.0.51:9100','2026-04-09T17:19:11+08:00','2026-04-09T17:21:04+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:21:04.474','','0000-00-00 00:00:00.000'),(60,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.52:9100\",\"severity\":\"critical\"}','192.168.0.52:9100','2026-04-09T17:22:00+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:22:00.705','','0000-00-00 00:00:00.000'),(61,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.51:9100\",\"severity\":\"critical\"}','192.168.0.51:9100','2026-04-09T17:22:00+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:22:00.705','','0000-00-00 00:00:00.000'),(62,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"critical\"}','192.168.0.53:9100','2026-04-09T17:22:00+08:00','','CPU 使用率高','CPU 使用率持续 > 90%。','firing','2026-04-09 17:22:00.705','','0000-00-00 00:00:00.000'),(63,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.52:9100\",\"severity\":\"critical\"}','192.168.0.52:9100','2026-04-09T17:22:00+08:00','2026-04-09T17:23:26+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:23:27.003','','0000-00-00 00:00:00.000'),(64,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.51:9100\",\"severity\":\"critical\"}','192.168.0.51:9100','2026-04-09T17:22:00+08:00','2026-04-09T17:23:26+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:23:27.005','','0000-00-00 00:00:00.000'),(65,'HighCPUUsage','','{\"alertname\":\"HighCPUUsage\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"critical\"}','192.168.0.53:9100','2026-04-09T17:22:00+08:00','2026-04-09T17:23:26+08:00','CPU 使用率高','CPU 使用率持续 > 90%。','resolved','2026-04-09 17:23:27.005','','0000-00-00 00:00:00.000'),(66,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:24:23+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:24:23.226','','0000-00-00 00:00:00.000'),(67,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T17:24:23+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:24:23.225','','0000-00-00 00:00:00.000'),(68,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T17:24:23+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:24:23.233','','0000-00-00 00:00:00.000'),(69,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T17:24:23+08:00','2026-04-09T17:26:43+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:26:43.896','','0000-00-00 00:00:00.000'),(70,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:24:23+08:00','2026-04-09T17:26:43+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:26:43.896','','0000-00-00 00:00:00.000'),(71,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T17:24:23+08:00','2026-04-09T17:26:43+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:26:43.899','','0000-00-00 00:00:00.000'),(72,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:28:08+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:28:08.299','','0000-00-00 00:00:00.000'),(73,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T17:28:08+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:28:08.299','','0000-00-00 00:00:00.000'),(74,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T17:28:08+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:28:08.301','','0000-00-00 00:00:00.000'),(75,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:28:08+08:00','2026-04-09T17:37:35+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:37:35.468','','0000-00-00 00:00:00.000'),(76,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.52:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-klpcj\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.52:9100','2026-04-09T17:28:08+08:00','2026-04-09T17:37:35+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:37:35.472','','0000-00-00 00:00:00.000'),(77,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.53:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-p4wjr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.53:9100','2026-04-09T17:28:08+08:00','2026-04-09T17:37:35+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:37:35.471','','0000-00-00 00:00:00.000'),(78,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:38:31+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:38:31.684','','0000-00-00 00:00:00.000'),(79,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:38:31+08:00','2026-04-09T17:39:27+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:39:27.963','','0000-00-00 00:00:00.000'),(80,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:42:46+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 17:42:46.752','','0000-00-00 00:00:00.000'),(81,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T17:42:46+08:00','2026-04-09T17:44:11+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 17:44:11.137','','0000-00-00 00:00:00.000'),(82,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T22:43:25+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 22:43:25.311','','0000-00-00 00:00:00.000'),(83,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T22:43:25+08:00','2026-04-09T22:44:21+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 22:44:21.531','','0000-00-00 00:00:00.000'),(84,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T22:46:44+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-04-09 22:46:44.406','','0000-00-00 00:00:00.000'),(85,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-04-09T22:46:44+08:00','2026-04-09T22:53:54+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-04-09 22:53:54.918','','0000-00-00 00:00:00.000'),(86,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-05-09T22:44:00+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-05-09 22:44:00.715','','0000-00-00 00:00:00.000'),(87,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-05-09T22:44:00+08:00','2026-05-09T22:45:57+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-05-09 22:45:57.360','','0000-00-00 00:00:00.000'),(88,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-05-10T10:08:24+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-05-10 10:08:24.509','','0000-00-00 00:00:00.000'),(89,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-05-10T10:08:24+08:00','2026-05-10T10:09:22+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-05-10 10:09:22.810','','0000-00-00 00:00:00.000'),(90,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-05-10T10:50:25+08:00','','内存告警','内存 使用率持续 > 90%。','firing','2026-05-10 10:50:25.150','','0000-00-00 00:00:00.000'),(91,'HighMemoryUsage','','{\"alertname\":\"HighMemoryUsage\",\"container\":\"node-exporter\",\"endpoint\":\"http-metrics\",\"instance\":\"192.168.0.51:9100\",\"job\":\"node-exporter\",\"namespace\":\"monitor\",\"pod\":\"prometheus-prometheus-node-exporter-vs7pr\",\"service\":\"prometheus-prometheus-node-exporter\",\"severity\":\"warning\",\"team\":\"devops\"}','192.168.0.51:9100','2026-05-10T10:50:25+08:00','2026-05-10T10:50:44+08:00','内存告警','内存 使用率持续 > 90%。','resolved','2026-05-10 10:50:44.722','','0000-00-00 00:00:00.000'),(92,'domain_test','','{\"alertname\":\"domain_test\",\"domain\":\"bookr.stariverfeel.eu.org\",\"port\":\"443\",\"remaining_days\":\"69\",\"severity\":\"warning\",\"status\":\"1\",\"team\":\"devops\"}','','2026-05-19T22:26:34+08:00','','','','firing','2026-05-19 22:27:49.203','','0000-00-00 00:00:00.000'),(93,'domain_test','','{\"alertname\":\"domain_test\",\"domain\":\"bookr.stariverfeel.eu.org\",\"port\":\"443\",\"remaining_days\":\"69\",\"severity\":\"warning\",\"status\":\"1\",\"team\":\"devops\"}','','2026-05-19T22:51:11+08:00','','','','firing','2026-05-19 22:52:26.765','','0000-00-00 00:00:00.000'),(94,'domain_test','','{\"alertname\":\"domain_test\",\"domain\":\"bookr.stariverfeel.eu.org\",\"port\":\"443\",\"remaining_days\":\"69\",\"severity\":\"warning\",\"status\":\"1\",\"team\":\"devops\"}','','2026-05-19T22:51:11+08:00','2026-05-19T22:53:02+08:00','','','resolved','2026-05-19 22:53:02.016','','0000-00-00 00:00:00.000'),(95,'api-test','','{\"alertname\":\"api-test\",\"last_response_ms\":\"2\",\"last_status_code\":\"404\",\"method\":\"GET\",\"name\":\"api-test\",\"severity\":\"warning\",\"status\":\"2\",\"team\":\"devops\",\"url\":\"http://172.22.107.76:8080/incident/list\"}','','2026-05-19T23:19:33+08:00','','','','firing','2026-05-19 23:20:48.103','','0000-00-00 00:00:00.000'),(96,'api-test','','{\"alertname\":\"api-test\",\"severity\":\"warning\",\"team\":\"devops\"}','','2026-05-19T23:19:33+08:00','2026-05-19T23:23:53+08:00','','','resolved','2026-05-19 23:23:53.256','','0000-00-00 00:00:00.000'),(97,'domain_test','','{\"alertname\":\"domain_test\",\"domain\":\"bookr.stariverfeel.eu.org\",\"port\":\"443\",\"remaining_days\":\"69\",\"severity\":\"warning\",\"status\":\"1\",\"team\":\"devops\"}','','2026-05-19T23:37:30+08:00','','','','firing','2026-05-19 23:37:30.252','','0000-00-00 00:00:00.000'),(98,'domain_test','','{\"alertname\":\"domain_test\",\"domain\":\"bookr.stariverfeel.eu.org\",\"port\":\"443\",\"remaining_days\":\"69\",\"severity\":\"warning\",\"status\":\"1\",\"team\":\"devops\"}','','2026-05-19T23:37:30+08:00','2026-05-22T23:55:43+08:00','','','resolved','2026-05-22 23:55:43.619','','0000-00-00 00:00:00.000'),(99,'HighCPUUsageCritical','','{\"alertname\":\"HighCPUUsageCritical\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"critical\"}','192.168.0.53:9100','2026-05-25T21:52:50+08:00','','CPU 使用率高','CPU 使用率持续 > 95%。','firing','2026-05-25 21:56:14.616','','0000-00-00 00:00:00.000'),(100,'HighCPUUsageCritical','','{\"alertname\":\"HighCPUUsageCritical\",\"instance\":\"192.168.0.53:9100\",\"severity\":\"critical\"}','192.168.0.53:9100','2026-05-25T21:52:50+08:00','2026-05-25T21:57:10+08:00','CPU 使用率高','CPU 使用率持续 > 95%。','resolved','2026-05-25 21:57:10.204','','0000-00-00 00:00:00.000'),(101,'api-test','','{\"alertname\":\"api-test\",\"last_response_ms\":\"165\",\"last_status_code\":\"200\",\"method\":\"GET\",\"name\":\"api-test\",\"severity\":\"warning\",\"status\":\"2\",\"team\":\"devops\",\"url\":\"http://172.22.107.76:8080/incident/list\"}','','2026-06-26T02:18:10+08:00','','','','firing','2026-06-26 02:19:12.699','','0000-00-00 00:00:00.000');
/*!40000 ALTER TABLE `monitor_alert_record` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_router`
--

DROP TABLE IF EXISTS `monitor_alert_router`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_router` (
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
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_router`
--

LOCK TABLES `monitor_alert_router` WRITE;
/*!40000 ALTER TABLE `monitor_alert_router` DISABLE KEYS */;
INSERT INTO `monitor_alert_router` VALUES (4,'test',3,'[{\"key\":\"severity\",\"type\":\"等于\",\"value\":\"warning\"},{\"key\":\"team\",\"type\":\"等于\",\"value\":\"devops\"}]','https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=6b93c3ab-8216-417e-bfd2-14ee5d18db33','@周俊杰',0,1,'0000-00-00 00:00:00.000');
/*!40000 ALTER TABLE `monitor_alert_router` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_rule`
--

DROP TABLE IF EXISTS `monitor_alert_rule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_rule` (
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
) ENGINE=InnoDB AUTO_INCREMENT=37 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_rule`
--

LOCK TABLES `monitor_alert_rule` WRITE;
/*!40000 ALTER TABLE `monitor_alert_rule` DISABLE KEYS */;
INSERT INTO `monitor_alert_rule` VALUES (13,'2026-04-08 17:58:52.164','2026-05-10 09:50:30.573',NULL,9,'HighCPUUsage','100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 90','0m','{\"severity\":\"critical\"}','{\"instance\":\"192.168.0.51:9100\"}','warning','CPU 使用率高','CPU 使用率持续 > 90%。','alert: HighCPUUsage\nexpr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 90\nfor: 3m\nlabels:\n    severity: warning\nannotations:\n    description: CPU 使用率持续 > 90%。\n    summary: CPU 使用率高\n','inactive','CPU',1),(14,'2026-04-08 17:58:52.174','2026-06-26 23:52:33.936',NULL,9,'HighCPUUsageCritical','100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 95','3m','{\"severity\":\"critical\"}','','critical','CPU 使用率高','CPU 使用率持续 > 95%。','alert: HighCPUUsageCritical\nexpr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 95\nfor: 3m\nlabels:\n    severity: critical\nannotations:\n    description: CPU 使用率持续 > 95%。\n    summary: CPU 使用率高\n','inactive','CPU',1),(24,'2026-04-08 21:10:42.093','2026-05-10 10:50:44.708',NULL,9,'HighMemoryUsage','(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 90','0m','{\"team\": \"devops\"}','{\"instance\":\"192.168.0.51:9100\"}','warning','内存告警','内存 使用率持续 > 90%。','','inactive','CPU',1),(35,'2026-05-19 22:26:03.472','2026-05-22 23:55:43.571',NULL,10,'domain_test','remaining_days <= 40','0m','{\"team\": \"devops\"}','{\"env\": \"prod\"}','warning','','','','inactive','domain_cert',1),(36,'2026-05-19 23:19:28.002','2026-06-27 00:00:27.582',NULL,10,'api-test','status != 1','1m','{\"team\": \"devops\"}','{\"env\": \"prod\"}','warning','','','','firing','api_endpoint',0);
/*!40000 ALTER TABLE `monitor_alert_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_rule_style`
--

DROP TABLE IF EXISTS `monitor_alert_rule_style`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_rule_style` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(64) NOT NULL COMMENT '分类名称(如CPU,Memory)',
  `description` varchar(255) DEFAULT NULL COMMENT '描述',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_monitor_alert_rule_style_name` (`name`),
  KEY `idx_monitor_alert_rule_style_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_rule_style`
--

LOCK TABLES `monitor_alert_rule_style` WRITE;
/*!40000 ALTER TABLE `monitor_alert_rule_style` DISABLE KEYS */;
INSERT INTO `monitor_alert_rule_style` VALUES (3,'2026-04-08 15:33:08.622','2026-04-08 15:33:08.622',NULL,'CPU','CPU指标告警规则'),(9,'2026-05-19 22:22:55.893','2026-05-19 22:22:55.893',NULL,'domain_cert','域名证书'),(10,'2026-05-19 22:23:20.779','2026-05-19 22:23:20.779',NULL,'api_endpoint','API监控');
/*!40000 ALTER TABLE `monitor_alert_rule_style` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_aliyun_config`
--

DROP TABLE IF EXISTS `monitor_aliyun_config`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_aliyun_config` (
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_aliyun_config`
--

LOCK TABLES `monitor_aliyun_config` WRITE;
/*!40000 ALTER TABLE `monitor_aliyun_config` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_aliyun_config` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_api_endpoint`
--

DROP TABLE IF EXISTS `monitor_api_endpoint`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_api_endpoint` (
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
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API端点监控表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_api_endpoint`
--

LOCK TABLES `monitor_api_endpoint` WRITE;
/*!40000 ALTER TABLE `monitor_api_endpoint` DISABLE KEYS */;
INSERT INTO `monitor_api_endpoint` VALUES (1,'api-test','http://172.22.107.76:8080/incident/list','GET','{\"Authorization\": \"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ODksInVzZXJuYW1lIjoiYWRtaW4iLCJuaWNrbmFtZSI6ImFkbWluIiwiaWNvbiI6Imh0dHA6Ly8xOTIuMTY4LjMuNzo4MDgwL2FwaS92MS91cGxvYWQvMjAyNTEyMTMvODYyMzI4MDAwLnBuZyIsImVtYWlsIjoiMTIzNDU2Nzg5QHFxLmNvbSIsInBob25lIjoiMTM3NTQzNTQ1MzYiLCJub3RlIjoi5ZCO56uv56CU5Y-RIiwiaXNzIjoiYWRtaW4ifQ.y-at-70OM1wVAK4nOt9dZSNL93xfZohCWHNXZoM13I4\"}','',300,10,404,'',200,165,2,'2026-06-26 02:17:44','状态码异常: 期望404, 实际200','2026-05-19 23:11:34.300','2026-06-26 02:17:44.431');
/*!40000 ALTER TABLE `monitor_api_endpoint` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_data_source`
--

DROP TABLE IF EXISTS `monitor_data_source`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_data_source` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(128) NOT NULL COMMENT '数据源名称，如：北京核心机房-K8s集群',
  `type` varchar(64) NOT NULL COMMENT '数据源类型：Prometheus, Nightingale, Zabbix等',
  `deploy_method` varchar(64) NOT NULL COMMENT '部署方式：Kubernetes, Docker, Host, CloudManaged等',
  `api_url` varchar(255) NOT NULL COMMENT '数据源的直连/查询接口地址',
  `config` json NOT NULL COMMENT '动态凭证配置 (格式由 deploy_method 决定)',
  `status` tinyint(1) DEFAULT '1' COMMENT '状态：1-启用，0-停用',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_data_source`
--

LOCK TABLES `monitor_data_source` WRITE;
/*!40000 ALTER TABLE `monitor_data_source` DISABLE KEYS */;
INSERT INTO `monitor_data_source` VALUES (1,'prometheus-01','Prometheus','Kubernetes','http://192.168.0.51:30901/','{\"token\": \"eyJhbGciOiJSUzI1NiIsImtpZCI6IkRtejVzMmo3b3JkMFFOa0Etby1JdVRBZWZfM2MtcTZMSTExbjZJajJfOUEifQ.eyJhdWQiOlsiYXBpIiwiaXN0aW8tY2EiXSwiZXhwIjoyMDkwOTAyNzY3LCJpYXQiOjE3NzU1NDI3NjcsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2YyIsImp0aSI6ImRiNjg4NjViLTRkM2EtNDAxZC1hYzY5LWFmOTkwNGJkMTNiMSIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoibW9uaXRvciIsInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJtb25pdG9yLW1hbmFnZXItc2EiLCJ1aWQiOiI4NmYwYTAyMS0wNjk1LTQ1ZTYtYmQ0OS00NjI1MTg2MTU2OGQifX0sIm5iZiI6MTc3NTU0Mjc2Nywic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Om1vbml0b3I6bW9uaXRvci1tYW5hZ2VyLXNhIn0.ez8h2RjDcMc7-OW55JWcz1h2IOMTQitEv-F8tqqIZNjV8EUbkzfU8CPU4UdqgLeZK26eh4-0Ql_I-7GgmS_Il93Dxow9jHi1ihwrQc7oao9EHCRyjEqDVUuJdECY4SKEolwupeFc_dJYQre0UNGbxPttyqFNfh5-36gekIe4rt96-5F-yDL_U6Jfv58wXEXGPl-ReAowjwHhY9djCrFYtKbA4Fs6hkmycIi_34lqn0w-9RmJ4MWss0XosUEqtPBw41EhO4cJ7xsrQgHEenQBKQOV7RGhtPfA8i8Pyd0OiYvKyb1dzXFbbB96zWOKdMKXIweFjMkD6V4cP6pg0r196A\", \"auth_type\": \"service_account\", \"namespace\": \"monitor\", \"k8s_api_url\": \"https://192.168.0.51:6443\", \"insecure_skip_tls_verify\": true}',1);
/*!40000 ALTER TABLE `monitor_data_source` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_domain`
--

DROP TABLE IF EXISTS `monitor_domain`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_domain` (
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
) ENGINE=InnoDB AUTO_INCREMENT=42 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_domain`
--

LOCK TABLES `monitor_domain` WRITE;
/*!40000 ALTER TABLE `monitor_domain` DISABLE KEYS */;
INSERT INTO `monitor_domain` VALUES (6,'gitee.com','','',1,1,200,484,'2026-02-17 23:59:59',28,'TrustAsia DV TLS RSA CA 2025','2026-01-20 10:33:28','','2025-12-04 10:21:13.921','2026-01-20 10:33:28.279'),(7,'ops.dding.net','','',1,0,0,0,'2025-12-20 23:59:59',11,'Encryption Everywhere DV TLS CA - G1','2026-01-20 10:47:47','连接失败: Get \"http://ops.dding.net\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)','2025-12-04 10:21:28.741','2026-01-20 10:47:46.788'),(8,'docker.aityp.com','','',1,1,200,494,'2026-03-23 03:50:15',62,'E8','2026-01-20 10:33:28','','2025-12-04 10:21:54.359','2026-01-20 10:33:28.289'),(12,'www.jd.com','ops','123',1,1,200,101,'2026-12-20 04:28:34',334,'GlobalSign RSA OV SSL CA 2018','2026-01-20 10:33:28','','2025-12-05 20:50:58.505','2026-01-20 10:33:27.897'),(28,'ai.feishu.cn','','',1,1,200,1913,'2026-04-17 23:59:59',87,'RapidSSL TLS RSA CA G1','2026-01-20 10:33:30','','2025-12-08 22:30:11.616','2026-01-20 10:33:29.709'),(41,'www.hyunying.com','','',1,1,200,752,'2026-04-19 23:59:59',89,'Encryption Everywhere DV TLS CA - G2','2026-01-20 10:47:24','','2026-01-20 10:32:39.214','2026-01-20 10:47:23.836');
/*!40000 ALTER TABLE `monitor_domain` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_domain_cert`
--

DROP TABLE IF EXISTS `monitor_domain_cert`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_domain_cert` (
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
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='域名证书监控表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_domain_cert`
--

LOCK TABLES `monitor_domain_cert` WRITE;
/*!40000 ALTER TABLE `monitor_domain_cert` DISABLE KEYS */;
INSERT INTO `monitor_domain_cert` VALUES (1,'baidu.com',443,'DigiCert Secure Site Pro G2 TLS CN RSA4096 SHA256 2022 CA1','www.baidu.cn','2026-02-04 00:00:00','2027-03-03 23:59:59',277,1,'2026-05-31 21:33:55','','2026-05-18 22:57:48.434','2026-05-31 21:33:56.546'),(4,'bookr.stariverfeel.eu.org',443,'WE1','bookr.stariverfeel.eu.org','2026-04-26 21:09:25','2026-07-25 22:09:22',56,1,'2026-05-31 21:33:56','','2026-05-18 22:59:26.845','2026-05-31 21:33:57.439');
/*!40000 ALTER TABLE `monitor_domain_cert` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_domain_schedule`
--

DROP TABLE IF EXISTS `monitor_domain_schedule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_domain_schedule` (
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
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_domain_schedule`
--

LOCK TABLES `monitor_domain_schedule` WRITE;
/*!40000 ALTER TABLE `monitor_domain_schedule` DISABLE KEYS */;
INSERT INTO `monitor_domain_schedule` VALUES (1,0,2,NULL,NULL,'stopped','2025-12-04 23:09:32.660','2025-12-05 23:12:27.687',NULL),(2,0,120,NULL,NULL,'stopped','2025-12-04 23:09:32.660','2025-12-04 23:09:32.660',NULL);
/*!40000 ALTER TABLE `monitor_domain_schedule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_incident`
--

DROP TABLE IF EXISTS `monitor_incident`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_incident` (
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
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_incident`
--

LOCK TABLES `monitor_incident` WRITE;
/*!40000 ALTER TABLE `monitor_incident` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_incident` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_prometheus_alert`
--

DROP TABLE IF EXISTS `monitor_prometheus_alert`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_prometheus_alert` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `tpltype` longtext,
  `tpluse` longtext,
  `tplname` varchar(191) DEFAULT NULL,
  `tpl` text,
  `webhook_content_type` longtext,
  `created` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_monitor_prometheus_alert_tplname` (`tplname`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_prometheus_alert`
--

LOCK TABLES `monitor_prometheus_alert` WRITE;
/*!40000 ALTER TABLE `monitor_prometheus_alert` DISABLE KEYS */;
INSERT INTO `monitor_prometheus_alert` VALUES (3,'wx','Prometheus','prometheus-wx','{{ $var := .externalURL}}{{ range $k,$v:=.alerts }}\n{{if eq $v.status \"resolved\"}}\n#### [Prometheus恢复信息]({{$v.generatorURL}})\n> <font color=\"info\">告警名称</font>：[{{$v.labels.alertname}}]({{$var}})\n> <font color=\"info\">告警级别</font>：{{$v.labels.severity}}\n> <font color=\"info\">开始时间</font>：{{GetCSTtime $v.startsAt}}\n> <font color=\"info\">结束时间</font>：{{GetCSTtime $v.endsAt}}\n> <font color=\"info\">故障主机</font>：{{$v.labels.instance}}\n> <font color=\"info\">恢复描述</font>：{{$v.annotations.description}}\n{{else}}\n#### [Prometheus告警信息]({{$v.generatorURL}})\n> <font color=\"#FF0000\">告警名称</font>：[{{$v.labels.alertname}}]({{$var}})\n> <font color=\"#FF0000\">告警级别</font>：{{$v.labels.severity}}\n> <font color=\"#FF0000\">开始时间</font>：{{GetCSTtime $v.startsAt}}\n> <font color=\"#FF0000\">故障主机</font>：{{$v.labels.instance}}\n> <font color=\"#FF0000\">告警描述</font>：{{$v.annotations.description}}\n{{end}}\n{{ end }}','application/json','0000-00-00 00:00:00.000'),(4,'wx','Prometheus','TestWXTemplate','## [告警]：收到新的节点告警\n {{.Alerts}}','application/json','0000-00-00 00:00:00.000'),(5,'dd','Prometheus','TestWXTemplate','## [告警]：收到新的节点告警\n {{.Alerts}}','application/json','0000-00-00 00:00:00.000'),(6,'wx','Prometheus','TestWXTemplate1-Updated','## [严重告警]：\n {{.Alerts}}','application/json','0000-00-00 00:00:00.000'),(7,'wx','Prometheus','TestWXTemplate1-Updated','## [严重告警]：\n {{.Alerts}}','application/json','0000-00-00 00:00:00.000'),(8,'wx','Prometheus','TestWXTemplate2-Updated','## [严重告警]：\n {{.alerts}}','application/json','0000-00-00 00:00:00.000'),(9,'wx','Prometheus','TestWXTemplate2-Updated','## [严重告警]：\n {{.alerts}}','application/json','0000-00-00 00:00:00.000'),(10,'wx','Prometheus','TestWXTemplate2-Updated','## [严重告警]：\n {{.alerts}}','application/json','0000-00-00 00:00:00.000'),(11,'wx','Prometheus','TestWXTemplate2-Updated','## [严重告警]：\n {{.alerts}}','application/json','0000-00-00 00:00:00.000');
/*!40000 ALTER TABLE `monitor_prometheus_alert` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_ssl_cert`
--

DROP TABLE IF EXISTS `monitor_ssl_cert`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_ssl_cert` (
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_ssl_cert`
--

LOCK TABLES `monitor_ssl_cert` WRITE;
/*!40000 ALTER TABLE `monitor_ssl_cert` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_ssl_cert` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `quick_deployment_tasks`
--

DROP TABLE IF EXISTS `quick_deployment_tasks`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `quick_deployment_tasks` (
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
) ENGINE=InnoDB AUTO_INCREMENT=44 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `quick_deployment_tasks`
--

LOCK TABLES `quick_deployment_tasks` WRITE;
/*!40000 ALTER TABLE `quick_deployment_tasks` DISABLE KEYS */;
/*!40000 ALTER TABLE `quick_deployment_tasks` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `quick_deployments`
--

DROP TABLE IF EXISTS `quick_deployments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `quick_deployments` (
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
) ENGINE=InnoDB AUTO_INCREMENT=34 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `quick_deployments`
--

LOCK TABLES `quick_deployments` WRITE;
/*!40000 ALTER TABLE `quick_deployments` DISABLE KEYS */;
/*!40000 ALTER TABLE `quick_deployments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `redis_instance`
--

DROP TABLE IF EXISTS `redis_instance`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `redis_instance` (
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `redis_instance`
--

LOCK TABLES `redis_instance` WRITE;
/*!40000 ALTER TABLE `redis_instance` DISABLE KEYS */;
/*!40000 ALTER TABLE `redis_instance` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_activity_log`
--

DROP TABLE IF EXISTS `sys_activity_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_activity_log` (
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_activity_log`
--

LOCK TABLES `sys_activity_log` WRITE;
/*!40000 ALTER TABLE `sys_activity_log` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_activity_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_admin`
--

DROP TABLE IF EXISTS `sys_admin`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_admin` (
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
) ENGINE=InnoDB AUTO_INCREMENT=107 DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='后台管理员表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_admin`
--

LOCK TABLES `sys_admin` WRITE;
/*!40000 ALTER TABLE `sys_admin` DISABLE KEYS */;
INSERT INTO `sys_admin` VALUES (89,1,15,'admin','e10adc3949ba59abbe56e057f20f883e','admin','http://192.168.3.7:8080/api/v1/upload/20251213/862328000.png','123456789@qq.com','13754354536','后端研发','2023-05-23 22:15:50',1),(106,10,5,'test','e10adc3949ba59abbe56e057f20f883e','test','','2065594589@qq.com','17377797941','','2026-04-13 22:29:49',1);
/*!40000 ALTER TABLE `sys_admin` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_admin_role`
--

DROP TABLE IF EXISTS `sys_admin_role`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_admin_role` (
  `admin_id` int NOT NULL COMMENT '管理员id',
  `role_id` int NOT NULL COMMENT '角色id'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='管理员和角色关系表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_admin_role`
--

LOCK TABLES `sys_admin_role` WRITE;
/*!40000 ALTER TABLE `sys_admin_role` DISABLE KEYS */;
INSERT INTO `sys_admin_role` VALUES (89,1),(106,13);
/*!40000 ALTER TABLE `sys_admin_role` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_config`
--

DROP TABLE IF EXISTS `sys_config`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_config` (
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
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_config`
--

LOCK TABLES `sys_config` WRITE;
/*!40000 ALTER TABLE `sys_config` DISABLE KEYS */;
INSERT INTO `sys_config` VALUES (1,'ldap','ldap','{\"enable\":true,\"host\":\"www.baidu.com\",\"port\":389,\"baseDn\":\"dc=dding,dc=cn\",\"bindUser\":\"cn=reader,dc=dding,dc=cn\",\"bindPass\":\"fsyunding2018\",\"authFilter\":\"(\\u0026(cn=%s))\",\"coverAttributes\":true,\"tls\":false,\"startTLS\":false,\"defaultRoles\":null,\"defaultRoleId\":13,\"attributes\":{\"nickname\":\"cn\",\"phone\":\"mobile\",\"email\":\"mail\"},\"remark\":\"\"}',1,'','2025-12-09 13:19:56.671','2025-12-14 14:56:54.371');
/*!40000 ALTER TABLE `sys_config` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_dept`
--

DROP TABLE IF EXISTS `sys_dept`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_dept` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` int NOT NULL COMMENT '父id',
  `dept_type` int NOT NULL COMMENT '部门类型（1->公司, 2->中心，3->部门）',
  `dept_name` varchar(30) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin NOT NULL COMMENT '部门名称',
  `dept_status` int NOT NULL DEFAULT '1' COMMENT '部门状态（1->正常 2->停用）',
  `create_time` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `dept_name` (`dept_name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin ROW_FORMAT=DYNAMIC COMMENT='部门表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dept`
--

LOCK TABLES `sys_dept` WRITE;
/*!40000 ALTER TABLE `sys_dept` DISABLE KEYS */;
INSERT INTO `sys_dept` VALUES (1,0,1,'神舟科技有限公司',1,'2023-06-14 17:53:23'),(2,1,2,'深圳研发中心',1,'2023-06-14 17:53:55'),(3,2,3,'架构设计部门',1,'2023-06-14 17:54:15'),(5,2,3,'后端研发部门',1,'2023-06-14 17:55:25'),(6,2,3,'系统测试部门',1,'2023-06-14 17:55:31'),(12,1,2,'北京研发中心',1,'2025-06-28 23:42:46'),(13,1,2,'重庆研发中心',1,'2025-06-28 23:43:15'),(14,12,3,'运维1部',1,'2025-06-28 23:43:34'),(15,13,3,'运维2部',1,'2025-06-28 23:44:15'),(16,13,3,'重庆研发部-001',1,'2025-07-04 13:10:58');
/*!40000 ALTER TABLE `sys_dept` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_login_info`
--

DROP TABLE IF EXISTS `sys_login_info`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_login_info` (
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
) ENGINE=InnoDB AUTO_INCREMENT=561 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin ROW_FORMAT=DYNAMIC COMMENT='登录日志记录';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_login_info`
--

LOCK TABLES `sys_login_info` WRITE;
/*!40000 ALTER TABLE `sys_login_info` DISABLE KEYS */;
INSERT INTO `sys_login_info` VALUES (450,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-19 11:50:39'),(451,'string','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',2,'验证码已过期','2026-01-20 11:04:33'),(452,'string','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',2,'验证码已过期','2026-01-20 11:04:40'),(453,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',2,'验证码已过期','2026-01-20 11:04:54'),(454,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-20 11:05:19'),(455,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',2,'验证码已过期','2026-01-20 11:06:07'),(456,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-20 11:07:16'),(457,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-20 11:24:21'),(458,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',2,'验证码不正确','2026-01-21 22:08:01'),(459,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-21 22:08:07'),(460,'admin','172.20.0.1','局域网','Chrome/144.0.0.0','Windows 10',1,'登录成功','2026-01-22 11:10:42'),(461,'admin','172.20.0.1','局域网','Chrome/144.0.0.0','Windows 10',1,'登录成功','2026-01-23 14:01:04'),(462,'admin','172.20.0.1','局域网','Chrome/144.0.0.0','Windows 10',1,'登录成功','2026-01-23 14:26:50'),(463,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-24 02:36:21'),(464,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-25 02:38:13'),(465,'admin','172.20.0.1','局域网','Chrome/144.0.0.0','Windows 10',1,'登录成功','2026-01-26 09:35:28'),(466,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-26 22:58:07'),(467,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-27 21:54:19'),(468,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-28 23:20:31'),(469,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',2,'验证码不正确','2026-01-28 23:21:05'),(470,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-28 23:21:15'),(471,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-29 23:40:24'),(472,'admin','172.20.0.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-01-31 14:53:06'),(473,'admin','192.168.65.123','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-02-25 14:43:02'),(474,'admin','192.168.65.123','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-02-26 15:15:47'),(475,'admin','192.168.65.123','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-02-27 15:17:29'),(476,'admin','192.168.1.223','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-02-28 15:23:12'),(477,'admin','192.168.65.123','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-03-02 15:46:18'),(478,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-03-05 23:28:13'),(479,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-03-08 00:01:25'),(480,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-03-08 21:27:35'),(481,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',2,'验证码不正确','2026-03-08 21:30:31'),(482,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-03-08 21:30:37'),(483,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-03-27 14:08:51'),(484,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-03-30 20:47:18'),(485,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-02 11:01:09'),(486,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-02 12:11:52'),(487,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-02 15:47:23'),(488,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-03 16:09:18'),(489,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-05 23:20:11'),(490,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-05 23:50:52'),(491,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-06 20:42:09'),(492,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-07 15:23:19'),(493,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-07 15:24:50'),(494,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-08 15:32:34'),(495,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-09 15:33:01'),(496,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-10 15:34:00'),(497,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-11 15:39:41'),(498,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-12 13:03:52'),(499,'admin','172.22.96.1','局域网','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-12 17:35:36'),(500,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-13 22:20:25'),(501,'test','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-13 22:30:05'),(502,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-13 22:31:25'),(503,'admin','127.0.0.1','服务器登录','Chrome/131.0.0.0','Windows 10',1,'登录成功','2026-04-15 17:25:30'),(504,'admin','127.0.0.1','服务器登录','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-09 22:41:58'),(505,'admin','192.168.0.102','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-10 09:38:52'),(506,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-17 12:39:16'),(507,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-17 17:27:41'),(508,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-17 17:39:00'),(509,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-17 17:51:44'),(510,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-17 17:52:35'),(511,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-17 17:55:36'),(512,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-17 18:04:53'),(513,'admin','172.22.96.1','局域网','Chrome/142.0.7444.265','Windows 10',1,'登录成功','2026-05-18 22:05:34'),(514,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',2,'验证码不正确','2026-05-19 22:17:50'),(515,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-19 22:17:58'),(516,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-20 22:18:40'),(517,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-21 22:19:36'),(518,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-22 22:29:08'),(519,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-22 23:16:37'),(520,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-22 23:20:19'),(521,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-22 23:22:21'),(522,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-22 23:23:16'),(523,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-22 23:23:32'),(524,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-22 23:24:07'),(525,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-23 13:59:09'),(526,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-24 20:55:54'),(527,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-24 21:23:47'),(528,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-24 23:03:12'),(529,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-25 21:32:13'),(530,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-25 21:42:28'),(531,'admin','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-27 00:08:29'),(532,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-27 00:08:38'),(533,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-27 21:51:14'),(534,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-28 22:55:58'),(535,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-29 22:22:21'),(536,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-30 18:32:55'),(537,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-31 00:22:48'),(538,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-31 00:26:49'),(539,'test','172.22.96.1','局域网','Chrome/148.0.0.0','Windows 10',1,'登录成功','2026-05-31 21:23:27'),(540,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-21 23:04:35'),(541,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-21 23:12:04'),(542,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',2,'验证码不正确','2026-06-21 23:47:10'),(543,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-21 23:47:18'),(544,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-21 23:47:22'),(545,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-21 23:47:28'),(546,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-21 23:47:31'),(547,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-21 23:47:34'),(548,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',2,'验证码不正确','2026-06-22 00:14:51'),(549,'admin','127.0.0.1','服务器登录','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-22 00:14:55'),(550,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-23 00:45:10'),(551,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',2,'验证码不正确','2026-06-23 00:46:23'),(552,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-23 00:46:25'),(553,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-23 22:53:27'),(554,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-24 01:04:15'),(555,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',2,'验证码不正确','2026-06-25 01:05:44'),(556,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-25 01:05:53'),(557,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-26 01:08:23'),(558,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',2,'验证码不正确','2026-06-27 16:07:00'),(559,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-06-27 16:07:07'),(560,'admin','172.22.96.1','局域网','Chrome/149.0.0.0','Windows 10',1,'登录成功','2026-07-02 21:51:35');
/*!40000 ALTER TABLE `sys_login_info` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_menu`
--

DROP TABLE IF EXISTS `sys_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_menu` (
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
) ENGINE=InnoDB AUTO_INCREMENT=246 DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='菜单表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_menu`
--

LOCK TABLES `sys_menu` WRITE;
/*!40000 ALTER TABLE `sys_menu` DISABLE KEYS */;
INSERT INTO `sys_menu` VALUES (4,0,'系统管理','StarFilled','',1,'',2,7,'2022-09-04 13:57:39'),(6,4,'用户信息','Avatar','base:admin:list',2,'system/admin',2,1,'2022-09-04 13:59:39'),(7,4,'角色信息','InfoFilled','base:role:list',2,'system/role',2,2,'2022-09-04 14:00:12'),(8,4,'菜单信息','Histogram','base:menu:list',2,'system/menu',2,3,'2022-09-04 14:00:17'),(9,4,'部门信息','Menu','base:dept:list',2,'system/dept',2,4,'2022-09-04 14:01:58'),(10,4,'岗位信息','Promotion','base:post:list',2,'system/post',2,5,'2022-09-04 14:02:06'),(16,6,'新增用户','','base:admin:add',3,'',2,1,'2022-09-04 18:32:55'),(17,6,'修改用户','','base:admin:edit',3,'',2,2,'2022-09-04 18:33:29'),(18,6,'删除用户','','base:admin:delete',3,'',2,3,'2022-09-04 18:33:51'),(21,7,'新增角色','','base:role:add',3,'',2,1,'2022-09-04 18:44:47'),(22,7,'修改角色','','base:role:edit',3,'',2,2,'2022-09-04 18:45:10'),(23,7,'删除角色','','base:role:delete',3,'',2,3,'2022-09-04 18:45:46'),(24,7,'分配权限','','base:role:assign',3,'',2,4,'2022-09-04 18:46:20'),(26,8,'新增菜单','','base:menu:add',3,'',2,1,'2022-09-04 18:49:51'),(27,8,'修改菜单','','base:menu:edit',3,'',2,2,'2022-09-04 18:50:24'),(28,8,'删除菜单','','base:menu:delete',3,'',2,3,'2022-09-04 18:50:53'),(29,9,'新增部门','','base:dept:add',3,'',2,1,'2022-09-04 18:52:16'),(30,9,'修改部门','','base:dept:edit',3,'',2,2,'2022-09-04 18:52:37'),(31,9,'删除部门','','base:dept:delete',3,'',2,3,'2022-09-04 18:52:50'),(32,10,'新增岗位','','base:post:add',3,'',2,1,'2022-09-04 18:53:28'),(33,10,'修改岗位','','base:post:edit',3,'',2,2,'2022-09-04 18:53:48'),(34,10,'删除岗位','','base:post:delete',3,'',2,3,'2022-09-04 18:54:00'),(44,0,'操作审计','BellFilled','',1,'',2,9,'2022-09-05 11:06:57'),(45,44,'操作日志','User','monitor:operator:list',2,'monitor/operator',2,1,'2022-09-05 11:10:54'),(46,44,'登录日志','DocumentRemove','monitor:loginLog:list',2,'monitor/loginlog',2,2,'2022-09-05 11:11:31'),(47,45,'清空操作日志','','monitor:operator:clean',3,'',2,1,'2022-09-05 11:12:36'),(49,46,'清空登录日志','','monitor:loginLog:clean',3,'',2,1,'2022-09-05 11:16:01'),(60,6,'重置密码',NULL,'base:admin:reset',3,NULL,2,6,'2022-12-01 16:33:34'),(62,46,'删除登录日志','','monitor:loginLog:delete',3,'',2,2,'2022-12-02 15:41:56'),(72,0,'仪表盘','HomeFilled','',1,'dashboard',2,1,'2023-05-24 22:11:13'),(73,45,'删除操作日志','','monitor:operator:delete',3,'',2,3,'2023-06-02 10:09:38'),(78,80,'主机管理','Platform','cmdb:ecs:list',2,'cmdb/ecs',2,1,'2025-06-29 00:30:35'),(80,0,'资产管理','TrendCharts','',1,'',2,2,'2025-07-03 11:47:07'),(81,0,'容器管理','UploadFilled','',1,'',2,3,'2025-07-03 11:50:47'),(82,81,'集群管理','Menu','cloud:k8s:list',2,'k8s/list',2,1,'2025-07-03 11:56:44'),(83,81,'节点管理','Help','cloud:k8s:node',2,'k8s/node',2,2,'2025-07-03 12:04:59'),(84,0,'配置中心','Tools','',1,'',2,8,'2025-07-04 17:00:01'),(85,84,'主机凭据','Setting','config:ecs:key',2,'config/ecskey',2,1,'2025-07-04 17:03:10'),(86,84,'通用凭据','User','config:accountauth:key',2,'config/accountauth',2,2,'2025-07-04 17:08:20'),(88,80,'业务分组','Shop','cmdb:group',2,'cmdb/group',2,2,'2025-07-16 15:17:14'),(89,88,'创建分组','','cmdb:group:add',3,'',2,1,'2025-07-18 15:24:31'),(90,88,'修改分组','','cmdb:group:update',3,'',2,2,'2025-07-18 15:25:49'),(91,88,'删除分组','','cmdb:group:delete',3,'',2,3,'2025-07-18 15:26:21'),(93,81,'工作负载','Star','cloud:k8s:workload',2,'k8s/workload',2,4,'2025-07-24 14:38:31'),(95,80,'数据管理','Coin','cmdb:db',2,'cmdb/db',2,3,'2025-07-29 15:27:50'),(96,44,'数据日志','Coin','monitor:dblog:list',2,'monitor/dblog',2,3,'2025-07-31 12:44:07'),(97,0,'任务中心','User','',1,'',2,5,'2025-08-06 13:33:47'),(98,97,'任务模版','connection','task:template',2,'task/template',2,2,'2025-08-06 13:35:19'),(99,97,'任务作业','key','task:job',2,'task/job',2,1,'2025-08-06 13:36:06'),(100,97,'Ansible任务','Eleme','task:ansible',2,'task/ansible',2,3,'2025-08-23 18:35:24'),(101,0,'运维工具','Search','',1,'',2,6,'2025-08-29 10:59:35'),(102,101,'agent列表','price-tag','ops:agent',2,'ops/agent',2,2,'2025-08-29 11:22:20'),(103,101,'工具列表','present','ops:tools',2,'ops/tools',2,1,'2025-08-29 11:29:02'),(104,84,'密钥管理','Phone','config:keymanage:key',2,'config/keymanage',2,3,'2025-09-08 13:24:40'),(105,81,'命名空间','discount','k8s:namespace',2,'k8s/namespace',2,3,'2025-09-11 15:02:14'),(106,81,'网络管理','guide','k8s:network',2,'k8s/network',2,5,'2025-09-11 15:04:14'),(107,81,'配置管理','connection','k8s:config',2,'k8s/config',2,7,'2025-09-11 15:04:52'),(108,81,'存储管理','Coin','k8s:storage',2,'k8s/storage',2,6,'2025-09-11 15:05:40'),(109,0,'服务管理','ElemeFilled','',1,'',2,4,'2025-09-16 09:49:55'),(110,109,'应用列表','Menu','app:application',2,'app/application',2,1,'2025-09-16 09:52:58'),(111,109,'快速发布','TrendCharts','app:quick-release',2,'app/quick-release',2,2,'2025-09-16 17:12:11'),(113,45,'批量删除','','monitor:operator:delete',3,'',2,2,'2025-09-17 20:55:13'),(114,104,'修改密钥','','config:keymanage:edit',3,'',2,1,'2025-09-18 10:45:57'),(115,104,'删除密钥','','config:keymanage:delete',3,'',2,2,'2025-09-18 10:53:44'),(117,104,'同步主机','','config:keymanage:rsync',3,'',2,3,'2025-09-18 10:57:25'),(118,104,'创建密钥','','config:keymanage:create',3,'',2,4,'2025-09-18 11:01:12'),(119,86,'修改账号','','config:common:edit',3,'',2,2,'2025-09-18 11:47:33'),(120,86,'删除账号','','config:common:delete',3,'',2,3,'2025-09-18 11:48:17'),(121,86,'解密账号','','config:common:decrypt',3,'',2,4,'2025-09-18 11:48:57'),(122,86,'创建账号','','config:common:add',3,'',2,1,'2025-09-18 11:49:30'),(123,85,'修改凭据','','config:ecs:edit',3,'',2,1,'2025-09-18 11:54:16'),(124,85,'删除凭据','','config:ecs:delete',3,'',2,2,'2025-09-18 11:54:51'),(125,85,'创建凭据','','config:ecs:create',3,'',2,3,'2025-09-18 11:55:21'),(126,102,'卸载agent','','ops:agent:delete',3,'',2,1,'2025-09-18 12:47:48'),(127,102,'查看agent','','ops:agent:get',3,'',2,2,'2025-09-18 12:49:02'),(128,102,'部署agent','','ops:agent:create',3,'',2,3,'2025-09-18 12:49:56'),(129,102,'批量卸载agent','','ops:agent:deleteall',3,'',2,4,'2025-09-18 12:50:52'),(130,100,'启动ansible任务流程','','task:ansible:start',3,'',2,1,'2025-09-18 12:59:30'),(131,100,'删除ansible任务','','task:ansible:delete',3,'',2,2,'2025-09-18 13:00:03'),(132,100,'新增ansible任务','','task:ansible:create',3,'',2,3,'2025-09-18 13:00:45'),(133,98,'新增模版','','task:template:add',3,'',2,1,'2025-09-18 13:16:38'),(134,98,'修改模版','','task:template:edit',3,'',2,2,'2025-09-18 13:17:04'),(135,98,'删除模版','','task:template:delete',3,'',2,3,'2025-09-18 13:18:25'),(136,99,'新增任务','','task:job:add',3,'',2,1,'2025-09-18 13:24:19'),(137,99,'启动任务','','task:job:start',3,'',2,2,'2025-09-18 13:24:59'),(138,99,'删除任务','','task:job:delete',3,'',2,3,'2025-09-18 13:25:41'),(139,111,'新建发布','','app:quick-release:add',3,'',2,1,'2025-09-18 13:30:53'),(140,111,'启动发布','','app:quick-release:start',3,'',2,2,'2025-09-18 13:32:11'),(141,111,'删除发布','','app:quick-release:delete',3,'',2,3,'2025-09-18 13:32:32'),(142,110,'创建应用','','app:application:add',3,'',2,1,'2025-09-18 14:28:07'),(143,110,'修改应用','','app:application:edit',3,'',2,2,'2025-09-18 14:28:59'),(144,110,'配置应用环境','','app:application:env',3,'',2,3,'2025-09-18 14:29:34'),(145,110,'删除应用','','app:application:delete',3,'',2,4,'2025-09-18 14:30:11'),(146,95,'创建数据库账号','','cmdb:db:add',3,'',2,1,'2025-09-18 14:41:32'),(147,95,'修改数据库配置','','cmdb:db:edit',3,'',2,2,'2025-09-18 14:42:47'),(148,95,'删除数据库账号','','cmdb:db:delete',3,'',2,3,'2025-09-18 14:43:57'),(149,78,'创建主机','','cmdb:ecs:add',3,'',2,1,'2025-09-18 14:47:42'),(150,78,'主机终端','','cmdb:ecs:terminal',3,'',2,2,'2025-09-18 14:48:36'),(151,78,'修改主机信息','','cmdb:ecs:edit',3,'',2,3,'2025-09-18 14:49:43'),(152,78,'上传文件到主机','','cmdb:ecs:upload',3,'',2,4,'2025-09-18 14:50:38'),(153,78,'执行主机命令','','cmdb:ecs:shell',3,'',2,5,'2025-09-18 14:51:10'),(154,78,'监控主机','','cmdb:ecs:monitor',3,'',2,6,'2025-09-18 14:51:52'),(155,78,'删除主机','','cmdb:ecs:delete',3,'',2,7,'2025-09-18 14:52:20'),(156,99,'启动脚本','','task:job:jobstart',3,'',2,4,'2025-09-18 18:36:38'),(157,99,'停止脚本','','task:job:jobstop',3,'',2,5,'2025-09-18 18:39:23'),(159,100,'开始ansible任务作业','','task:ansible:jobstart',3,'',2,4,'2025-09-18 18:43:40'),(160,111,'启动jenkins任务','','app:quick-release:jobstart',3,'',2,4,'2025-09-18 18:47:39'),(161,111,'停止jenkins任务','','app:quick-release:jobstop',3,'',2,5,'2025-09-18 18:48:16'),(162,110,'新增环境','','app:application:envadd',3,'',2,5,'2025-09-18 21:02:28'),(163,110,'修改环境','','app:application:envedit',3,'',2,6,'2025-09-18 21:03:08'),(164,110,'删除环境','','app:application:envdelete',3,'',2,7,'2025-09-18 21:04:43'),(165,78,'连接主机终端','','cmdb:ecs:connecthost',3,'',2,8,'2025-09-18 21:11:43'),(166,78,'同步主机信息','','cmdb:ecs:rsync',3,'',2,9,'2025-09-19 21:35:06'),(167,82,'注册集群','','cloud:k8s:register',3,'',2,1,'2025-09-19 21:57:54'),(168,82,'创建集群','','cloud:k8s:add',3,'',2,2,'2025-09-19 21:58:19'),(169,82,'修改集群','','cloud:k8s:edit',3,'',2,3,'2025-09-19 21:59:06'),(170,82,'同步集群','','cloud:k8s:rsync',3,'',2,4,'2025-09-19 21:59:31'),(171,82,'删除集群','','cloud:k8s:delete',3,'',2,5,'2025-09-19 21:59:56'),(172,83,'查看监控仪表盘','','k8s:node:monitor',3,'',2,1,'2025-09-20 00:19:49'),(173,83,'查看节点资源详情','','k8s:node:details',3,'',2,2,'2025-09-20 00:21:20'),(174,83,'配置节点污点','','k8s:node:stain',3,'',2,3,'2025-09-20 00:22:17'),(175,83,'增加标签','','k8s:node:label',3,'',2,4,'2025-09-20 00:23:15'),(176,83,'封锁节点','','k8s:node:close',3,'',2,5,'2025-09-20 00:24:13'),(177,83,'驱逐节点','','k8s:node:expel',3,'',2,6,'2025-09-20 00:25:04'),(178,105,'创建命名空间','','k8s:namespace:add',3,'',2,1,'2025-09-20 00:36:14'),(179,105,'查看命名空间详情','','k8s:namespace:details',3,'',2,2,'2025-09-20 00:37:22'),(180,105,'查看命名空间资源配置','','k8s:namespace:setup',3,'',2,3,'2025-09-20 00:39:05'),(181,105,'添加命名空间资源配置','','k8s:namespace:setupadd',3,'',2,4,'2025-09-20 00:40:06'),(182,105,'查看限制命名空间资源','','k8s:namespace:restriction',3,'',2,5,'2025-09-20 00:41:25'),(183,105,'添加限制命名空间资源','','k8s:namespace:restrictionadd',3,'',2,6,'2025-09-20 00:42:23'),(184,105,'删除命名空间','','k8s:namespace:delete',3,'',2,7,'2025-09-20 00:43:03'),(185,93,'新增工作负载','','k8s:workload:add',3,'',2,1,'2025-09-20 01:05:08'),(186,93,'伸缩pod','','k8s:workload:expandable',3,'',2,2,'2025-09-20 01:06:18'),(187,93,'重启pod','','k8s:workload:restart',3,'',2,3,'2025-09-20 01:07:13'),(188,93,'更新pod资源限制','','k8s:workload:resource',3,'',2,4,'2025-09-20 01:08:39'),(189,93,'更新pod资调度','','k8s:workload:dispatch',3,'',2,5,'2025-09-20 01:09:36'),(190,93,'更新yaml文件','','k8s:workload:edityaml',3,'',2,6,'2025-09-20 01:10:57'),(191,93,'删除工作负载','','k8s:workload:delete',3,'',2,7,'2025-09-20 01:11:52'),(192,93,'回滚工作负载版本','','k8s:workload:rollback_version',3,'',2,8,'2025-09-20 01:39:38'),(193,93,'查看pod日志','','k8s:workload:podlog',3,'',2,9,'2025-09-20 01:44:37'),(194,93,'删除pod','','k8s:workload:poddelete',3,'',2,10,'2025-09-20 01:45:16'),(195,93,'登陆pod终端','','k8s:workload:terminal',3,'',2,11,'2025-09-20 01:46:07'),(196,93,'编辑pod yaml文件','','k8s:workload:edityaml',3,'',2,12,'2025-09-20 01:47:22'),(197,106,'新增service','','k8s:network:addservice',3,'',2,1,'2025-09-20 02:14:21'),(198,106,'编辑 Service','','k8s:network:editservice',3,'',2,2,'2025-09-20 02:15:19'),(199,106,'编辑service YAML','','k8s:network:edit_service_yaml',3,'',2,3,'2025-09-20 02:16:15'),(200,106,'查看Service 事件','','k8s:network:service_event',3,'',2,4,'2025-09-20 02:18:11'),(201,106,'删除Service','','k8s:network:deleteservice',3,'',2,5,'2025-09-20 02:18:59'),(202,106,'编辑ingress','','k8s:network:editingress',3,'',2,6,'2025-09-20 02:26:59'),(203,106,'新增ingress','','k8s:network:addingress',3,'',2,7,'2025-09-20 02:27:29'),(204,106,'编辑ingress_yaml','','k8s:network:edit_ingress_yaml',3,'',2,8,'2025-09-20 02:28:23'),(205,106,'查看ingress 事件','','k8s:network:ingress_event',3,'',2,9,'2025-09-20 02:29:24'),(206,106,'删除ingress','','k8s:network:delete_ingress',3,'',2,10,'2025-09-20 02:30:04'),(212,0,'监控中心','Shop','',1,'',2,4,'2025-12-03 21:21:04'),(213,212,'域名监控','UploadFilled','monitor:domain',2,'monitor/domain',2,1,'2025-12-03 21:22:11'),(215,4,'系统配置','List','system:config',2,'system/config',2,6,'2025-12-09 11:03:54'),(216,212,'故障管理','Help','monitor:incident',2,'monitor/incident',2,2,'2025-12-10 15:10:29'),(217,95,'查看密码','','cmdb:db:passwd',3,'',2,4,'2025-12-13 14:27:22'),(218,95,'测试链接','','cmdb:db:test',3,'',2,5,'2025-12-13 14:28:38'),(219,95,'DBMS终端','','cmdb:db:dbms',3,'',2,6,'2025-12-13 14:32:59'),(220,95,'Redis终端','','cmdb:db:redis',3,'',2,7,'2025-12-13 14:33:27'),(221,95,'ES终端','','cmdb:db:es',3,'',2,8,'2025-12-13 14:33:53'),(222,95,'Mongo终端','','cmdb:db:mongo',3,'',2,9,'2025-12-13 14:34:14'),(229,213,'新增域名','','monitor:domain:add',3,'',2,1,'2025-12-13 15:22:47'),(230,213,'批量新增域名','','monitor:domain:add_all',3,'',2,2,'2025-12-13 15:23:44'),(231,213,'域名自动巡检','','monitor:domain:auto_inspection',3,'',2,3,'2025-12-13 15:25:35'),(232,213,'域名手动巡检','','monitor:domain:manual_inspection',3,'',2,4,'2025-12-13 15:26:55'),(233,213,'单域名手动巡检','','monitor:domain:ops_inspection',3,'',2,5,'2025-12-13 15:36:47'),(234,213,'编辑域名','','monitor:domain:edit',3,'',2,6,'2025-12-13 15:38:30'),(235,213,'删除域名','','monitor:domain:delete',3,'',2,7,'2025-12-13 15:38:54'),(236,216,'新增故障记录','','monitor:incident:add',3,'',2,1,'2025-12-13 15:55:41'),(237,216,'编辑故障记录','','monitor:incident:edit',3,'',2,2,'2025-12-13 15:57:20'),(238,216,'编辑故障状态','','monitor:incident:status',3,'',2,3,'2025-12-13 15:57:39'),(239,216,'删除故障记录','','monitor:incident:delete',3,'',2,4,'2025-12-13 15:58:12'),(240,216,'访问故障链接','','monitor:incident:url',3,'',2,5,'2025-12-13 16:02:46'),(241,78,'下载文件','','cmdb:ecs:download',3,'',2,10,'2025-12-13 16:13:17'),(243,80,'物理机管理','Platform','cmdb:physical:list',2,'cmdb/physical',2,4,'2026-05-30 12:46:25'),(244,243,'创建物理机','','cmdb:physical:add',3,'',2,1,'2026-05-30 13:01:16');
/*!40000 ALTER TABLE `sys_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_operation_log`
--

DROP TABLE IF EXISTS `sys_operation_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_operation_log` (
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
) ENGINE=InnoDB AUTO_INCREMENT=1384 DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='操作日志记录表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_operation_log`
--

LOCK TABLES `sys_operation_log` WRITE;
/*!40000 ALTER TABLE `sys_operation_log` DISABLE KEYS */;
INSERT INTO `sys_operation_log` VALUES (1,89,'admin','delete','192.168.3.7','/api/v1/sysOperationLog/clean','2025-12-14 14:57:14.270','清空操作日志'),(2,89,'admin','post','172.20.0.1','/api/v1/config/ecsauthadd','2026-01-19 13:39:56.951','新增ECS认证'),(3,89,'admin','post','172.20.0.1','/api/v1/cmdb/hostcreate','2026-01-19 13:40:22.661','创建主机'),(4,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 13:46:52.871','创建Ansible任务'),(5,89,'admin','post','172.20.0.1','/api/v1/task/ansible/52/start','2026-01-19 13:47:18.491','启动Ansible任务'),(6,89,'admin','post','172.20.0.1','/api/v1/task/ansible/52/start','2026-01-19 13:47:21.697','启动Ansible任务'),(7,89,'admin','post','172.20.0.1','/api/v1/task/ansible/52/start','2026-01-19 13:47:22.572','启动Ansible任务'),(8,89,'admin','post','172.20.0.1','/api/v1/task/ansible/52/start','2026-01-19 13:47:30.703','启动Ansible任务'),(9,89,'admin','post','172.20.0.1','/api/v1/task/ansible/52/start','2026-01-19 13:47:33.028','启动Ansible任务'),(10,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/52','2026-01-19 13:47:39.221','删除Ansible任务'),(11,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 13:48:34.346','创建Ansible任务'),(12,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 13:50:42.553','创建Ansible任务'),(13,89,'admin','post','172.20.0.1','/api/v1/task/ansible/53/start','2026-01-19 13:51:23.443','启动Ansible任务'),(14,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/53','2026-01-19 13:54:37.289','删除Ansible任务'),(15,89,'admin','post','172.20.0.1','/api/v1/task/add','2026-01-19 13:55:18.047','新增任务'),(16,89,'admin','post','172.20.0.1','/api/v1/taskjob/start','2026-01-19 13:55:28.269','启动任务作业'),(17,89,'admin','post','172.20.0.1','/api/v1/taskjob/stop','2026-01-19 13:55:53.198','停止任务作业'),(18,89,'admin','post','172.20.0.1','/api/v1/taskjob/stop','2026-01-19 13:55:53.246','停止任务作业'),(19,89,'admin','post','172.20.0.1','/api/v1/monitor/domain/check/28','2026-01-19 13:57:03.183','检查单个域名'),(20,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-19 13:59:24.051','查询K8s'),(21,89,'admin','post','172.20.0.1','/api/v1/db/instances','2026-01-19 14:52:57.491','操作数据库实例'),(22,89,'admin','put','172.20.0.1','/api/v1/db/instances','2026-01-19 14:53:13.059','操作数据库实例'),(23,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-19 14:53:49.880','执行SQL'),(24,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-19 14:54:30.847','执行SQL'),(25,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-19 14:54:33.243','执行SQL'),(26,89,'admin','post','172.20.0.1','/api/v1/db/pre-check-sql','2026-01-19 14:54:44.249','预检查SQL'),(27,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-19 14:54:55.888','执行SQL'),(28,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-19 14:56:14.070','执行SQL'),(29,89,'admin','delete','172.20.0.1','/api/v1/task/delete','2026-01-19 15:11:33.820','删除任务'),(30,89,'admin','post','172.20.0.1','/api/v1/cmdb/hostsync','2026-01-19 16:10:43.844','同步主机信息'),(31,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 16:59:59.526','创建Ansible任务'),(32,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 17:00:17.692','创建Ansible任务'),(33,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 17:00:23.532','创建Ansible任务'),(34,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 17:00:26.144','创建Ansible任务'),(35,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-19 17:01:01.341','创建Ansible任务'),(36,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/55','2026-01-19 17:01:05.679','删除Ansible任务'),(37,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/54','2026-01-19 17:01:07.296','删除Ansible任务'),(38,89,'admin','post','172.20.0.1','/api/v1/task/ansible/56/start','2026-01-19 17:01:47.294','启动Ansible任务'),(39,89,'admin','post','172.20.0.1','/api/v1/task/ansible/56/start','2026-01-19 17:02:49.440','启动Ansible任务'),(40,89,'admin','delete','172.20.0.1','/api/v1/tool/deploy/12','2026-01-20 10:31:15.776','部署'),(41,89,'admin','post','172.20.0.1','/api/v1/monitor/domain','2026-01-20 10:32:39.971','创建域名监控'),(42,89,'admin','post','172.20.0.1','/api/v1/monitor/domain/check-all','2026-01-20 10:33:27.794','检查所有域名'),(43,89,'admin','post','172.20.0.1','/api/v1/monitor/domain/check/41','2026-01-20 10:47:23.850','检查单个域名'),(44,89,'admin','post','172.20.0.1','/api/v1/monitor/domain/check/7','2026-01-20 10:47:46.803','检查单个域名'),(45,89,'admin','post','172.20.0.1','/api/v1/task/add','2026-01-20 14:09:03.019','新增任务'),(46,89,'admin','post','172.20.0.1','/api/v1/task/ansible/56/start','2026-01-20 16:04:49.446','启动Ansible任务'),(47,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-20 17:50:24.875','执行SQL'),(48,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-20 17:50:29.124','执行SQL'),(49,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-20 17:50:29.733','执行SQL'),(50,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-20 17:50:30.862','执行SQL'),(51,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-20 17:50:31.346','执行SQL'),(52,89,'admin','post','172.20.0.1','/api/v1/db/databases/exec-sql','2026-01-20 17:50:31.948','执行SQL'),(53,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-23 09:11:10.281','查询K8s'),(54,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-23 09:13:52.339','查询K8s'),(55,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-23 09:14:23.846','查询K8s'),(56,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-23 09:31:35.973','查询K8s'),(57,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-23 09:31:46.527','查询K8s'),(58,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-23 09:31:50.616','查询K8s'),(59,89,'admin','post','172.20.0.1','/api/v1/k8s/cluster','2026-01-23 09:32:18.541','查询K8s'),(60,89,'admin','delete','172.20.0.1','/api/v1/k8s/cluster/34','2026-01-23 09:39:47.474','操作K8s集群'),(61,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-23 14:16:10.011','创建Ansible任务'),(62,89,'admin','post','172.20.0.1','/api/v1/task/ansible/57/start','2026-01-23 14:16:27.113','启动Ansible任务'),(63,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/57','2026-01-23 14:17:29.633','删除Ansible任务'),(64,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 02:52:38.324','创建Ansible任务'),(65,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 02:52:50.446','创建Ansible任务'),(66,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 02:52:54.719','创建Ansible任务'),(67,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 02:53:07.002','创建Ansible任务'),(68,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 02:53:07.866','创建Ansible任务'),(69,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/58','2026-01-24 02:53:19.281','删除Ansible任务'),(70,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/59','2026-01-24 02:53:20.980','删除Ansible任务'),(71,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 02:53:43.505','创建Ansible任务'),(72,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 02:53:48.997','创建Ansible任务'),(73,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 02:55:20.411','创建Ansible任务'),(74,89,'admin','post','192.168.1.223','/api/v1/task/ansible/61/start','2026-01-24 02:58:50.804','启动Ansible任务'),(75,89,'admin','post','172.20.0.1','/api/v1/task/ansible/60/start','2026-01-24 03:05:05.606','启动Ansible任务'),(76,89,'admin','post','172.20.0.1','/api/v1/task/ansible/60/start','2026-01-24 03:09:29.755','启动Ansible任务'),(77,89,'admin','post','172.20.0.1','/api/v1/task/ansible/61/start','2026-01-24 03:11:18.280','启动Ansible任务'),(78,89,'admin','post','172.20.0.1','/api/v1/task/ansible/60/start','2026-01-24 03:11:36.284','启动Ansible任务'),(79,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/61','2026-01-24 03:11:55.293','删除Ansible任务'),(80,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/60','2026-01-24 03:11:58.193','删除Ansible任务'),(81,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 03:12:23.693','创建Ansible任务'),(82,89,'admin','post','172.20.0.1','/api/v1/task/ansible/62/start','2026-01-24 03:12:27.735','启动Ansible任务'),(83,89,'admin','post','172.20.0.1','/api/v1/task/ansible/56/start','2026-01-24 03:14:33.086','启动Ansible任务'),(84,89,'admin','post','172.20.0.1','/api/v1/task/ansible/56/start','2026-01-24 03:14:58.116','启动Ansible任务'),(85,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 03:22:27.054','创建Ansible任务'),(86,89,'admin','post','192.168.1.223','/api/v1/task/ansible/63/start','2026-01-24 03:23:10.725','启动Ansible任务'),(87,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/63','2026-01-24 03:25:03.799','删除Ansible任务'),(88,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 03:39:30.120','创建Ansible任务'),(89,89,'admin','post','192.168.1.223','/api/v1/task/ansible/64/start','2026-01-24 03:39:45.603','启动Ansible任务'),(90,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/64','2026-01-24 03:50:36.794','删除Ansible任务'),(91,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 03:50:57.348','创建Ansible任务'),(92,89,'admin','post','192.168.1.223','/api/v1/task/ansible/65/start','2026-01-24 03:51:30.725','启动Ansible任务'),(93,89,'admin','post','172.20.0.1','/api/v1/task/ansible/65/start','2026-01-24 03:54:05.317','启动Ansible任务'),(94,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/65','2026-01-24 03:58:47.212','删除Ansible任务'),(95,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 04:11:32.529','创建Ansible任务'),(96,89,'admin','post','192.168.1.223','/api/v1/task/ansible/66/start','2026-01-24 04:11:48.332','启动Ansible任务'),(97,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/66','2026-01-24 04:16:55.589','删除Ansible任务'),(98,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 04:17:43.244','创建Ansible任务'),(99,89,'admin','post','172.20.0.1','/api/v1/task/ansible/67/start','2026-01-24 04:18:06.689','启动Ansible任务'),(100,89,'admin','post','172.20.0.1','/api/v1/task/ansible/67/start','2026-01-24 04:19:05.652','启动Ansible任务'),(101,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/67','2026-01-24 04:19:26.668','删除Ansible任务'),(102,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 04:19:46.327','创建Ansible任务'),(103,89,'admin','post','172.20.0.1','/api/v1/task/ansible/68/start','2026-01-24 04:20:00.848','启动Ansible任务'),(104,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/68','2026-01-24 04:21:38.974','删除Ansible任务'),(105,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 04:21:59.079','创建Ansible任务'),(106,89,'admin','post','172.20.0.1','/api/v1/task/ansible/69/start','2026-01-24 04:22:14.475','启动Ansible任务'),(107,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/69','2026-01-24 04:23:36.246','删除Ansible任务'),(108,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 04:24:29.449','创建Ansible任务'),(109,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 04:24:46.852','创建Ansible任务'),(110,89,'admin','post','172.20.0.1','/api/v1/task/ansible/70/start','2026-01-24 04:25:19.631','启动Ansible任务'),(111,89,'admin','post','172.20.0.1','/api/v1/task/ansible/62/start','2026-01-24 04:25:42.179','启动Ansible任务'),(112,89,'admin','post','172.20.0.1','/api/v1/task/ansible/70/start','2026-01-24 15:13:14.554','启动Ansible任务'),(113,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/70','2026-01-24 15:13:29.210','删除Ansible任务'),(114,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 15:14:56.330','创建Ansible任务'),(115,89,'admin','post','172.20.0.1','/api/v1/task/ansible/71/start','2026-01-24 15:29:48.295','启动Ansible任务'),(116,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/71','2026-01-24 15:29:59.233','删除Ansible任务'),(117,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 15:32:07.102','创建Ansible任务'),(118,89,'admin','post','192.168.1.223','/api/v1/task/ansible/72/start','2026-01-24 15:32:24.775','启动Ansible任务'),(119,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/72','2026-01-24 15:34:38.697','删除Ansible任务'),(120,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 15:35:11.605','创建Ansible任务'),(121,89,'admin','post','172.20.0.1','/api/v1/task/ansible/73/start','2026-01-24 15:35:26.982','启动Ansible任务'),(122,89,'admin','post','192.168.1.223','/api/v1/task/ansible/73/start','2026-01-24 15:36:21.817','启动Ansible任务'),(123,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/73','2026-01-24 15:37:59.688','删除Ansible任务'),(124,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 15:38:22.895','创建Ansible任务'),(125,89,'admin','post','192.168.1.223','/api/v1/task/ansible/74/start','2026-01-24 15:38:36.052','启动Ansible任务'),(126,89,'admin','post','192.168.1.223','/api/v1/task/ansible/74/start','2026-01-24 16:08:25.474','启动Ansible任务'),(127,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/74','2026-01-24 16:33:42.771','删除Ansible任务'),(128,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 16:39:37.335','创建Ansible任务'),(129,89,'admin','post','192.168.1.223','/api/v1/task/ansible/75/start','2026-01-24 16:44:50.939','启动Ansible任务'),(130,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/75','2026-01-24 16:47:53.046','删除Ansible任务'),(131,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 16:49:49.779','创建Ansible任务'),(132,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 16:51:25.604','创建Ansible任务'),(133,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 16:52:08.796','创建Ansible任务'),(134,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/76','2026-01-24 16:52:40.535','删除Ansible任务'),(135,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 16:52:57.027','创建Ansible任务'),(136,89,'admin','post','192.168.1.223','/api/v1/task/ansible/77/start','2026-01-24 16:53:49.518','启动Ansible任务'),(137,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/77','2026-01-24 18:10:52.429','删除Ansible任务'),(138,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 18:11:25.498','创建Ansible任务'),(139,89,'admin','post','192.168.1.223','/api/v1/task/ansible/78/start','2026-01-24 18:11:48.703','启动Ansible任务'),(140,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/78','2026-01-24 18:25:20.698','删除Ansible任务'),(141,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 18:25:35.310','创建Ansible任务'),(142,89,'admin','post','192.168.1.223','/api/v1/task/ansible/79/start','2026-01-24 18:25:48.916','启动Ansible任务'),(143,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/79','2026-01-24 20:33:53.082','删除Ansible任务'),(144,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 20:36:39.891','创建Ansible任务'),(145,89,'admin','post','192.168.1.223','/api/v1/task/ansible/80/start','2026-01-24 20:36:50.162','启动Ansible任务'),(146,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/80','2026-01-24 20:44:55.043','删除Ansible任务'),(147,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 20:45:21.811','创建Ansible任务'),(148,89,'admin','post','192.168.1.223','/api/v1/task/ansible/81/start','2026-01-24 20:45:37.136','启动Ansible任务'),(149,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/81','2026-01-24 20:53:48.049','删除Ansible任务'),(150,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 20:59:06.412','创建Ansible任务'),(151,89,'admin','post','192.168.1.223','/api/v1/task/ansible/82/start','2026-01-24 20:59:33.426','启动Ansible任务'),(152,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/82','2026-01-24 21:04:00.769','删除Ansible任务'),(153,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:04:18.977','创建Ansible任务'),(154,89,'admin','post','192.168.1.223','/api/v1/task/ansible/83/start','2026-01-24 21:04:29.380','启动Ansible任务'),(155,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/83','2026-01-24 21:05:32.922','删除Ansible任务'),(156,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:07:56.554','创建Ansible任务'),(157,89,'admin','post','192.168.1.223','/api/v1/task/ansible/84/start','2026-01-24 21:08:05.969','启动Ansible任务'),(158,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/84','2026-01-24 21:11:55.837','删除Ansible任务'),(159,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:12:14.138','创建Ansible任务'),(160,89,'admin','post','192.168.1.223','/api/v1/task/ansible/84/start','2026-01-24 21:12:29.029','启动Ansible任务'),(161,89,'admin','post','192.168.1.223','/api/v1/task/ansible/85/start','2026-01-24 21:12:51.938','启动Ansible任务'),(162,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/85','2026-01-24 21:15:43.827','删除Ansible任务'),(163,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:16:03.713','创建Ansible任务'),(164,89,'admin','post','192.168.1.223','/api/v1/task/ansible/86/start','2026-01-24 21:16:14.259','启动Ansible任务'),(165,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/86','2026-01-24 21:39:39.247','删除Ansible任务'),(166,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:40:40.103','创建Ansible任务'),(167,89,'admin','post','192.168.1.223','/api/v1/task/ansible/87/start','2026-01-24 21:40:52.489','启动Ansible任务'),(168,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/87','2026-01-24 21:51:36.844','删除Ansible任务'),(169,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:52:47.789','创建Ansible任务'),(170,89,'admin','post','192.168.1.223','/api/v1/task/ansible/88/start','2026-01-24 21:53:09.415','启动Ansible任务'),(171,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/88','2026-01-24 21:55:09.600','删除Ansible任务'),(172,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:57:00.253','创建Ansible任务'),(173,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:57:24.693','创建Ansible任务'),(174,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 21:57:50.214','创建Ansible任务'),(175,89,'admin','post','192.168.1.223','/api/v1/task/ansible/89/start','2026-01-24 21:58:17.876','启动Ansible任务'),(176,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/89','2026-01-24 21:59:46.135','删除Ansible任务'),(177,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:00:04.894','创建Ansible任务'),(178,89,'admin','post','192.168.1.223','/api/v1/task/ansible/90/start','2026-01-24 22:00:22.143','启动Ansible任务'),(179,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/90','2026-01-24 22:01:24.844','删除Ansible任务'),(180,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:04:55.913','创建Ansible任务'),(181,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:05:26.855','创建Ansible任务'),(182,89,'admin','post','192.168.1.223','/api/v1/task/ansible/91/start','2026-01-24 22:05:52.521','启动Ansible任务'),(183,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/91','2026-01-24 22:09:08.508','删除Ansible任务'),(184,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:09:27.638','创建Ansible任务'),(185,89,'admin','post','192.168.1.223','/api/v1/task/ansible/92/start','2026-01-24 22:09:37.720','启动Ansible任务'),(186,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/92','2026-01-24 22:13:45.992','删除Ansible任务'),(187,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:14:40.563','创建Ansible任务'),(188,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:14:51.137','创建Ansible任务'),(189,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:15:04.617','创建Ansible任务'),(190,89,'admin','post','192.168.1.223','/api/v1/task/ansible/93/start','2026-01-24 22:15:18.888','启动Ansible任务'),(191,89,'admin','post','192.168.1.223','/api/v1/task/ansible/93/start','2026-01-24 22:26:15.581','启动Ansible任务'),(192,89,'admin','post','172.20.0.1','/api/v1/task/ansible','2026-01-24 22:28:50.156','创建Ansible任务'),(193,89,'admin','post','172.20.0.1','/api/v1/task/ansible/94/start','2026-01-24 22:28:57.476','启动Ansible任务'),(194,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/94','2026-01-24 22:29:49.184','删除Ansible任务'),(195,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/93','2026-01-24 22:40:43.253','删除Ansible任务'),(196,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:46:49.574','创建Ansible任务'),(197,89,'admin','post','192.168.1.223','/api/v1/task/ansible/95/start','2026-01-24 22:46:59.789','启动Ansible任务'),(198,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/95','2026-01-24 22:49:22.465','删除Ansible任务'),(199,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:49:59.157','创建Ansible任务'),(200,89,'admin','post','192.168.1.223','/api/v1/task/ansible/96/start','2026-01-24 22:50:25.035','启动Ansible任务'),(201,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/96','2026-01-24 22:54:10.486','删除Ansible任务'),(202,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-24 22:57:21.576','创建Ansible任务'),(203,89,'admin','post','192.168.1.223','/api/v1/task/ansible/97/start','2026-01-24 22:57:40.161','启动Ansible任务'),(204,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:43:14.883','创建Ansible任务'),(205,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/97','2026-01-25 01:43:36.120','删除Ansible任务'),(206,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:43:41.233','创建Ansible任务'),(207,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:44:20.293','创建Ansible任务'),(208,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:44:43.022','创建Ansible任务'),(209,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:46:08.414','创建Ansible任务'),(210,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:57:44.313','创建Ansible任务'),(211,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:58:09.469','创建Ansible任务'),(212,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:58:26.006','创建Ansible任务'),(213,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/98','2026-01-25 01:58:34.680','删除Ansible任务'),(214,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:58:37.729','创建Ansible任务'),(215,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/99','2026-01-25 01:58:46.195','删除Ansible任务'),(216,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 01:59:33.556','创建Ansible任务'),(217,89,'admin','post','192.168.1.223','/api/v1/task/ansible/100/start','2026-01-25 02:02:09.503','启动Ansible任务'),(218,89,'admin','post','192.168.1.223','/api/v1/task/ansible/100/start','2026-01-25 02:04:44.939','启动Ansible任务'),(219,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/100','2026-01-25 02:12:25.616','删除Ansible任务'),(220,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 02:12:46.628','创建Ansible任务'),(221,89,'admin','post','192.168.1.223','/api/v1/task/ansible/101/start','2026-01-25 02:12:58.107','启动Ansible任务'),(222,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/101','2026-01-25 02:38:55.171','删除Ansible任务'),(223,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 02:39:02.210','创建Ansible任务'),(224,89,'admin','post','192.168.1.223','/api/v1/task/ansible/102/start','2026-01-25 02:40:13.713','启动Ansible任务'),(225,89,'admin','post','192.168.1.223','/api/v1/task/ansible/102/start','2026-01-25 02:56:22.928','启动Ansible任务'),(226,89,'admin','delete','172.20.0.1','/api/v1/task/ansible/102','2026-01-25 03:04:57.153','删除Ansible任务'),(227,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 03:05:40.710','创建Ansible任务'),(228,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-25 03:06:43.018','启动Ansible任务'),(229,89,'admin','post','172.20.0.1','/api/v1/task/ansible/103/start','2026-01-25 03:19:18.272','启动Ansible任务'),(230,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-01-25 18:03:50.322','创建Ansible任务'),(231,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:22:57.229',''),(232,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:38:10.429',''),(233,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:39:16.650',''),(234,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:39:31.489',''),(235,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:40:45.669',''),(236,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:41:19.377',''),(237,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:42:21.669',''),(238,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:43:44.790',''),(239,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:44:08.779',''),(240,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:45:15.649',''),(241,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-01-25 19:46:45.283',''),(242,89,'admin','put','192.168.1.223','/api/v1/config/ansible/5','2026-01-25 19:55:02.464',''),(243,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/5','2026-01-25 19:55:29.475',''),(244,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-25 20:53:04.846','启动Ansible任务'),(245,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-25 21:25:02.045','启动Ansible任务'),(246,89,'admin','put','172.20.0.1','/api/v1/post/updateStatus','2026-01-26 15:11:57.046','修改岗位状态'),(247,89,'admin','put','172.20.0.1','/api/v1/menu/update','2026-01-26 15:13:16.637','修改菜单'),(248,89,'admin','put','172.20.0.1','/api/v1/menu/update','2026-01-26 15:14:04.833','修改菜单'),(249,89,'admin','post','192.168.65.94','/api/v1/task/ansible/103/start','2026-01-26 15:55:14.576','启动Ansible任务'),(250,89,'admin','post','192.168.65.94','/api/v1/task/ansible/103/start','2026-01-26 15:56:38.885','启动Ansible任务'),(251,89,'admin','post','192.168.65.94','/api/v1/task/ansible/103/start','2026-01-26 15:59:26.721','启动Ansible任务'),(252,89,'admin','post','192.168.65.94','/api/v1/task/ansible/103/start','2026-01-26 16:01:18.262','启动Ansible任务'),(253,89,'admin','post','192.168.65.94','/api/v1/task/ansible/103/start','2026-01-26 17:23:54.364','启动Ansible任务'),(254,89,'admin','post','172.20.0.1','/api/v1/taskjob/start','2026-01-26 17:30:20.134','启动任务作业'),(255,89,'admin','post','172.20.0.1','/api/v1/template/add','2026-01-26 17:36:09.725','新增任务模板'),(256,89,'admin','post','172.20.0.1','/api/v1/task/add','2026-01-26 17:36:29.877','新增任务'),(257,89,'admin','post','172.20.0.1','/api/v1/taskjob/start','2026-01-26 17:36:34.410','启动任务作业'),(258,89,'admin','put','172.20.0.1','/api/v1/template/update','2026-01-26 17:38:29.034','修改任务模板'),(259,89,'admin','delete','172.20.0.1','/api/v1/task/delete','2026-01-26 17:38:40.398','删除任务'),(260,89,'admin','post','172.20.0.1','/api/v1/task/add','2026-01-26 17:38:58.797','新增任务'),(261,89,'admin','post','172.20.0.1','/api/v1/taskjob/start','2026-01-26 17:39:02.719','启动任务作业'),(262,89,'admin','delete','172.20.0.1','/api/v1/task/delete','2026-01-26 17:40:11.315','删除任务'),(263,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-26 19:57:11.928','启动Ansible任务'),(264,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-27 21:54:59.348','启动Ansible任务'),(265,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-27 21:59:00.732','启动Ansible任务'),(266,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-27 21:59:31.211','启动Ansible任务'),(267,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-27 22:14:26.367','启动Ansible任务'),(268,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-27 22:15:08.292','启动Ansible任务'),(269,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-27 22:16:29.318','启动Ansible任务'),(270,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 00:25:59.906','启动Ansible任务'),(271,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 00:30:35.087','启动Ansible任务'),(272,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 01:04:28.125','启动Ansible任务'),(273,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 01:06:51.286','启动Ansible任务'),(274,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 01:10:39.775','启动Ansible任务'),(275,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 01:14:22.788','启动Ansible任务'),(276,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 20:49:01.720','启动Ansible任务'),(277,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 20:51:41.101','启动Ansible任务'),(278,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 20:58:09.839','启动Ansible任务'),(279,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 21:07:40.712','启动Ansible任务'),(280,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 21:08:43.607','启动Ansible任务'),(281,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 21:10:18.797','启动Ansible任务'),(282,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-28 23:22:28.679','启动Ansible任务'),(283,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/103/history/11','2026-01-29 00:38:24.426',''),(284,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-29 00:58:22.786','启动Ansible任务'),(285,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-29 00:59:34.981','启动Ansible任务'),(286,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-29 01:00:43.467','启动Ansible任务'),(287,89,'admin','post','172.20.0.1','/api/v1/task/add','2026-01-29 21:48:48.181','新增任务'),(288,89,'admin','post','172.20.0.1','/api/v1/task/add','2026-01-29 21:48:57.443','新增任务'),(289,89,'admin','delete','172.20.0.1','/api/v1/task/delete','2026-01-29 22:23:45.335','删除任务'),(290,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-30 00:35:39.675','启动Ansible任务'),(291,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 14:56:23.649','删除Ansible任务'),(292,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 14:57:27.341','删除Ansible任务'),(293,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 14:59:35.085','删除Ansible任务'),(294,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 14:59:45.812','删除Ansible任务'),(295,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 14:59:55.103','删除Ansible任务'),(296,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:00:17.012','删除Ansible任务'),(297,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:00:25.316','删除Ansible任务'),(298,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:14:42.784','删除Ansible任务'),(299,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:15.360','删除Ansible任务'),(300,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:23.393','删除Ansible任务'),(301,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:24.452','删除Ansible任务'),(302,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:25.273','删除Ansible任务'),(303,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:26.107','删除Ansible任务'),(304,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:26.753','删除Ansible任务'),(305,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:27.346','删除Ansible任务'),(306,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:27.951','删除Ansible任务'),(307,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:28.516','删除Ansible任务'),(308,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:15:29.082','删除Ansible任务'),(309,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:28:39.627','删除Ansible任务'),(310,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:29:06.858','删除Ansible任务'),(311,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:30:06.217','删除Ansible任务'),(312,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 15:30:12.569','删除Ansible任务'),(313,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 16:59:40.283','删除Ansible任务'),(314,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:16.480','删除Ansible任务'),(315,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:18.157','删除Ansible任务'),(316,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:19.231','删除Ansible任务'),(317,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:20.066','删除Ansible任务'),(318,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:20.924','删除Ansible任务'),(319,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:21.682','删除Ansible任务'),(320,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:22.517','删除Ansible任务'),(321,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:23.258','删除Ansible任务'),(322,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:23.995','删除Ansible任务'),(323,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:24.657','删除Ansible任务'),(324,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:25.253','删除Ansible任务'),(325,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:25.860','删除Ansible任务'),(326,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:26.485','删除Ansible任务'),(327,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:27.039','删除Ansible任务'),(328,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:27.638','删除Ansible任务'),(329,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:28.453','删除Ansible任务'),(330,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:29.387','删除Ansible任务'),(331,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:30.233','删除Ansible任务'),(332,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:31.123','删除Ansible任务'),(333,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:31.873','删除Ansible任务'),(334,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-01-31 17:00:33.234','删除Ansible任务'),(335,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-01-31 21:25:44.613','启动Ansible任务'),(336,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:18:01.735','删除Ansible任务'),(337,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:20:21.297','删除Ansible任务'),(338,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:23:11.399','删除Ansible任务'),(339,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:23:57.051','删除Ansible任务'),(340,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:24:28.177','删除Ansible任务'),(341,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:24:44.194','删除Ansible任务'),(342,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:24:44.574','删除Ansible任务'),(343,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:25:02.949','删除Ansible任务'),(344,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:25:25.900','删除Ansible任务'),(345,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:25:26.486','删除Ansible任务'),(346,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:25:28.633','删除Ansible任务'),(347,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:25:28.934','删除Ansible任务'),(348,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:25:31.740','删除Ansible任务'),(349,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-02-27 15:25:32.525','删除Ansible任务'),(350,89,'admin','post','192.168.65.123','/api/v1/task/ansible/103/start','2026-02-27 16:05:46.487','启动Ansible任务'),(351,89,'admin','post','192.168.65.123','/api/v1/task/ansible/103/start','2026-02-27 17:17:13.369','启动Ansible任务'),(352,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-02-27 20:49:19.991','启动Ansible任务'),(353,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 03:54:09.298',''),(354,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/undefined','2026-02-28 03:56:13.596',''),(355,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/undefined','2026-02-28 03:56:25.254',''),(356,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/undefined','2026-02-28 03:56:55.657',''),(357,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/undefined','2026-02-28 03:56:59.538',''),(358,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/undefined','2026-02-28 03:57:17.691',''),(359,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/6','2026-02-28 03:57:19.977',''),(360,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 03:59:28.057','删除Ansible任务'),(361,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-02-28 04:00:01.989','启动Ansible任务'),(362,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 04:04:19.540','删除Ansible任务'),(363,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 04:05:21.723','删除Ansible任务'),(364,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-02-28 04:05:24.308','启动Ansible任务'),(365,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 04:17:30.673','删除Ansible任务'),(366,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 04:17:38.397','删除Ansible任务'),(367,89,'admin','put','192.168.1.223','/api/v1/config/ansible/3','2026-02-28 04:34:13.235',''),(368,89,'admin','put','192.168.1.223','/api/v1/config/ansible/3','2026-02-28 04:34:20.818',''),(369,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-02-28 04:34:25.510','启动Ansible任务'),(370,89,'admin','put','192.168.1.223','/api/v1/config/ansible/3','2026-02-28 04:36:08.830',''),(371,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-02-28 04:36:12.622','启动Ansible任务'),(372,89,'admin','put','192.168.1.223','/api/v1/config/ansible/3','2026-02-28 04:38:25.595',''),(373,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-02-28 04:38:32.952','启动Ansible任务'),(374,89,'admin','post','192.168.1.223','/api/v1/task/ansible/104/start','2026-02-28 04:40:47.607','启动Ansible任务'),(375,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 14:38:27.029','删除Ansible任务'),(376,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 14:38:54.875','删除Ansible任务'),(377,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 14:39:13.178','删除Ansible任务'),(378,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 14:39:13.352','删除Ansible任务'),(379,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 14:39:27.543','删除Ansible任务'),(380,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 14:39:47.736','删除Ansible任务'),(381,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/104','2026-02-28 14:43:56.835','删除Ansible任务'),(382,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 14:45:10.059','创建Ansible任务'),(383,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 14:45:28.644','创建Ansible任务'),(384,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 14:46:05.564','创建Ansible任务'),(385,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 14:46:14.263','创建Ansible任务'),(386,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 15:25:49.063','创建Ansible任务'),(387,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 15:26:25.140','创建Ansible任务'),(388,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 15:26:50.139','创建Ansible任务'),(389,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 15:26:55.424','创建Ansible任务'),(390,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 16:39:26.025','创建Ansible任务'),(391,89,'admin','post','192.168.1.223','/api/v1/task/ansible/105/start','2026-02-28 16:39:44.885','启动Ansible任务'),(392,89,'admin','put','192.168.1.223','/api/v1/config/ansible/2','2026-02-28 16:40:53.978',''),(393,89,'admin','post','192.168.1.223','/api/v1/task/ansible/105/start','2026-02-28 16:41:00.057','启动Ansible任务'),(394,89,'admin','put','192.168.1.223','/api/v1/task/ansible/105','2026-02-28 16:46:22.544','删除Ansible任务'),(395,89,'admin','post','192.168.1.223','/api/v1/task/ansible/105/start','2026-02-28 16:46:59.772','启动Ansible任务'),(396,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 17:02:40.731','删除Ansible任务'),(397,89,'admin','put','192.168.1.223','/api/v1/task/ansible/103','2026-02-28 17:02:48.015','删除Ansible任务'),(398,89,'admin','put','192.168.1.223','/api/v1/task/ansible/undefined','2026-02-28 17:12:21.130',''),(399,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 17:14:37.736','创建Ansible任务'),(400,89,'admin','put','192.168.1.223','/api/v1/task/ansible/undefined','2026-02-28 17:16:18.186',''),(401,89,'admin','put','192.168.1.223','/api/v1/task/ansible/undefined','2026-02-28 17:16:32.999',''),(402,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/106','2026-02-28 17:16:36.290','删除Ansible任务'),(403,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 17:17:02.698','创建Ansible任务'),(404,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:18:04.497','删除Ansible任务'),(405,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:23:10.161','删除Ansible任务'),(406,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:26:07.361','删除Ansible任务'),(407,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:28:24.681','删除Ansible任务'),(408,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:31:41.090','删除Ansible任务'),(409,89,'admin','post','192.168.1.223','/api/v1/task/ansible/107/start','2026-02-28 17:44:19.206','启动Ansible任务'),(410,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:45:54.987','删除Ansible任务'),(411,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:49:50.359','删除Ansible任务'),(412,89,'admin','put','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:51:00.411','删除Ansible任务'),(413,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/107','2026-02-28 17:51:09.301','删除Ansible任务'),(414,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 17:51:28.936','创建Ansible任务'),(415,89,'admin','post','192.168.1.223','/api/v1/task/ansible/108/start','2026-02-28 17:51:33.578','启动Ansible任务'),(416,89,'admin','post','192.168.1.223','/api/v1/task/ansible/108/start','2026-02-28 17:54:40.479','启动Ansible任务'),(417,89,'admin','put','192.168.1.223','/api/v1/task/ansible/108','2026-02-28 17:54:49.375','删除Ansible任务'),(418,89,'admin','post','192.168.1.223','/api/v1/task/ansible/108/start','2026-02-28 17:54:51.762','启动Ansible任务'),(419,89,'admin','put','192.168.1.223','/api/v1/task/ansible/108','2026-02-28 17:54:57.202','删除Ansible任务'),(420,89,'admin','post','192.168.1.223','/api/v1/task/ansible/108/start','2026-02-28 17:55:00.092','启动Ansible任务'),(421,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/108','2026-02-28 17:55:03.254','删除Ansible任务'),(422,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 17:55:24.357','创建Ansible任务'),(423,89,'admin','put','192.168.1.223','/api/v1/task/ansible/109','2026-02-28 17:56:03.269','删除Ansible任务'),(424,89,'admin','put','192.168.1.223','/api/v1/task/ansible/109','2026-02-28 17:56:30.929','删除Ansible任务'),(425,89,'admin','post','192.168.1.223','/api/v1/task/ansible/109/start','2026-02-28 17:56:33.530','启动Ansible任务'),(426,89,'admin','put','192.168.1.223','/api/v1/task/ansible/109','2026-02-28 17:56:53.974','删除Ansible任务'),(427,89,'admin','post','192.168.1.223','/api/v1/task/ansible/109/start','2026-02-28 17:56:57.523','启动Ansible任务'),(428,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/109','2026-02-28 18:10:02.266','删除Ansible任务'),(429,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 18:10:11.165','创建Ansible任务'),(430,89,'admin','put','192.168.1.223','/api/v1/task/ansible/110','2026-02-28 18:10:23.172','删除Ansible任务'),(431,89,'admin','post','192.168.1.223','/api/v1/task/ansible/110/start','2026-02-28 18:10:29.311','启动Ansible任务'),(432,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/110','2026-02-28 18:10:34.030','删除Ansible任务'),(433,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 18:10:54.396','创建Ansible任务'),(434,89,'admin','delete','192.168.1.223','/api/v1/task/ansible/111','2026-02-28 18:11:08.164','删除Ansible任务'),(435,89,'admin','post','192.168.1.223','/api/v1/task/ansible','2026-02-28 18:11:27.270','创建Ansible任务'),(436,89,'admin','post','192.168.1.223','/api/v1/task/ansible/112/start','2026-02-28 18:11:45.290','启动Ansible任务'),(437,89,'admin','post','192.168.1.223','/api/v1/task/ansible/105/start','2026-02-28 18:13:04.017','启动Ansible任务'),(438,89,'admin','post','192.168.1.223','/api/v1/task/ansible/103/start','2026-02-28 18:13:13.431','启动Ansible任务'),(439,89,'admin','put','192.168.1.223','/api/v1/task/ansible/112','2026-02-28 18:15:25.791','删除Ansible任务'),(440,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:19.657',''),(441,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:25.645',''),(442,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:30.478',''),(443,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:35.831',''),(444,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:41.054',''),(445,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:45.121',''),(446,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:48.811',''),(447,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:53.060',''),(448,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:16:58.411',''),(449,89,'admin','post','192.168.1.223','/api/v1/config/ansible','2026-02-28 18:17:03.393',''),(450,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/7','2026-02-28 18:48:13.580',''),(451,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/16','2026-02-28 18:48:18.478',''),(452,89,'admin','delete','192.168.1.223','/api/v1/config/ansible/15','2026-02-28 18:48:21.111',''),(453,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-03-02 16:15:16.065','删除Ansible任务'),(454,89,'admin','post','192.168.65.123','/api/v1/task/ansible/103/start','2026-03-02 16:15:21.837','启动Ansible任务'),(455,89,'admin','put','192.168.65.123','/api/v1/task/ansible/103','2026-03-02 16:16:35.776','删除Ansible任务'),(456,89,'admin','post','192.168.65.123','/api/v1/task/ansible','2026-03-02 16:19:57.218','创建Ansible任务'),(457,89,'admin','put','192.168.65.123','/api/v1/task/ansible/113','2026-03-02 16:20:59.758','删除Ansible任务'),(458,89,'admin','delete','192.168.65.123','/api/v1/task/ansible/113','2026-03-02 16:21:12.976','删除Ansible任务'),(459,89,'admin','post','192.168.65.123','/api/v1/task/ansible','2026-03-02 16:21:39.253','创建Ansible任务'),(460,89,'admin','put','192.168.65.123','/api/v1/task/ansible/114','2026-03-02 16:23:07.090','删除Ansible任务'),(461,89,'admin','put','192.168.65.123','/api/v1/task/ansible/114','2026-03-02 16:23:39.723','删除Ansible任务'),(462,89,'admin','delete','192.168.65.123','/api/v1/task/ansible/114','2026-03-02 16:24:55.434','删除Ansible任务'),(463,89,'admin','post','192.168.65.123','/api/v1/task/ansible','2026-03-02 16:25:14.036','创建Ansible任务'),(464,89,'admin','post','192.168.65.123','/api/v1/config/ansible','2026-03-02 16:32:50.352',''),(465,89,'admin','put','192.168.65.123','/api/v1/config/ansible/3','2026-03-02 16:33:12.526',''),(466,89,'admin','put','192.168.65.123','/api/v1/task/ansible/112','2026-03-02 16:33:30.772','删除Ansible任务'),(467,89,'admin','post','192.168.65.123','/api/v1/task/ansible/112/start','2026-03-02 16:33:36.152','启动Ansible任务'),(468,89,'admin','put','192.168.65.123','/api/v1/task/ansible/112','2026-03-02 16:35:42.975','删除Ansible任务'),(469,89,'admin','post','192.168.65.123','/api/v1/task/ansible/112/start','2026-03-02 16:35:47.401','启动Ansible任务'),(470,89,'admin','delete','192.168.65.123','/api/v1/task/ansible/112','2026-03-02 16:36:33.798','删除Ansible任务'),(471,89,'admin','put','192.168.65.123','/api/v1/task/ansible/105','2026-03-02 16:36:50.177','删除Ansible任务'),(472,89,'admin','post','192.168.65.123','/api/v1/task/ansible/105/start','2026-03-02 16:36:54.573','启动Ansible任务'),(473,89,'admin','put','127.0.0.1','/api/v1/config/ansible/1','2026-03-06 00:30:14.486',''),(474,89,'admin','post','127.0.0.1','/api/v1/task/ansible/103/start','2026-03-06 00:30:21.742','启动Ansible任务'),(475,89,'admin','post','127.0.0.1','/api/v1/task/ansible/103/start','2026-03-06 00:33:31.733','启动Ansible任务'),(476,89,'admin','delete','127.0.0.1','/api/v1/task/ansible/103','2026-03-06 00:37:23.635','删除Ansible任务'),(477,89,'admin','post','127.0.0.1','/api/v1/task/ansible','2026-03-06 00:37:37.208','创建Ansible任务'),(478,89,'admin','post','127.0.0.1','/api/v1/task/ansible','2026-03-06 00:38:12.752','创建Ansible任务'),(479,89,'admin','post','127.0.0.1','/api/v1/task/ansible','2026-03-06 00:43:46.589','创建Ansible任务'),(480,89,'admin','delete','127.0.0.1','/api/v1/task/ansible/116','2026-03-06 00:45:03.606','删除Ansible任务'),(481,89,'admin','post','127.0.0.1','/api/v1/task/ansible','2026-03-06 00:45:23.494','创建Ansible任务'),(482,89,'admin','delete','127.0.0.1','/api/v1/task/ansible/117','2026-03-06 00:45:44.522','删除Ansible任务'),(483,89,'admin','post','127.0.0.1','/api/v1/task/ansible','2026-03-06 00:46:25.804','创建Ansible任务'),(484,89,'admin','post','127.0.0.1','/api/v1/task/ansible/118/start','2026-03-06 00:46:29.744','启动Ansible任务'),(485,89,'admin','delete','127.0.0.1','/api/v1/task/ansible/118','2026-03-08 00:08:39.800','删除Ansible任务'),(486,89,'admin','post','127.0.0.1','/api/v1/task/ansible','2026-03-08 00:12:24.065','创建Ansible任务'),(487,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 00:12:30.436','启动Ansible任务'),(488,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 00:21:25.559','启动Ansible任务'),(489,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 00:22:37.506','启动Ansible任务'),(490,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 00:22:56.462','启动Ansible任务'),(491,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 21:27:46.873','启动Ansible任务'),(492,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 21:31:35.776','启动Ansible任务'),(493,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 21:32:00.145','启动Ansible任务'),(494,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 21:40:35.124','启动Ansible任务'),(495,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 21:46:54.782','启动Ansible任务'),(496,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-08 21:50:06.050','启动Ansible任务'),(497,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 17:30:53.260','删除Ansible任务'),(498,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-27 17:32:55.259','启动Ansible任务'),(499,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-27 17:33:41.672','启动Ansible任务'),(500,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-27 17:35:17.923','启动Ansible任务'),(501,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-27 17:37:22.823','启动Ansible任务'),(502,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 17:57:31.098','删除Ansible任务'),(503,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 17:57:50.044','删除Ansible任务'),(504,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:13:42.047','删除Ansible任务'),(505,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:13:46.839','删除Ansible任务'),(506,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:14:00.296','删除Ansible任务'),(507,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:14:23.720','删除Ansible任务'),(508,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:14:30.232','删除Ansible任务'),(509,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:14:35.386','删除Ansible任务'),(510,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:14:40.481','删除Ansible任务'),(511,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:16:21.208','删除Ansible任务'),(512,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:16:26.256','删除Ansible任务'),(513,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:17:32.024','删除Ansible任务'),(514,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:18:46.933','删除Ansible任务'),(515,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:21:53.322','删除Ansible任务'),(516,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:24:20.043','删除Ansible任务'),(517,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:24:28.556','删除Ansible任务'),(518,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:24:33.421','删除Ansible任务'),(519,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:24:38.938','删除Ansible任务'),(520,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:25:45.463','删除Ansible任务'),(521,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:25:50.485','删除Ansible任务'),(522,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:25:57.424','删除Ansible任务'),(523,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:26:01.432','删除Ansible任务'),(524,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:26:03.857','删除Ansible任务'),(525,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:28:19.981','删除Ansible任务'),(526,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:28:29.543','删除Ansible任务'),(527,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-03-27 18:28:59.720','启动Ansible任务'),(528,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:40:45.364','删除Ansible任务'),(529,89,'admin','put','127.0.0.1','/api/v1/task/ansible/119','2026-03-27 18:40:56.663','删除Ansible任务'),(530,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 12:13:11.711','启动Ansible任务'),(531,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 12:13:38.160','启动Ansible任务'),(532,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 13:17:45.170','启动Ansible任务'),(533,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 13:24:07.502','启动Ansible任务'),(534,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:08:09.840','启动Ansible任务'),(535,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:13:30.780','启动Ansible任务'),(536,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:17:50.347','启动Ansible任务'),(537,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:18:36.780','启动Ansible任务'),(538,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:34:30.534','启动Ansible任务'),(539,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:37:27.054','启动Ansible任务'),(540,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:38:05.025','启动Ansible任务'),(541,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:40:56.235','启动Ansible任务'),(542,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:54:04.565','启动Ansible任务'),(543,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 14:54:30.648','启动Ansible任务'),(544,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:01:11.731','启动Ansible任务'),(545,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:01:31.703','启动Ansible任务'),(546,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:01:57.677','启动Ansible任务'),(547,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:02:42.857','启动Ansible任务'),(548,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:03:21.455','启动Ansible任务'),(549,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:05:48.101','启动Ansible任务'),(550,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:06:49.699','启动Ansible任务'),(551,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:07:39.281','启动Ansible任务'),(552,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:09:46.900','启动Ansible任务'),(553,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:51:19.961','启动Ansible任务'),(554,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:55:46.444','启动Ansible任务'),(555,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:58:11.164','启动Ansible任务'),(556,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:58:27.782','启动Ansible任务'),(557,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 15:59:48.298','启动Ansible任务'),(558,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 16:00:39.722','启动Ansible任务'),(559,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 16:01:25.187','启动Ansible任务'),(560,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 16:33:34.278','启动Ansible任务'),(561,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 16:34:11.609','启动Ansible任务'),(562,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 16:35:01.819','启动Ansible任务'),(563,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 16:35:44.715','启动Ansible任务'),(564,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 16:36:44.414','启动Ansible任务'),(565,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 17:42:22.810','启动Ansible任务'),(566,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 17:43:51.274','启动Ansible任务'),(567,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 17:51:31.967','启动Ansible任务'),(568,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 17:53:29.335','启动Ansible任务'),(569,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 17:53:56.424','启动Ansible任务'),(570,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 17:58:11.849','启动Ansible任务'),(571,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 18:01:05.323','启动Ansible任务'),(572,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 18:09:26.091','启动Ansible任务'),(573,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 18:16:06.548','启动Ansible任务'),(574,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 18:19:52.542','启动Ansible任务'),(575,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 18:23:44.034','启动Ansible任务'),(576,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 19:42:57.879','启动Ansible任务'),(577,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 19:52:15.972','启动Ansible任务'),(578,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 19:52:38.302','启动Ansible任务'),(579,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:00:38.731','启动Ansible任务'),(580,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:16:52.204','启动Ansible任务'),(581,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:18:08.018','启动Ansible任务'),(582,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:23:10.155','启动Ansible任务'),(583,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:26:35.962','启动Ansible任务'),(584,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:26:49.776','启动Ansible任务'),(585,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:31:58.717','启动Ansible任务'),(586,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:34:10.939','启动Ansible任务'),(587,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:34:24.704','启动Ansible任务'),(588,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:34:47.162','启动Ansible任务'),(589,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:35:19.877','启动Ansible任务'),(590,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:48:50.969','启动Ansible任务'),(591,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:50:55.061','启动Ansible任务'),(592,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:51:10.995','启动Ansible任务'),(593,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 20:53:54.231','启动Ansible任务'),(594,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 21:03:03.817','启动Ansible任务'),(595,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-02 21:04:28.932','启动Ansible任务'),(596,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-03 16:10:00.548','启动Ansible任务'),(597,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-03 16:10:24.993','启动Ansible任务'),(598,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-03 16:22:55.233','启动Ansible任务'),(599,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-03 16:25:20.950','启动Ansible任务'),(600,89,'admin','post','127.0.0.1','/api/v1/task/ansible/62/start','2026-04-03 16:27:27.779','启动Ansible任务'),(601,89,'admin','post','127.0.0.1','/api/v1/task/ansible','2026-04-03 16:28:48.715','创建Ansible任务'),(602,89,'admin','post','127.0.0.1','/api/v1/task/ansible/120/start','2026-04-03 16:28:52.029','启动Ansible任务'),(603,89,'admin','put','127.0.0.1','/api/v1/task/ansible/120','2026-04-03 16:29:27.024','删除Ansible任务'),(604,89,'admin','post','127.0.0.1','/api/v1/task/ansible/120/start','2026-04-03 16:29:29.458','启动Ansible任务'),(605,89,'admin','post','127.0.0.1','/api/v1/task/ansible/119/start','2026-04-03 17:25:04.042','启动Ansible任务'),(606,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster','2026-04-05 23:20:22.750','创建K8s集群'),(607,89,'admin','post','172.22.96.1','/api/v1/monitor/alert/webhook/prometheus','2026-04-06 00:00:03.384',''),(608,89,'admin','post','172.22.96.1','/api/v1/monitor/alert/webhook/prometheus','2026-04-06 00:01:17.553',''),(609,89,'admin','post','172.22.96.1','/api/v1/monitor/alert/webhook/prometheus','2026-04-06 00:01:28.045',''),(610,89,'admin','post','172.22.96.1','/api/v1/monitor/alert/webhook/prometheus','2026-04-06 00:02:51.929',''),(611,89,'admin','post','172.22.96.1','/api/v1/monitor/alert/template','2026-04-06 00:04:49.425',''),(612,89,'admin','post','172.22.96.1','/api/v1/monitor/alert/template','2026-04-06 10:53:07.892',''),(613,89,'admin','post','172.22.96.1','/api/v1/monitor/alert/template','2026-04-06 10:53:36.016',''),(614,89,'admin','delete','172.22.96.1','/api/v1/cmdb/hostdelete','2026-04-06 14:14:57.333','删除主机'),(615,89,'admin','put','172.22.96.1','/api/v1/config/ecsauthupdate','2026-04-06 14:17:21.001','修改ECS认证'),(616,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostcreate','2026-04-06 14:17:48.256','创建主机'),(617,89,'admin','put','172.22.96.1','/api/v1/k8s/cluster/35','2026-04-07 10:27:35.184','操作K8s集群'),(618,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/35','2026-04-07 10:28:27.852','操作K8s集群'),(619,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster','2026-04-07 10:28:35.074','创建K8s集群'),(620,89,'admin','delete','172.22.96.1','/api/v1/cmdb/hostdelete','2026-04-07 10:49:50.206','删除主机'),(621,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostcreate','2026-04-07 10:50:12.420','创建主机'),(622,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-04-07 11:52:39.454','启动Ansible任务'),(623,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-04-07 12:41:03.435','同步K8s集群'),(624,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-04-07 13:18:04.989','启动Ansible任务'),(625,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-04-09 17:49:44.917','启动Ansible任务'),(626,89,'admin','post','172.22.96.1','/api/v1/cmdb/sql/databaselist','2026-04-09 23:13:27.691',''),(627,89,'admin','post','172.22.96.1','/api/v1/cmdb/sql/databaselist','2026-04-09 23:13:30.025',''),(628,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-04-10 09:40:03.609','同步K8s集群'),(629,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-04-10 11:52:15.713','同步K8s集群'),(630,89,'admin','post','172.22.96.1','/api/v1/task/add','2026-04-10 12:49:55.676','新增任务'),(631,89,'admin','post','172.22.96.1','/api/v1/taskjob/start','2026-04-10 12:50:03.056','启动任务作业'),(632,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/fmusic-server/rollback','2026-04-10 15:09:02.204','回滚K8s部署'),(633,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-04-10 15:11:17.837','同步K8s集群'),(634,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/hot-reload','2026-04-10 16:38:55.732',''),(635,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/hot-reload','2026-04-10 16:39:04.562',''),(636,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/hot-reload','2026-04-10 16:42:11.172',''),(637,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files','2026-04-10 16:48:15.496',''),(638,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files/upload','2026-04-10 17:00:55.580',''),(639,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files/upload','2026-04-10 17:01:08.493',''),(640,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files/upload','2026-04-10 17:04:30.291',''),(641,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files/upload','2026-04-10 17:04:40.450',''),(642,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files/upload','2026-04-10 17:04:53.147',''),(643,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files/upload','2026-04-10 17:08:01.983',''),(644,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/pods/prometheus-grafana-5c5678db4c-rkwp6/files/upload','2026-04-10 17:08:59.210',''),(645,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files/upload','2026-04-10 17:10:12.932',''),(646,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files','2026-04-10 17:17:21.862',''),(647,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files/upload','2026-04-10 17:17:27.000',''),(648,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files/directory','2026-04-10 17:26:40.596',''),(649,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files','2026-04-10 17:26:44.404',''),(650,89,'admin','put','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files/content','2026-04-10 17:27:48.098',''),(651,89,'admin','put','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files/content','2026-04-10 17:27:53.305',''),(652,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-04-10 19:12:05.742','同步K8s集群'),(653,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces','2026-04-10 19:17:39.459','创建K8s命名空间'),(654,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-04-10 19:17:43.481','操作K8s命名空间'),(655,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-04-10 19:18:47.450','操作K8s命名空间'),(656,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-04-10 19:19:02.801','同步K8s集群'),(657,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/hot-reload','2026-04-10 19:21:26.919',''),(658,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/fmusic-server/scale','2026-04-10 19:22:56.507','伸缩K8s部署'),(659,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/fmusic-server/scale','2026-04-10 19:43:57.960','伸缩K8s部署'),(660,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files/upload','2026-04-11 13:23:07.413',''),(661,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files','2026-04-11 13:23:10.602',''),(662,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files/upload','2026-04-11 14:34:33.127',''),(663,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files','2026-04-11 14:34:35.941',''),(664,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files','2026-04-11 14:34:43.759',''),(665,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/fmusic-server-d6d948cf4-75gkh/files','2026-04-11 14:34:47.551',''),(666,89,'admin','post','172.22.96.1','/api/v1/menu/add','2026-04-11 15:41:56.733','新增菜单'),(667,89,'admin','put','172.22.96.1','/api/v1/menu/update','2026-04-11 15:42:58.876','修改菜单'),(668,89,'admin','put','172.22.96.1','/api/v1/menu/update','2026-04-11 15:43:12.086','修改菜单'),(669,89,'admin','delete','172.22.96.1','/api/v1/menu/delete','2026-04-11 15:47:31.320','删除菜单'),(670,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:48:35.657',''),(671,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:48:38.353',''),(672,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:48:43.607',''),(673,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:48:46.700',''),(674,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:48:56.221',''),(675,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:49:18.679',''),(676,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources/node-cpu-usage','2026-04-12 13:50:07.977',''),(677,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:50:11.187',''),(678,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:50:27.642',''),(679,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:54:00.182',''),(680,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:54:27.267',''),(681,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:56:04.682',''),(682,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:56:18.116',''),(683,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:56:34.392',''),(684,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 13:59:46.040',''),(685,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 14:00:06.153',''),(686,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources/node-memory-usage','2026-04-12 14:03:39.312',''),(687,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/monitor/crds/prometheusrules.monitoring.coreos.com/resources','2026-04-12 14:05:40.812',''),(688,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-04-12 17:57:11.666','启动Ansible任务'),(689,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-04-12 18:08:42.606','启动Ansible任务'),(690,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-04-12 18:17:45.896','启动Ansible任务'),(691,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-04-12 18:41:05.184','启动Ansible任务'),(692,89,'admin','post','127.0.0.1','/api/v1/admin/add','2026-04-13 22:29:49.089','新增管理员'),(693,106,'test','put','127.0.0.1','/api/v1/admin/updateStatus','2026-04-13 22:30:14.870',''),(694,106,'test','put','127.0.0.1','/api/v1/admin/updateStatus','2026-04-13 22:30:19.755',''),(695,89,'admin','put','127.0.0.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets/fmusic-config/yaml','2026-05-09 22:56:36.810',''),(696,89,'admin','post','127.0.0.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/fmusic-server/restart','2026-05-09 22:58:08.992','重启K8s部署'),(697,89,'admin','post','192.168.0.102','/api/v1/cmdb/hostcreate','2026-05-10 09:39:38.208','创建主机'),(698,89,'admin','delete','192.168.0.102','/api/v1/cmdb/hostdelete','2026-05-10 09:40:20.687','删除主机'),(699,89,'admin','post','192.168.0.102','/api/v1/cmdb/hostcreate','2026-05-10 09:40:56.823','创建主机'),(700,89,'admin','post','192.168.0.102','/api/v1/cmdb/hostcreate','2026-05-10 09:41:11.189','创建主机'),(701,89,'admin','put','192.168.0.102','/api/v1/cmdb/hostupdate','2026-05-10 09:43:59.617','修改主机'),(702,89,'admin','put','192.168.0.102','/api/v1/cmdb/hostupdate','2026-05-10 09:44:11.444','修改主机'),(703,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostcreate','2026-05-17 15:29:42.495','创建主机'),(704,89,'admin','put','172.22.96.1','/api/v1/cmdb/hostupdate','2026-05-17 15:30:03.825','修改主机'),(705,89,'admin','put','172.22.96.1','/api/v1/config/ecsauthupdate','2026-05-17 15:30:34.397','修改ECS认证'),(706,89,'admin','put','172.22.96.1','/api/v1/cmdb/hostupdate','2026-05-17 15:30:51.784','修改主机'),(707,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostssh/upload/532','2026-05-17 16:25:53.239',''),(708,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostssh/upload/532','2026-05-17 17:34:57.314',''),(709,89,'admin','delete','172.22.96.1','/api/v1/cmdb/hostssh/file','2026-05-17 17:35:05.186',''),(710,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostssh/upload/532','2026-05-17 17:36:10.642',''),(711,89,'admin','delete','172.22.96.1','/api/v1/cmdb/hostssh/file','2026-05-17 17:36:21.765',''),(712,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostssh/upload/532','2026-05-18 22:36:13.809',''),(713,89,'admin','delete','172.22.96.1','/api/v1/cmdb/hostssh/file','2026-05-18 22:36:17.833',''),(714,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-05-19 23:28:20.572','启动Ansible任务'),(715,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-05-19 23:28:44.721','启动Ansible任务'),(716,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostssh/upload/532','2026-05-19 23:34:52.225',''),(717,89,'admin','delete','172.22.96.1','/api/v1/cmdb/hostssh/file','2026-05-19 23:35:16.022',''),(718,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-05-19 23:38:17.605','启动Ansible任务'),(719,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-20 22:44:06.360','同步主机信息'),(720,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-20 22:58:27.897','同步主机信息'),(721,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-20 23:04:03.564','同步主机信息'),(722,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-20 23:05:30.704','同步主机信息'),(723,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-20 23:06:30.763','同步主机信息'),(724,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-20 23:07:00.112','同步主机信息'),(725,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-20 23:51:38.742','同步主机信息'),(726,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-21 22:48:58.668','同步主机信息'),(727,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-21 22:49:17.324','同步主机信息'),(728,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-21 22:59:02.635','同步主机信息'),(729,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-22 23:33:13.684',''),(730,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-23 00:09:06.535','同步主机信息'),(731,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-23 00:09:13.867','同步主机信息'),(732,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-23 00:09:19.254','同步主机信息'),(733,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-23 00:09:24.101','同步主机信息'),(734,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-23 14:03:34.027','创建K8s配置'),(735,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-23 14:03:39.528','创建K8s配置'),(736,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-23 14:03:50.636','创建K8s密钥'),(737,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-23 14:56:51.915','操作K8s Pod'),(738,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779519415912','2026-05-23 14:57:22.754','操作K8s部署'),(739,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/custom-deployment-1779519415912-5c5455495b-lpnnq','2026-05-23 14:57:45.718','操作K8s Pod'),(740,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779519415912','2026-05-23 14:57:53.838','操作K8s部署'),(741,106,'test','post','172.22.96.1','/api/v1/k8s/permission','2026-05-23 16:12:57.518',''),(742,106,'test','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 16:14:00.448',''),(743,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 16:14:37.036',''),(744,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 16:15:21.664',''),(745,89,'admin','put','172.22.96.1','/api/v1/k8s/user-group/1','2026-05-23 16:16:17.486',''),(746,89,'admin','put','172.22.96.1','/api/v1/k8s/user-group/1','2026-05-23 16:16:20.264',''),(747,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/1','2026-05-23 16:16:34.434',''),(748,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 16:52:33.503',''),(749,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/2','2026-05-23 16:52:38.509',''),(750,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group/members','2026-05-23 16:52:53.470',''),(751,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:01:32.556',''),(752,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/3','2026-05-23 17:01:38.253',''),(753,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:34:42.906',''),(754,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/4','2026-05-23 17:34:45.440',''),(755,89,'admin','delete','172.22.96.1','/api/v1/k8s/group-permission/undefined','2026-05-23 17:34:59.517',''),(756,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/member','2026-05-23 17:35:09.382',''),(757,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/member','2026-05-23 17:35:14.554',''),(758,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/member','2026-05-23 17:35:19.931',''),(759,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:39:34.652',''),(760,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/5','2026-05-23 17:39:37.102',''),(761,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:40:06.181',''),(762,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:42:28.743',''),(763,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:42:34.914',''),(764,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/8','2026-05-23 17:42:37.144',''),(765,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/6','2026-05-23 17:42:39.245',''),(766,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:48:36.567',''),(767,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/member','2026-05-23 17:48:41.376',''),(768,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/member','2026-05-23 17:48:46.451',''),(769,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group/members','2026-05-23 17:48:51.509',''),(770,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/member','2026-05-23 17:48:54.076',''),(771,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:49:02.934',''),(772,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/10','2026-05-23 17:49:07.951',''),(773,89,'admin','post','172.22.96.1','/api/v1/k8s/group-permission/batch','2026-05-23 17:49:16.369',''),(774,89,'admin','delete','172.22.96.1','/api/v1/k8s/group-permission/1','2026-05-23 17:49:25.949',''),(775,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:52:21.641',''),(776,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 17:52:23.766',''),(777,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/12','2026-05-23 17:52:27.117',''),(778,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/9','2026-05-23 17:52:48.209',''),(779,89,'admin','post','172.22.96.1','/api/v1/k8s/group-permission/batch','2026-05-23 17:53:03.625',''),(780,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/member','2026-05-23 17:53:33.503',''),(781,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group/members','2026-05-23 17:53:34.801',''),(782,89,'admin','put','172.22.96.1','/api/v1/k8s/group-permission/2','2026-05-23 17:53:43.143',''),(783,89,'admin','put','172.22.96.1','/api/v1/k8s/user-group/1','2026-05-23 17:53:59.449',''),(784,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-23 17:54:17.298',''),(785,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/2','2026-05-23 17:54:29.380',''),(786,89,'admin','put','172.22.96.1','/api/v1/k8s/group-permission/2','2026-05-23 17:55:50.808',''),(787,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 18:03:31.634',''),(788,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/13','2026-05-23 18:03:34.026',''),(789,89,'admin','delete','172.22.96.1','/api/v1/k8s/group-permission/2','2026-05-23 18:05:50.665',''),(790,89,'admin','post','172.22.96.1','/api/v1/k8s/group-permission/batch','2026-05-23 18:07:21.938',''),(791,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-23 18:07:37.553',''),(792,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/3','2026-05-23 18:08:24.438',''),(793,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/3','2026-05-23 18:08:58.007',''),(794,89,'admin','put','172.22.96.1','/api/v1/k8s/group-permission/3','2026-05-23 18:10:09.890',''),(795,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-23 18:48:39.051',''),(796,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/14','2026-05-23 18:48:49.339',''),(797,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/3','2026-05-23 18:52:51.422',''),(798,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/3','2026-05-23 19:21:22.343',''),(799,106,'test','delete','127.0.0.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/some-pod','2026-05-24 22:08:59.318','操作K8s Pod'),(800,106,'test','post','127.0.0.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments','2026-05-24 22:08:59.353','创建K8s部署'),(801,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-24 22:13:00.282',''),(802,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-24 23:01:53.350',''),(803,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/2','2026-05-24 23:02:09.800',''),(804,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/4','2026-05-24 23:02:15.486',''),(805,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/1','2026-05-24 23:02:20.620',''),(806,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/2','2026-05-24 23:03:39.409',''),(807,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/2','2026-05-24 23:03:49.168',''),(808,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/2','2026-05-24 23:03:58.803',''),(809,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-24 23:04:47.155',''),(810,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/2','2026-05-24 23:06:48.477',''),(811,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/1','2026-05-24 23:07:22.368',''),(812,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-24 23:07:58.869',''),(813,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/15','2026-05-24 23:08:10.202',''),(814,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-24 23:08:38.243',''),(815,89,'admin','post','172.22.96.1','/api/v1/k8s/group-permission/batch','2026-05-24 23:09:28.970',''),(816,89,'admin','delete','172.22.96.1','/api/v1/k8s/group-permission/4','2026-05-24 23:14:18.373',''),(817,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/3','2026-05-24 23:14:44.410',''),(818,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-24 23:17:50.916',''),(819,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/2','2026-05-24 23:17:55.217',''),(820,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/2','2026-05-24 23:18:06.912',''),(821,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-24 23:18:40.158',''),(822,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/4','2026-05-24 23:19:09.605',''),(823,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-24 23:19:55.628',''),(824,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/3','2026-05-24 23:20:53.150',''),(825,106,'test','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-24 23:22:00.426',''),(826,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/3','2026-05-24 23:42:20.114',''),(827,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/3','2026-05-24 23:53:11.730',''),(828,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-24 23:54:14.026',''),(829,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/5','2026-05-24 23:54:17.831',''),(830,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-24 23:54:32.166',''),(831,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-24 23:55:05.916',''),(832,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-24 23:55:33.918',''),(833,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/6','2026-05-24 23:55:41.134',''),(834,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-24 23:55:50.894',''),(835,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-24 23:56:31.557',''),(836,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-24 23:56:43.507',''),(837,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-24 23:56:53.428',''),(838,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/7','2026-05-24 23:57:12.801',''),(839,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-24 23:57:37.100',''),(840,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-25 21:43:34.177',''),(841,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/5','2026-05-25 21:43:40.033',''),(842,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 21:43:49.857',''),(843,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/9','2026-05-25 21:43:53.477',''),(844,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-25 21:54:12.677',''),(845,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/5','2026-05-25 21:54:47.869',''),(846,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/8','2026-05-25 22:04:38.792',''),(847,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 22:04:49.423',''),(848,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 22:08:50.962',''),(849,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/11','2026-05-25 22:08:55.530',''),(850,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-25 22:09:12.030',''),(851,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/6','2026-05-25 22:09:15.203',''),(852,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-25 22:10:06.485',''),(853,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/10','2026-05-25 22:14:17.525',''),(854,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:19:43.070','创建K8s配置'),(855,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:19:47.843','创建K8s配置'),(856,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces','2026-05-25 22:28:18.404','创建K8s命名空间'),(857,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-05-25 22:28:35.328','操作K8s命名空间'),(858,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 22:30:23.000',''),(859,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/12','2026-05-25 22:30:30.976',''),(860,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 22:30:48.183',''),(861,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:30:57.623',''),(862,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:31:35.318',''),(863,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-05-25 22:32:15.067','同步K8s集群'),(864,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/13','2026-05-25 22:33:19.187',''),(865,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 22:33:24.774',''),(866,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/6','2026-05-25 22:35:10.235',''),(867,89,'admin','post','172.22.96.1','/api/v1/k8s/permission/batch','2026-05-25 22:35:41.192',''),(868,89,'admin','post','172.22.96.1','/api/v1/k8s/permission/batch','2026-05-25 22:35:44.603',''),(869,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:36:57.046','创建K8s密钥'),(870,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:37:10.628',''),(871,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:37:15.760','创建K8s密钥'),(872,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:37:24.332','创建K8s密钥'),(873,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:37:51.593','创建K8s密钥'),(874,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:38:25.682','创建K8s配置'),(875,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:38:39.444','创建K8s配置'),(876,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:38:37.144','创建K8s配置'),(877,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:38:41.215','创建K8s配置'),(878,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:38:45.139','创建K8s配置'),(879,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:39:03.697','创建K8s配置'),(880,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:39:22.555','创建K8s配置'),(881,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:39:32.604','创建K8s配置'),(882,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-25 22:39:36.599','创建K8s配置'),(883,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:39:57.743','创建K8s密钥'),(884,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:40:03.336','创建K8s密钥'),(885,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:40:40.863','创建K8s密钥'),(886,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/secrets','2026-05-25 22:41:20.652','创建K8s密钥'),(887,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:41:45.815','操作K8s Pod'),(888,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779720104566','2026-05-25 22:41:49.695','操作K8s部署'),(889,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:42:06.682',''),(890,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:43:22.296','操作K8s Pod'),(891,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/9','2026-05-25 22:43:56.484',''),(892,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/6','2026-05-25 22:44:00.239',''),(893,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:44:06.363','操作K8s Pod'),(894,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779720245351','2026-05-25 22:44:10.606','操作K8s部署'),(895,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/6','2026-05-25 22:45:30.343',''),(896,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:45:47.883',''),(897,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:45:55.349','操作K8s Pod'),(898,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:46:09.207',''),(899,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:46:13.462','操作K8s Pod'),(900,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:46:37.777',''),(901,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:46:41.769','操作K8s Pod'),(902,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779720401757','2026-05-25 22:46:44.761','操作K8s部署'),(903,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:47:40.297',''),(904,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:47:46.577','操作K8s Pod'),(905,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779720464350','2026-05-25 22:47:54.069','操作K8s部署'),(906,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 22:48:17.792',''),(907,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 22:48:25.624','操作K8s Pod'),(908,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779720504421','2026-05-25 22:48:29.349','操作K8s部署'),(909,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-25 23:02:26.656',''),(910,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/7','2026-05-25 23:02:29.827',''),(911,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 23:02:42.291',''),(912,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/15','2026-05-25 23:02:46.207',''),(913,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:03:20.031','操作K8s Pod'),(914,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779721398398','2026-05-25 23:03:31.643','操作K8s部署'),(915,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:06:07.165','操作K8s Pod'),(916,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779721566514','2026-05-25 23:06:11.708','操作K8s部署'),(917,106,'test','post','127.0.0.1','/api/v1/k8s/cluster/1/namespaces/fmusic/deployments','2026-05-25 23:25:29.524','创建K8s部署'),(918,89,'admin','post','127.0.0.1','/api/v1/k8s/cluster/1/namespaces/fmusic/deployments','2026-05-25 23:27:00.053','创建K8s部署'),(919,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:30:47.971','操作K8s Pod'),(920,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-25 23:31:12.864',''),(921,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/8','2026-05-25 23:31:13.517',''),(922,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:31:19.801',''),(923,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:31:26.254','操作K8s Pod'),(924,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:31:27.698','操作K8s Pod'),(925,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:31:41.607',''),(926,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-25 23:31:49.717',''),(927,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/16','2026-05-25 23:31:52.725',''),(928,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:32:13.136','操作K8s Pod'),(929,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779723086134','2026-05-25 23:32:17.888','操作K8s部署'),(930,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779723086134','2026-05-25 23:32:24.536','操作K8s部署'),(931,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779723086134','2026-05-25 23:32:42.555','操作K8s部署'),(932,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:33:03.800',''),(933,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:33:09.083',''),(934,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:33:14.803','操作K8s Pod'),(935,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:33:26.868',''),(936,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:33:30.949','操作K8s Pod'),(937,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779723195069','2026-05-25 23:33:34.726','操作K8s部署'),(938,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:33:46.118',''),(939,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:33:52.378','操作K8s Pod'),(940,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:34:15.293',''),(941,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:34:23.210','操作K8s Pod'),(942,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:34:23.681','操作K8s Pod'),(943,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779723232485','2026-05-25 23:34:32.184','操作K8s部署'),(944,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:34:40.367',''),(945,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779723232485','2026-05-25 23:34:43.603','操作K8s部署'),(946,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:34:58.226',''),(947,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:35:01.676','操作K8s Pod'),(948,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:35:02.887','操作K8s Pod'),(949,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-25 23:35:11.581',''),(950,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-25 23:35:16.827','操作K8s Pod'),(951,89,'admin','put','172.22.96.1','/api/v1/k8s/permission/6','2026-05-25 23:36:15.418',''),(952,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/6','2026-05-25 23:36:29.022',''),(953,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-25 23:37:12.604',''),(954,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-25 23:37:49.833',''),(955,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-25 23:37:54.368',''),(956,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/17','2026-05-25 23:37:57.648',''),(957,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/10','2026-05-25 23:38:04.953',''),(958,89,'admin','post','172.22.96.1','/api/v1/k8s/group-permission/batch','2026-05-25 23:38:22.724',''),(959,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779723316301','2026-05-25 23:38:40.708','操作K8s部署'),(960,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-27 00:08:55.403',''),(961,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/11','2026-05-27 00:08:58.023',''),(962,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-27 00:09:06.951',''),(963,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/18','2026-05-27 00:09:10.189',''),(964,89,'admin','delete','172.22.96.1','/api/v1/k8s/group-permission/5','2026-05-27 00:09:14.335',''),(965,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-27 00:09:24.218',''),(966,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/9','2026-05-27 00:09:26.651',''),(967,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-27 00:11:05.964',''),(968,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-27 00:12:19.268',''),(969,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/17','2026-05-27 00:12:22.115',''),(970,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/12','2026-05-27 00:33:23.747',''),(971,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-27 00:34:49.001',''),(972,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/13','2026-05-27 00:34:57.007',''),(973,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-27 00:35:29.540',''),(974,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-27 00:36:00.595','操作K8s Pod'),(975,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779813362396','2026-05-27 00:36:07.520','操作K8s部署'),(976,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 00:36:20.565',''),(977,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-27 00:36:26.861','操作K8s Pod'),(978,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-27 00:36:27.925','操作K8s Pod'),(979,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-27 21:51:30.088',''),(980,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/15','2026-05-27 21:51:33.692',''),(981,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-27 21:51:39.824',''),(982,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/19','2026-05-27 21:51:44.346',''),(983,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-27 21:52:55.006',''),(984,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/10','2026-05-27 21:52:59.228',''),(985,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 21:53:05.260',''),(986,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 21:54:03.159',''),(987,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 21:54:50.072',''),(988,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 21:59:03.356',''),(989,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-27 21:59:27.690','操作K8s Pod'),(990,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-27 22:26:43.791',''),(991,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/12','2026-05-27 22:26:48.701',''),(992,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/11','2026-05-27 22:26:58.309',''),(993,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 22:27:53.174',''),(994,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 22:29:25.097',''),(995,89,'admin','put','172.22.96.1','/api/v1/admin/updateStatus','2026-05-27 22:32:01.505',''),(996,89,'admin','put','172.22.96.1','/api/v1/admin/updateStatus','2026-05-27 22:32:38.773',''),(997,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 22:36:47.785',''),(998,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-27 22:37:00.340','操作K8s Pod'),(999,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/configmaps','2026-05-27 22:37:09.953','创建K8s配置'),(1000,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-27 23:07:44.773',''),(1001,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/20','2026-05-27 23:07:48.562',''),(1002,89,'admin','delete','172.22.96.1','/api/v1/k8s/group-permission/undefined','2026-05-27 23:07:59.039',''),(1003,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-27 23:12:14.265',''),(1004,89,'admin','post','172.22.96.1','/api/v1/k8s/user-group','2026-05-27 23:27:38.734',''),(1005,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/21','2026-05-27 23:27:47.624',''),(1006,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-27 23:33:02.143',''),(1007,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:34:19.892',''),(1008,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-27 23:34:44.212',''),(1009,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/13','2026-05-27 23:34:49.717',''),(1010,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-27 23:37:35.349',''),(1011,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/14','2026-05-27 23:37:40.016',''),(1012,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/16','2026-05-27 23:43:45.813',''),(1013,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/18','2026-05-27 23:45:21.746',''),(1014,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/19','2026-05-27 23:45:34.180',''),(1015,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:45:44.394',''),(1016,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-27 23:45:59.584','操作K8s Pod'),(1017,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779896763072','2026-05-27 23:46:03.186','操作K8s部署'),(1018,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:46:13.391',''),(1019,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779896763072','2026-05-27 23:46:18.297','操作K8s部署'),(1020,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:46:35.949',''),(1021,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:47:23.465',''),(1022,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/14','2026-05-27 23:48:00.672',''),(1023,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:48:33.739',''),(1024,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:49:14.079',''),(1025,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:49:41.288',''),(1026,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:50:02.177',''),(1027,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:50:19.753',''),(1028,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:50:38.750',''),(1029,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:50:51.647',''),(1030,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-27 23:51:11.902',''),(1031,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/pods/yaml','2026-05-27 23:51:40.653','操作K8s Pod'),(1032,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/fmusic/deployments/custom-deployment-1779897104580','2026-05-27 23:51:46.335','操作K8s部署'),(1033,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-28 23:33:29.003',''),(1034,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-28 23:34:42.500',''),(1035,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 22:26:03.718',''),(1036,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces','2026-05-29 22:28:07.611','创建K8s命名空间'),(1037,89,'admin','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-05-29 22:29:07.599','操作K8s命名空间'),(1038,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 22:30:39.566',''),(1039,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/14','2026-05-29 22:31:23.717',''),(1040,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/14','2026-05-29 22:31:55.719',''),(1041,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/14','2026-05-29 22:33:24.445',''),(1042,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-29 22:35:50.725',''),(1043,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces','2026-05-29 22:55:47.157','创建K8s命名空间'),(1044,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-29 22:56:25.735',''),(1045,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-05-29 22:56:56.809','操作K8s命名空间'),(1046,106,'test','post','172.22.96.1','/api/v1/k8s/cluster/36/namespaces','2026-05-29 22:57:09.805','创建K8s命名空间'),(1047,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 22:57:41.192',''),(1048,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-05-29 22:57:46.834','操作K8s命名空间'),(1049,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 22:59:02.569',''),(1050,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-05-29 22:59:51.365','操作K8s命名空间'),(1051,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 23:00:06.425',''),(1052,106,'test','delete','172.22.96.1','/api/v1/k8s/cluster/36/namespaces/test','2026-05-29 23:00:14.530','操作K8s命名空间'),(1053,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 23:00:49.342',''),(1054,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 23:15:58.705',''),(1055,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-29 23:16:28.728',''),(1056,106,'test','post','172.22.96.1','/api/v1/k8s/permission','2026-05-29 23:21:34.203',''),(1057,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-29 23:21:43.056',''),(1058,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/17','2026-05-29 23:22:28.403',''),(1059,89,'admin','post','172.22.96.1','/api/v1/k8s/permission','2026-05-29 23:23:18.464',''),(1060,89,'admin','delete','172.22.96.1','/api/v1/k8s/permission/18','2026-05-29 23:23:41.040',''),(1061,89,'admin','post','172.22.96.1','/api/v1/task/ansible/view','2026-05-30 00:00:38.502',''),(1062,89,'admin','put','172.22.96.1','/api/v1/task/ansible/view/undefined','2026-05-30 00:01:04.038',''),(1063,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/view/undefined','2026-05-30 00:01:10.655',''),(1064,89,'admin','post','172.22.96.1','/api/v1/task/ansible/view','2026-05-30 00:01:19.700',''),(1065,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/view/undefined','2026-05-30 00:01:28.303',''),(1066,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/view/undefined','2026-05-30 00:01:31.152',''),(1067,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/view/undefined','2026-05-30 00:01:42.435',''),(1068,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/view/undefined','2026-05-30 00:04:27.717',''),(1069,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/view/2','2026-05-30 00:05:18.909',''),(1070,89,'admin','post','172.22.96.1','/api/v1/task/ansible/view','2026-05-30 00:05:23.992',''),(1071,89,'admin','post','172.22.96.1','/api/v1/task/ansible/view','2026-05-30 00:05:45.250',''),(1072,89,'admin','put','172.22.96.1','/api/v1/task/ansible/56','2026-05-30 00:16:34.099','删除Ansible任务'),(1073,89,'admin','put','172.22.96.1','/api/v1/task/ansible/56','2026-05-30 00:16:59.655','删除Ansible任务'),(1074,89,'admin','put','172.22.96.1','/api/v1/task/ansible/56','2026-05-30 00:22:33.778','删除Ansible任务'),(1075,89,'admin','put','172.22.96.1','/api/v1/task/ansible/56','2026-05-30 00:27:16.708','删除Ansible任务'),(1076,89,'admin','put','172.22.96.1','/api/v1/task/ansible/56','2026-05-30 00:28:02.479','删除Ansible任务'),(1077,89,'admin','post','172.22.96.1','/api/v1/task/ansible','2026-05-30 00:28:49.184','创建Ansible任务'),(1078,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/121','2026-05-30 00:29:03.588','删除Ansible任务'),(1079,89,'admin','put','172.22.96.1','/api/v1/task/ansible/56','2026-05-30 00:39:14.559','删除Ansible任务'),(1080,89,'admin','put','172.22.96.1','/api/v1/task/ansible/56','2026-05-30 00:39:20.194','删除Ansible任务'),(1081,89,'admin','post','172.22.96.1','/api/v1/menu/add','2026-05-30 12:46:25.089','新增菜单'),(1082,89,'admin','put','172.22.96.1','/api/v1/menu/update','2026-05-30 12:47:16.065','修改菜单'),(1083,89,'admin','put','172.22.96.1','/api/v1/menu/update','2026-05-30 12:47:49.714','修改菜单'),(1084,89,'admin','put','172.22.96.1','/api/v1/menu/update','2026-05-30 12:49:03.872','修改菜单'),(1085,89,'admin','post','172.22.96.1','/api/v1/menu/add','2026-05-30 13:01:15.580','新增菜单'),(1086,89,'admin','put','172.22.96.1','/api/v1/menu/update','2026-05-30 13:01:43.765','修改菜单'),(1087,89,'admin','post','172.22.96.1','/api/v1/cmdb/idc','2026-05-30 13:03:32.878',''),(1088,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 13:04:14.786',''),(1089,89,'admin','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-30 13:05:46.171',''),(1090,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-30 13:06:26.789',''),(1091,89,'admin','delete','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-30 13:20:56.127',''),(1092,89,'admin','post','172.22.96.1','/api/v1/menu/add','2026-05-30 13:22:15.057','新增菜单'),(1093,89,'admin','delete','172.22.96.1','/api/v1/menu/delete','2026-05-30 13:22:32.026','删除菜单'),(1094,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 17:38:57.459',''),(1095,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 17:41:05.307',''),(1096,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 17:45:30.570',''),(1097,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/2','2026-05-30 17:59:51.864',''),(1098,89,'admin','post','172.22.96.1','/api/v1/cmdb/idc','2026-05-30 18:00:57.669',''),(1099,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/3','2026-05-30 18:05:53.014',''),(1100,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/3','2026-05-30 18:09:38.525',''),(1101,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/3','2026-05-30 18:10:28.351',''),(1102,89,'admin','delete','172.22.96.1','/api/v1/cmdb/idc/2','2026-05-30 18:11:06.917',''),(1103,89,'admin','delete','172.22.96.1','/api/v1/cmdb/cabinet/3','2026-05-30 18:11:18.815',''),(1104,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 18:11:43.095',''),(1105,89,'admin','delete','172.22.96.1','/api/v1/cmdb/cabinet/4','2026-05-30 18:11:49.053',''),(1106,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 18:11:54.015',''),(1107,89,'admin','delete','172.22.96.1','/api/v1/cmdb/cabinet/5','2026-05-30 18:18:44.001',''),(1108,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 18:18:47.180',''),(1109,89,'admin','delete','172.22.96.1','/api/v1/cmdb/cabinet/6','2026-05-30 18:18:56.667',''),(1110,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 18:19:13.952',''),(1111,89,'admin','delete','172.22.96.1','/api/v1/cmdb/cabinet/2','2026-05-30 18:20:17.839',''),(1112,89,'admin','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-30 18:30:05.411',''),(1113,89,'admin','post','172.22.96.1','/api/v1/cmdb/network','2026-05-30 18:42:02.547',''),(1114,89,'admin','post','172.22.96.1','/api/v1/cmdb/network','2026-05-30 18:42:22.299',''),(1115,89,'admin','post','172.22.96.1','/api/v1/cmdb/idc','2026-05-30 18:42:55.833',''),(1116,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/2','2026-05-30 18:43:06.835',''),(1117,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/2','2026-05-30 18:43:25.882',''),(1118,89,'admin','post','172.22.96.1','/api/v1/cmdb/cabinet','2026-05-30 18:43:50.756',''),(1119,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/2','2026-05-30 18:44:02.414',''),(1120,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/2','2026-05-30 18:44:30.044',''),(1121,89,'admin','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-30 18:44:29.248',''),(1122,89,'admin','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-30 18:44:37.846',''),(1123,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/2','2026-05-30 18:44:42.801',''),(1124,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/2','2026-05-30 18:44:48.641',''),(1125,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/2','2026-05-30 18:45:12.846',''),(1126,89,'admin','delete','172.22.96.1','/api/v1/cmdb/physical/2','2026-05-30 18:45:14.896',''),(1127,89,'admin','delete','172.22.96.1','/api/v1/cmdb/network/2','2026-05-30 20:38:20.633',''),(1128,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-30 20:43:31.567',''),(1129,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-30 20:43:36.881',''),(1130,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-30 20:43:40.864',''),(1131,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/1','2026-05-30 23:52:16.678',''),(1132,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/1','2026-05-30 23:54:31.920',''),(1133,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/1','2026-05-30 23:54:36.077',''),(1134,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/1','2026-05-30 23:54:41.665',''),(1135,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/8','2026-05-30 23:54:56.134',''),(1136,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/8','2026-05-30 23:54:59.472',''),(1137,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-30 23:55:41.162',''),(1138,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 00:06:37.788',''),(1139,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 00:06:36.440',''),(1140,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 00:06:40.504',''),(1141,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 00:06:54.709',''),(1142,89,'admin','put','172.22.96.1','/api/v1/cmdb/network/1','2026-05-31 00:06:54.292',''),(1143,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-05-31 00:08:17.995','删除Ansible任务'),(1144,89,'admin','post','172.22.96.1','/api/v1/task/ansible/115/start','2026-05-31 00:09:19.125','启动Ansible任务'),(1145,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-05-31 00:09:29.115','启动Ansible任务'),(1146,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/8','2026-05-31 00:11:17.471',''),(1147,89,'admin','put','172.22.96.1','/api/v1/cmdb/cabinet/8','2026-05-31 00:11:20.786',''),(1148,106,'test','put','172.22.96.1','/api/v1/cmdb/network/1','2026-05-31 00:23:52.726',''),(1149,106,'test','put','172.22.96.1','/api/v1/cmdb/network/1','2026-05-31 00:24:06.382',''),(1150,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 00:24:38.161',''),(1151,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/1','2026-05-31 00:24:42.339',''),(1152,106,'test','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 00:24:58.993',''),(1153,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/3','2026-05-31 00:25:11.613',''),(1154,106,'test','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 00:25:24.520',''),(1155,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 00:26:04.732',''),(1156,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 00:26:07.643',''),(1157,89,'admin','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-31 00:35:00.039','同步主机信息'),(1158,89,'admin','delete','172.22.96.1','/api/v1/cmdb/permission/1','2026-05-31 03:11:21.014',''),(1159,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 03:11:47.065',''),(1160,106,'test','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-31 03:13:49.619','同步主机信息'),(1161,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/2','2026-05-31 03:14:07.815',''),(1162,106,'test','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-31 03:14:10.872','同步主机信息'),(1163,106,'test','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-31 03:14:13.228','同步主机信息'),(1164,106,'test','put','172.22.96.1','/api/v1/cmdb/network/4','2026-05-31 03:14:26.787',''),(1165,106,'test','put','172.22.96.1','/api/v1/cmdb/network/4','2026-05-31 03:14:29.574',''),(1166,89,'admin','delete','172.22.96.1','/api/v1/cmdb/permission/2','2026-05-31 03:15:27.034',''),(1167,106,'test','put','172.22.96.1','/api/v1/cmdb/network/4','2026-05-31 03:15:35.871',''),(1168,106,'test','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 03:17:03.232',''),(1169,106,'test','delete','172.22.96.1','/api/v1/cmdb/permission/3','2026-05-31 03:17:05.911',''),(1170,106,'test','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 03:39:28.309',''),(1171,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 03:46:09.302',''),(1172,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/4','2026-05-31 03:46:26.499',''),(1173,106,'test','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 03:46:37.629',''),(1174,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 03:52:28.738',''),(1175,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/5','2026-05-31 03:52:53.431',''),(1176,89,'admin','delete','172.22.96.1','/api/v1/cmdb/permission/5','2026-05-31 03:54:02.729',''),(1177,89,'admin','delete','172.22.96.1','/api/v1/cmdb/permission/4','2026-05-31 03:54:51.085',''),(1178,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 03:55:24.382',''),(1179,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/6','2026-05-31 03:55:40.975',''),(1180,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 03:56:01.463',''),(1181,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 03:56:06.165',''),(1182,106,'test','post','172.22.96.1','/api/v1/cmdb/hostsync','2026-05-31 03:56:25.909','同步主机信息'),(1183,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/6','2026-05-31 03:56:56.090',''),(1184,89,'admin','delete','172.22.96.1','/api/v1/cmdb/permission/6','2026-05-31 03:57:49.530',''),(1185,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 03:58:17.603',''),(1186,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/7','2026-05-31 04:06:04.917',''),(1187,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/7','2026-05-31 04:06:08.920',''),(1188,106,'test','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 04:22:39.260',''),(1189,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 04:23:17.017',''),(1190,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:23:39.066',''),(1191,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission','2026-05-31 04:24:06.551',''),(1192,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:24:24.237',''),(1193,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:24:57.923',''),(1194,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:25:00.398',''),(1195,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 04:25:13.374',''),(1196,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:25:12.208',''),(1197,106,'test','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 04:25:20.732',''),(1198,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 04:25:27.016',''),(1199,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 04:25:31.263',''),(1200,106,'test','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-31 04:25:36.954',''),(1201,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:27:49.771',''),(1202,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:28:08.783',''),(1203,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:31:26.175',''),(1204,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:31:34.301',''),(1205,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:31:45.121',''),(1206,106,'test','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 04:31:53.719',''),(1207,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/6','2026-05-31 04:31:56.883',''),(1208,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:33:31.142',''),(1209,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/6','2026-05-31 04:33:44.578',''),(1210,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:33:44.621',''),(1211,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:33:50.369',''),(1212,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:33:56.162',''),(1213,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:34:02.643',''),(1214,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:34:07.321',''),(1215,106,'test','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 04:34:14.136',''),(1216,106,'test','delete','172.22.96.1','/api/v1/cmdb/network/7','2026-05-31 04:34:16.781',''),(1217,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 04:34:34.718',''),(1218,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 04:34:40.202',''),(1219,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:34:51.079',''),(1220,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:35:05.403',''),(1221,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:35:11.196',''),(1222,106,'test','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-31 04:35:19.685',''),(1223,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:35:25.758',''),(1224,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:35:33.915',''),(1225,106,'test','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-31 04:35:42.652',''),(1226,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/3','2026-05-31 04:35:45.379',''),(1227,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:36:52.815',''),(1228,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:36:59.163',''),(1229,106,'test','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-31 04:37:19.863',''),(1230,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/3','2026-05-31 04:37:12.227',''),(1231,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/4','2026-05-31 04:37:13.995',''),(1232,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:38:06.867',''),(1233,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/4','2026-05-31 04:38:10.629',''),(1234,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/3','2026-05-31 04:38:13.034',''),(1235,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:41:52.969',''),(1236,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:42:08.265',''),(1237,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:42:21.521',''),(1238,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:54:52.701',''),(1239,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-05-31 04:55:03.666',''),(1240,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:55:14.272',''),(1241,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:55:45.349',''),(1242,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 04:56:58.306',''),(1243,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:57:01.828',''),(1244,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:57:09.309',''),(1245,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 04:57:14.718',''),(1246,89,'admin','post','172.22.96.1','/api/v1/cmdb/network','2026-05-31 04:57:43.570',''),(1247,106,'test','put','172.22.96.1','/api/v1/cmdb/network/8','2026-05-31 04:57:58.939',''),(1248,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:58:23.392',''),(1249,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 04:58:48.360',''),(1250,106,'test','put','172.22.96.1','/api/v1/cmdb/network/8','2026-05-31 04:59:00.449',''),(1251,106,'test','put','172.22.96.1','/api/v1/cmdb/network/8','2026-05-31 04:59:05.616',''),(1252,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 05:20:04.190',''),(1253,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 05:20:20.273',''),(1254,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 05:20:14.779',''),(1255,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 05:20:28.166',''),(1256,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/7','2026-05-31 05:20:43.338',''),(1257,89,'admin','post','172.22.96.1','/api/v1/cmdb/physical','2026-05-31 05:21:20.388',''),(1258,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 05:21:32.253',''),(1259,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 05:21:45.123',''),(1260,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/5','2026-05-31 05:21:57.755',''),(1261,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 05:22:08.227',''),(1262,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/5','2026-05-31 05:22:25.622',''),(1263,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 05:22:26.361',''),(1264,106,'test','put','172.22.96.1','/api/v1/cmdb/network/5','2026-05-31 05:22:34.080',''),(1265,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 05:22:39.799',''),(1266,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 05:22:42.273',''),(1267,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/5','2026-05-31 05:22:53.121',''),(1268,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/8','2026-05-31 05:23:03.414',''),(1269,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/5','2026-05-31 05:23:07.214',''),(1270,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission/user-group','2026-05-31 05:23:38.638',''),(1271,89,'admin','post','172.22.96.1','/api/v1/cmdb/permission/user-group/members','2026-05-31 05:23:45.710',''),(1272,106,'test','put','172.22.96.1','/api/v1/cmdb/hostupdate','2026-05-31 05:27:39.628','修改主机'),(1273,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 23:01:47.657',''),(1274,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 23:01:51.876',''),(1275,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/21','2026-05-31 23:09:21.574',''),(1276,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/21','2026-05-31 23:09:23.319',''),(1277,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/21','2026-05-31 23:09:43.621',''),(1278,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/21','2026-05-31 23:09:40.138',''),(1279,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-05-31 23:14:18.812',''),(1280,89,'admin','delete','172.22.96.1','/api/v1/k8s/user-group/1','2026-05-31 23:45:35.479',''),(1281,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/role/4','2026-05-31 23:45:47.082',''),(1282,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/21','2026-05-31 23:45:39.657',''),(1283,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/20','2026-05-31 23:45:41.317',''),(1284,89,'admin','delete','172.22.96.1','/api/v1/k8s/rbac/binding/14','2026-05-31 23:45:43.156',''),(1285,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/role','2026-05-31 23:47:30.205',''),(1286,89,'admin','post','172.22.96.1','/api/v1/k8s/rbac/binding','2026-05-31 23:48:38.632',''),(1287,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:50:07.377',''),(1288,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:50:53.906',''),(1289,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:51:30.477',''),(1290,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:51:42.834',''),(1291,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:52:04.946',''),(1292,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:52:23.954',''),(1293,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:52:38.404',''),(1294,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:52:57.485',''),(1295,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:53:27.696',''),(1296,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:53:50.684',''),(1297,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:54:06.955',''),(1298,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:54:30.761',''),(1299,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:55:05.173',''),(1300,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/binding/22','2026-05-31 23:55:27.409',''),(1301,89,'admin','put','172.22.96.1','/api/v1/k8s/rbac/role/15','2026-05-31 23:56:12.356',''),(1302,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/7','2026-06-01 00:07:55.986',''),(1303,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/7','2026-06-01 00:09:53.947',''),(1304,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/7','2026-06-01 00:10:39.808',''),(1305,106,'test','post','172.22.96.1','/api/v1/cmdb/idc','2026-06-01 00:12:31.381',''),(1306,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/5','2026-06-01 00:13:17.720',''),(1307,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/5','2026-06-01 00:13:20.699',''),(1308,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-06-01 00:13:35.839',''),(1309,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/5','2026-06-01 00:13:46.341',''),(1310,106,'test','delete','172.22.96.1','/api/v1/cmdb/physical/5','2026-06-01 00:13:48.545',''),(1311,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-06-01 00:13:59.359',''),(1312,106,'test','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-06-01 00:14:24.580',''),(1313,89,'admin','put','172.22.96.1','/api/v1/cmdb/physical/1','2026-06-01 00:14:49.528',''),(1314,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-08 22:30:30.420','删除Ansible任务'),(1315,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-08 22:30:38.983','删除Ansible任务'),(1316,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-08 22:30:42.512','启动Ansible任务'),(1317,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-08 22:31:48.646','删除Ansible任务'),(1318,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-08 22:31:51.385','删除Ansible任务'),(1319,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-08 23:43:56.962','启动Ansible任务'),(1320,89,'admin','post','172.22.96.1','/api/v1/task/ansible/120/start','2026-06-08 23:44:36.303','启动Ansible任务'),(1321,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-08 23:45:43.881','删除Ansible任务'),(1322,89,'admin','put','172.22.96.1','/api/v1/task/ansible/115','2026-06-08 23:51:18.700','删除Ansible任务'),(1323,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-08 23:51:35.144','删除Ansible任务'),(1324,89,'admin','put','172.22.96.1','/api/v1/task/ansible/115','2026-06-08 23:51:47.987','删除Ansible任务'),(1325,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-08 23:52:25.530','启动Ansible任务'),(1326,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 00:42:40.725','删除Ansible任务'),(1327,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 00:42:50.704','删除Ansible任务'),(1328,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 00:43:23.593','删除Ansible任务'),(1329,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 00:57:05.832','删除Ansible任务'),(1330,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 00:57:41.711','删除Ansible任务'),(1331,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-09 00:58:22.469','启动Ansible任务'),(1332,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 22:02:13.058','删除Ansible任务'),(1333,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 22:02:20.935','删除Ansible任务'),(1334,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-09 22:02:23.638','启动Ansible任务'),(1335,89,'admin','put','172.22.96.1','/api/v1/task/ansible/119','2026-06-09 22:02:52.419','删除Ansible任务'),(1336,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-09 22:02:57.552','启动Ansible任务'),(1337,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/56','2026-06-09 22:03:41.482','删除Ansible任务'),(1338,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/62','2026-06-09 22:03:44.288','删除Ansible任务'),(1339,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/105','2026-06-09 22:03:45.921','删除Ansible任务'),(1340,89,'admin','delete','172.22.96.1','/api/v1/task/ansible/120','2026-06-09 22:03:54.195','删除Ansible任务'),(1341,89,'admin','post','172.22.96.1','/api/v1/task/ansible','2026-06-09 22:07:17.572','创建Ansible任务'),(1342,89,'admin','put','172.22.96.1','/api/v1/config/ansible/3','2026-06-09 22:07:35.481',''),(1343,89,'admin','put','172.22.96.1','/api/v1/task/ansible/122','2026-06-09 22:07:51.234','删除Ansible任务'),(1344,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-06-09 22:07:55.016','启动Ansible任务'),(1345,89,'admin','put','172.22.96.1','/api/v1/task/ansible/122','2026-06-09 22:08:18.640','删除Ansible任务'),(1346,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-06-09 22:08:26.278','启动Ansible任务'),(1347,89,'admin','put','172.22.96.1','/api/v1/task/ansible/122','2026-06-09 22:09:15.748','删除Ansible任务'),(1348,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-06-09 22:09:18.491','启动Ansible任务'),(1349,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-06-09 22:31:15.289','启动Ansible任务'),(1350,89,'admin','put','172.22.96.1','/api/v1/task/ansible/122','2026-06-09 22:31:30.126','删除Ansible任务'),(1351,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-06-09 22:31:33.390','启动Ansible任务'),(1352,89,'admin','put','172.22.96.1','/api/v1/config/ansible/3','2026-06-09 22:35:09.257',''),(1353,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-06-09 22:35:14.822','启动Ansible任务'),(1354,89,'admin','put','172.22.96.1','/api/v1/config/ansible/3','2026-06-09 22:36:35.785',''),(1355,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-06-09 22:36:31.335','启动Ansible任务'),(1356,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-13 19:08:51.120','启动Ansible任务'),(1357,89,'admin','post','127.0.0.1','/api/v1/apps','2026-06-22 22:46:32.781','创建应用'),(1358,89,'admin','delete','127.0.0.1','/api/v1/apps/20','2026-06-22 22:46:36.241','操作应用'),(1359,89,'admin','post','127.0.0.1','/api/v1/apps','2026-06-22 22:46:45.243','创建应用'),(1360,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-23 01:26:48.030','启动Ansible任务'),(1361,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-23 02:08:50.625','启动Ansible任务'),(1362,89,'admin','put','172.22.96.1','/api/v1/cmdb/hostupdate','2026-06-23 22:51:43.910','修改主机'),(1363,89,'admin','put','172.22.96.1','/api/v1/cmdb/hostupdate','2026-06-23 22:52:00.409','修改主机'),(1364,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-24 01:05:59.711','启动Ansible任务'),(1365,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-25 02:04:32.621','启动Ansible任务'),(1366,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-06-26 01:27:19.093','同步K8s集群'),(1367,89,'admin','post','172.22.96.1','/api/v1/task/ansible/119/start','2026-06-26 01:43:58.119','启动Ansible任务'),(1368,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-06-26 02:07:03.101',''),(1369,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-06-26 02:07:17.378',''),(1370,89,'admin','put','172.22.96.1','/api/v1/cmdb/permission/9','2026-06-26 02:07:44.556',''),(1371,89,'admin','post','172.22.96.1','/api/v1/k8s/cluster/36/sync','2026-06-26 02:08:10.579','同步K8s集群'),(1372,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-07-02 22:02:33.840','启动Ansible任务'),(1373,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-07-02 22:02:54.879','启动Ansible任务'),(1374,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-07-02 22:03:44.031','启动Ansible任务'),(1375,89,'admin','post','172.22.96.1','/api/v1/task/ansible/122/start','2026-07-02 22:04:08.958','启动Ansible任务'),(1376,89,'admin','post','172.22.96.1','/api/v1/taskjob/stop','2026-07-02 22:21:54.286','停止任务作业'),(1377,89,'admin','post','172.22.96.1','/api/v1/taskjob/stop','2026-07-02 22:25:16.209','停止任务作业'),(1378,89,'admin','delete','172.22.96.1','/api/v1/task/delete','2026-07-02 22:40:30.904','删除任务'),(1379,89,'admin','post','172.22.96.1','/api/v1/task/add','2026-07-02 22:40:37.941','新增任务'),(1380,89,'admin','post','172.22.96.1','/api/v1/taskjob/start','2026-07-02 22:40:41.404','启动任务作业'),(1381,89,'admin','put','172.22.96.1','/api/v1/config/ansible/2','2026-07-02 22:53:43.029',''),(1382,89,'admin','put','172.22.96.1','/api/v1/config/ansible/2','2026-07-02 22:53:59.915',''),(1383,89,'admin','put','172.22.96.1','/api/v1/config/ansible/2','2026-07-02 23:00:43.065','');
/*!40000 ALTER TABLE `sys_operation_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_post`
--

DROP TABLE IF EXISTS `sys_post`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_post` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '岗位ID',
  `post_code` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin NOT NULL COMMENT '岗位编码',
  `post_name` varchar(50) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin NOT NULL COMMENT '岗位名称',
  `post_status` int NOT NULL DEFAULT '1' COMMENT '状态（1->正常 2->停用）',
  `create_time` datetime DEFAULT NULL COMMENT '创建时间',
  `remark` varchar(500) CHARACTER SET utf8mb3 COLLATE utf8mb3_bin DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin ROW_FORMAT=DYNAMIC COMMENT='岗位信息表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_post`
--

LOCK TABLES `sys_post` WRITE;
/*!40000 ALTER TABLE `sys_post` DISABLE KEYS */;
INSERT INTO `sys_post` VALUES (1,'AAA','研发总监',1,'2023-06-14 20:08:22','主管各个部门'),(10,'ops','运维工程师',1,'2025-06-28 22:46:33','运维工程师'),(11,'dev','研发工程师',1,'2025-06-28 22:50:29','研发工程师'),(12,'test','测试工程师',1,'2025-06-28 22:52:57','测试工程师');
/*!40000 ALTER TABLE `sys_post` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role`
--

DROP TABLE IF EXISTS `sys_role`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键',
  `role_name` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '角色名称',
  `role_key` varchar(64) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL COMMENT '角色权限字符串',
  `status` int NOT NULL DEFAULT '1' COMMENT '启用状态：1->启用；2->禁用',
  `description` varchar(500) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL COMMENT '描述',
  `create_time` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `name` (`role_name`) USING BTREE,
  KEY `create_time` (`create_time`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='后台角色表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role`
--

LOCK TABLES `sys_role` WRITE;
/*!40000 ALTER TABLE `sys_role` DISABLE KEYS */;
INSERT INTO `sys_role` VALUES (1,'超级管理员','admin',1,'最大权限','2023-06-12 20:04:53'),(13,'游客','test',1,'test1','2025-07-03 18:47:25');
/*!40000 ALTER TABLE `sys_role` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role_menu`
--

DROP TABLE IF EXISTS `sys_role_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role_menu` (
  `role_id` int DEFAULT NULL COMMENT '角色ID',
  `menu_id` int DEFAULT NULL COMMENT '菜单ID'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 ROW_FORMAT=DYNAMIC COMMENT='角色和菜单关系表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role_menu`
--

LOCK TABLES `sys_role_menu` WRITE;
/*!40000 ALTER TABLE `sys_role_menu` DISABLE KEYS */;
INSERT INTO `sys_role_menu` VALUES (1,72),(1,80),(1,78),(1,149),(1,150),(1,151),(1,152),(1,153),(1,154),(1,155),(1,165),(1,166),(1,241),(1,88),(1,89),(1,90),(1,91),(1,95),(1,146),(1,147),(1,148),(1,217),(1,218),(1,219),(1,220),(1,221),(1,222),(1,81),(1,82),(1,167),(1,168),(1,169),(1,170),(1,171),(1,83),(1,172),(1,173),(1,174),(1,175),(1,176),(1,177),(1,105),(1,178),(1,179),(1,180),(1,181),(1,182),(1,183),(1,184),(1,93),(1,185),(1,186),(1,187),(1,188),(1,189),(1,190),(1,191),(1,192),(1,193),(1,194),(1,195),(1,196),(1,106),(1,197),(1,198),(1,199),(1,200),(1,201),(1,202),(1,203),(1,204),(1,205),(1,206),(1,108),(1,107),(1,110),(1,142),(1,143),(1,144),(1,145),(1,162),(1,163),(1,164),(1,111),(1,139),(1,140),(1,141),(1,160),(1,161),(1,212),(1,213),(1,229),(1,230),(1,231),(1,232),(1,233),(1,234),(1,235),(1,216),(1,236),(1,237),(1,238),(1,239),(1,240),(1,97),(1,99),(1,136),(1,137),(1,138),(1,156),(1,157),(1,98),(1,133),(1,134),(1,135),(1,100),(1,130),(1,131),(1,132),(1,159),(1,101),(1,103),(1,102),(1,126),(1,127),(1,128),(1,129),(1,4),(1,6),(1,16),(1,17),(1,18),(1,60),(1,7),(1,21),(1,22),(1,23),(1,24),(1,8),(1,26),(1,27),(1,28),(1,9),(1,29),(1,30),(1,31),(1,10),(1,32),(1,33),(1,34),(1,215),(1,84),(1,85),(1,123),(1,124),(1,125),(1,86),(1,122),(1,119),(1,120),(1,121),(1,104),(1,114),(1,115),(1,117),(1,118),(1,44),(1,45),(1,47),(1,113),(1,73),(1,46),(1,49),(1,62),(1,96),(1,109),(13,72),(13,149),(13,150),(13,154),(13,165),(13,89),(13,146),(13,218),(13,219),(13,167),(13,168),(13,170),(13,172),(13,173),(13,174),(13,175),(13,178),(13,179),(13,180),(13,181),(13,182),(13,185),(13,186),(13,190),(13,193),(13,195),(13,197),(13,200),(13,203),(13,205),(13,108),(13,107),(13,142),(13,144),(13,162),(13,139),(13,140),(13,160),(13,229),(13,233),(13,236),(13,240),(13,136),(13,137),(13,156),(13,133),(13,130),(13,132),(13,159),(13,103),(13,127),(13,16),(13,125),(13,122),(13,118),(13,73),(13,62),(13,96),(13,80),(13,78),(13,88),(13,95),(13,81),(13,82),(13,83),(13,105),(13,93),(13,106),(13,109),(13,110),(13,111),(13,212),(13,213),(13,216),(13,97),(13,99),(13,98),(13,100),(13,101),(13,102),(13,4),(13,6),(13,84),(13,85),(13,86),(13,104),(13,44),(13,45),(13,46);
/*!40000 ALTER TABLE `sys_role_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_ansible`
--

DROP TABLE IF EXISTS `task_ansible`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_ansible` (
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
) ENGINE=InnoDB AUTO_INCREMENT=123 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_ansible`
--

LOCK TABLES `task_ansible` WRITE;
/*!40000 ALTER TABLE `task_ansible` DISABLE KEYS */;
INSERT INTO `task_ansible` VALUES (115,'te','',2,'https://gitee.com/zhang_fan1024/ansible-playbook.git','null','[]','',4,'2026-03-02 16:25:13.184','2026-05-31 00:09:19.125','任务目录不存在: /home/zhoujunjie/pythonproject/AutoOps/api/task/115/te (请尝试删除并重新创建任务)',1,0,'','',1,1,NULL,NULL,NULL,3,'',0,NULL),(119,'test','',2,'https://gitee.com/zhang_fan1024/ansible-playbook.git','null','[]','',3,'2026-03-08 00:12:22.997','2026-06-26 01:44:11.173','',2,12,'','',1,1,0,0,0,5,'0 0 * * *',0,NULL),(122,'print','',1,'','null','[]','',3,'2026-06-09 22:07:17.538','2026-07-02 22:04:01.650','',1,0,'','',1,1,2,3,0,3,'',0,1);
/*!40000 ALTER TABLE `task_ansible` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_ansible_history`
--

DROP TABLE IF EXISTS `task_ansible_history`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_ansible_history` (
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
) ENGINE=InnoDB AUTO_INCREMENT=223 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_ansible_history`
--

LOCK TABLES `task_ansible_history` WRITE;
/*!40000 ALTER TABLE `task_ansible_history` DISABLE KEYS */;
INSERT INTO `task_ansible_history` VALUES (85,104,'104-1772224847',4,'',0,1,0,'',NULL,NULL,'2026-02-28 04:40:47.647'),(87,103,'103-1772260766',4,'',26,1,0,'',NULL,NULL,'2026-02-28 14:39:26.465'),(90,105,'105-1772268422',3,'',2,1,0,'',NULL,NULL,'2026-02-28 16:47:02.138'),(91,107,'107-1772271861',3,'',2,1,0,'',NULL,NULL,'2026-02-28 17:44:21.814'),(92,109,'109-1772272596',3,'',2,1,0,'',NULL,NULL,'2026-02-28 17:56:36.136'),(93,109,'109-1772272620',3,'',2,1,0,'',NULL,NULL,'2026-02-28 17:57:00.059'),(94,112,'112-1772273507',3,'',2,1,0,'',NULL,NULL,'2026-02-28 18:11:47.805'),(95,105,'105-1772273586',3,'',2,1,0,'',NULL,NULL,'2026-02-28 18:13:06.594'),(96,103,'103-1772273621',4,'',27,1,0,'',NULL,NULL,'2026-02-28 18:13:41.033'),(97,103,'103-1772439349',4,'',27,1,0,'',NULL,NULL,'2026-03-02 16:15:49.770'),(98,105,'105-1772440617',3,'',2,1,0,'',NULL,NULL,'2026-03-02 16:36:57.262'),(99,118,'118-1772729190',4,'',0,1,0,'',NULL,NULL,'2026-03-06 00:46:30.017'),(185,120,'120-1775204933',3,'',1,1,0,'',NULL,NULL,'2026-04-03 16:28:53.160'),(186,120,'120-1775204970',3,'',0,1,0,'',NULL,NULL,'2026-04-03 16:29:30.230'),(201,120,'120-1780933476',4,'',0,1,0,'',NULL,NULL,'2026-06-08 23:44:36.356'),(214,119,'119-1782149228',3,'',22,1,0,'',NULL,NULL,'2026-06-23 01:27:08.719'),(215,119,'119-1782151754',3,'',25,1,0,'',NULL,NULL,'2026-06-23 02:09:14.931'),(216,119,'119-1782234378',3,'',18,1,0,'',NULL,NULL,'2026-06-24 01:06:18.990'),(217,119,'119-1782324285',3,'',11,1,0,'',NULL,NULL,'2026-06-25 02:04:45.677'),(218,119,'119-1782409451',3,'',12,1,0,'',NULL,NULL,'2026-06-26 01:44:11.183'),(220,122,'122-1783000975',3,'',0,1,0,'',NULL,NULL,'2026-07-02 22:02:55.486'),(221,122,'122-1783001017',3,'',0,1,0,'',NULL,NULL,'2026-07-02 22:03:37.073'),(222,122,'122-1783001041',3,'',0,1,0,'',NULL,NULL,'2026-07-02 22:04:01.655');
/*!40000 ALTER TABLE `task_ansible_history` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_ansible_view`
--

DROP TABLE IF EXISTS `task_ansible_view`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_ansible_view` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键ID''',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '''视图名称''',
  `created_at` datetime(3) DEFAULT NULL COMMENT '''创建时间''',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_task_ansible_view_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_ansible_view`
--

LOCK TABLES `task_ansible_view` WRITE;
/*!40000 ALTER TABLE `task_ansible_view` DISABLE KEYS */;
INSERT INTO `task_ansible_view` VALUES (1,'test','2026-05-30 00:00:38.480','2026-05-30 00:00:38.480'),(3,'1','2026-05-30 00:05:23.980','2026-05-30 00:05:23.980');
/*!40000 ALTER TABLE `task_ansible_view` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_ansiblework`
--

DROP TABLE IF EXISTS `task_ansiblework`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_ansiblework` (
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
) ENGINE=InnoDB AUTO_INCREMENT=142 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_ansiblework`
--

LOCK TABLES `task_ansiblework` WRITE;
/*!40000 ALTER TABLE `task_ansiblework` DISABLE KEYS */;
INSERT INTO `task_ansiblework` VALUES (113,115,'01-linux-os.yaml','task/115/te/01-linux-os.yaml','',4,NULL,NULL,0,0,'任务目录不存在: /home/zhoujunjie/pythonproject/AutoOps/api/task/115/te (请尝试删除并重新创建任务)',NULL),(139,119,'01-linux-os.yaml','/home/zhoujunjie/pythonproject/AutoOps/api/task/119/test/01-linux-os.yaml','logs/ansible/119/139/20260626014358/01-linux-os.yaml.log',3,'2026-06-26 01:43:58.131','2026-06-26 01:44:10.371',12,0,'',NULL),(140,119,'02-os.yaml','/home/zhoujunjie/pythonproject/AutoOps/api/task/119/test/02-os.yaml','logs/ansible/119/140/20260626014410/02-os.yaml.log',3,'2026-06-26 01:44:10.386','2026-06-26 01:44:11.162',0,0,'',NULL),(141,122,'test.yml','task/122/print/test.yml','logs/ansible/122/141/20260702220408/test.yml.log',3,'2026-07-02 22:04:08.965','2026-07-02 22:04:01.644',0,0,'',NULL);
/*!40000 ALTER TABLE `task_ansiblework` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_ansiblework_history`
--

DROP TABLE IF EXISTS `task_ansiblework_history`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_ansiblework_history` (
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
) ENGINE=InnoDB AUTO_INCREMENT=422 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_ansiblework_history`
--

LOCK TABLES `task_ansiblework_history` WRITE;
/*!40000 ALTER TABLE `task_ansiblework_history` DISABLE KEYS */;
INSERT INTO `task_ansiblework_history` VALUES (169,85,104,104,'test.yml',4,'logs/ansible/104/104/20260228044047/test.yml.log',0,'2026-02-28 04:40:47.653'),(172,87,103,102,'01-linux-os.yaml',3,'logs/ansible/103/102/20260228143900/01-linux-os.yaml.log',24,'2026-02-28 14:39:26.471'),(173,87,103,103,'02-os.yaml',4,'logs/ansible/103/103/20260228143924/02-os.yaml.log',2,'2026-02-28 14:39:26.471'),(176,90,105,105,'test.yml',3,'logs/ansible/105/105/20260228164659/test.yml.log',2,'2026-02-28 16:47:02.145'),(177,91,107,107,'test.yml',3,'logs/ansible/107/107/20260228174419/test.yml.log',2,'2026-02-28 17:44:21.825'),(178,92,109,108,'test.yml',3,'logs/ansible/109/108/20260228175633/test.yml.log',2,'2026-02-28 17:56:36.143'),(179,93,109,108,'test.yml',3,'logs/ansible/109/108/20260228175657/test.yml.log',2,'2026-02-28 17:57:00.066'),(180,94,112,109,'test.yml',3,'logs/ansible/112/109/20260228181145/test.yml.log',2,'2026-02-28 18:11:47.813'),(181,95,105,105,'test.yml',3,'logs/ansible/105/105/20260228181304/test.yml.log',2,'2026-02-28 18:13:06.603'),(182,96,103,102,'01-linux-os.yaml',3,'logs/ansible/103/102/20260228181313/01-linux-os.yaml.log',25,'2026-02-28 18:13:41.047'),(183,96,103,103,'02-os.yaml',4,'logs/ansible/103/103/20260228181338/02-os.yaml.log',2,'2026-02-28 18:13:41.047'),(184,97,103,102,'01-linux-os.yaml',3,'logs/ansible/103/102/20260302161521/01-linux-os.yaml.log',26,'2026-03-02 16:15:49.774'),(185,97,103,103,'02-os.yaml',4,'logs/ansible/103/103/20260302161547/02-os.yaml.log',1,'2026-03-02 16:15:49.774'),(186,98,105,105,'test.yml',3,'logs/ansible/105/105/20260302163654/test.yml.log',2,'2026-03-02 16:36:57.270'),(187,99,118,122,'01-linux-os.yaml',4,'logs/ansible/118/122/20260306004629/01-linux-os.yaml.log',0,'2026-03-06 00:46:30.025'),(188,99,118,123,'02-os.yaml',4,'logs/ansible/118/123/20260306004629/02-os.yaml.log',0,'2026-03-06 00:46:30.025'),(359,185,120,126,'test.yml',3,'logs/ansible/120/126/20260403162852/test.yml.log',1,'2026-04-03 16:28:53.167'),(360,186,120,126,'test.yml',3,'logs/ansible/120/126/20260403162929/test.yml.log',0,'2026-04-03 16:29:30.235'),(389,201,120,126,'test.yml',4,'logs/ansible/120/126/20260608234436/test.yml.log',0,'2026-06-08 23:44:36.371'),(408,214,119,139,'01-linux-os.yaml',3,'logs/ansible/119/139/20260623012648/01-linux-os.yaml.log',20,'2026-06-23 01:27:08.778'),(409,214,119,140,'02-os.yaml',3,'logs/ansible/119/140/20260623012706/02-os.yaml.log',2,'2026-06-23 01:27:08.778'),(410,215,119,139,'01-linux-os.yaml',3,'logs/ansible/119/139/20260623020850/01-linux-os.yaml.log',24,'2026-06-23 02:09:14.965'),(411,215,119,140,'02-os.yaml',3,'logs/ansible/119/140/20260623020913/02-os.yaml.log',1,'2026-06-23 02:09:14.965'),(412,216,119,139,'01-linux-os.yaml',3,'logs/ansible/119/139/20260624010559/01-linux-os.yaml.log',17,'2026-06-24 01:06:19.012'),(413,216,119,140,'02-os.yaml',3,'logs/ansible/119/140/20260624010617/02-os.yaml.log',1,'2026-06-24 01:06:19.012'),(414,217,119,139,'01-linux-os.yaml',3,'logs/ansible/119/139/20260625020432/01-linux-os.yaml.log',11,'2026-06-25 02:04:45.728'),(415,217,119,140,'02-os.yaml',3,'logs/ansible/119/140/20260625020444/02-os.yaml.log',0,'2026-06-25 02:04:45.728'),(416,218,119,139,'01-linux-os.yaml',3,'logs/ansible/119/139/20260626014358/01-linux-os.yaml.log',12,'2026-06-26 01:44:11.327'),(417,218,119,140,'02-os.yaml',3,'logs/ansible/119/140/20260626014410/02-os.yaml.log',0,'2026-06-26 01:44:11.327'),(419,220,122,141,'test.yml',3,'logs/ansible/122/141/20260702220254/test.yml.log',0,'2026-07-02 22:02:55.491'),(420,221,122,141,'test.yml',3,'logs/ansible/122/141/20260702220344/test.yml.log',0,'2026-07-02 22:03:37.078'),(421,222,122,141,'test.yml',3,'logs/ansible/122/141/20260702220408/test.yml.log',0,'2026-07-02 22:04:01.661');
/*!40000 ALTER TABLE `task_ansiblework_history` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_job`
--

DROP TABLE IF EXISTS `task_job`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_job` (
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
) ENGINE=InnoDB AUTO_INCREMENT=69 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_job`
--

LOCK TABLES `task_job` WRITE;
/*!40000 ALTER TABLE `task_job` DISABLE KEYS */;
INSERT INTO `task_job` VALUES (67,'11',1,'2','528','','',3,107,'',NULL,'2026-04-10 12:51:46.198','2026-04-10 12:49:55.651',1,NULL,NULL,NULL,1,NULL),(68,'test',1,'2','532','','',3,101,'',NULL,'2026-07-02 22:42:22.986','2026-07-02 22:40:37.933',1,NULL,NULL,NULL,1,NULL);
/*!40000 ALTER TABLE `task_job` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_template`
--

DROP TABLE IF EXISTS `task_template`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_template` (
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
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_template`
--

LOCK TABLES `task_template` WRITE;
/*!40000 ALTER TABLE `task_template` DISABLE KEYS */;
INSERT INTO `task_template` VALUES (2,'1-数字 1-100',1,'#!/bin/bash\n\n# 从 1 到 100，每秒打印一个数字\nfor ((i = 1; i <= 100; i++)); do\n    echo \"[$(date +%H:%M:%S)] $i\"\n    sleep 1\ndone\n\necho \"完成：所有数字 1-100 已打印完毕。\"\n','2025-08-06 12:47:57.073','2025-08-12 16:14:49.394','admin','admin','测试脚本'),(17,'Hello World',2,'print(\"Hello World!\")\r\n','2026-01-26 17:36:09.716','2026-01-26 17:38:29.026','admin','admin','');
/*!40000 ALTER TABLE `task_template` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `task_work`
--

DROP TABLE IF EXISTS `task_work`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `task_work` (
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
) ENGINE=InnoDB AUTO_INCREMENT=117 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `task_work`
--

LOCK TABLES `task_work` WRITE;
/*!40000 ALTER TABLE `task_work` DISABLE KEYS */;
INSERT INTO `task_work` VALUES (90,48,11,1,2,'[2025-09-29-21:42:02] 任务开始\n进程统计信息如下\n总进程数量为:79\nRunning 进程数为:1\nStoped 进程数为:0\nSleeping 进程数为:55\nZombie 进程数为:0\n[2025-09-29-21:42:02] 任务完成\n','logs/task_48/task_48_template_11.log','2025-09-29 21:42:00.299','2025-09-29 21:42:06.276',5,'2025-09-29 20:58:17.259',0,'2025-09-29 21:45:00.000',NULL,NULL),(91,48,0,0,4,'','',NULL,NULL,0,'2025-09-29 20:58:17.515',2,'2025-09-29 21:00:00.000',NULL,NULL),(115,67,2,528,3,'[2026-04-10-12:50:04] 任务开始\n[12:50:04] 1\n[12:50:05] 2\n[12:50:06] 3\n[12:50:07] 4\n[12:50:08] 5\n[12:50:09] 6\n[12:50:10] 7\n[12:50:11] 8\n[12:50:12] 9\n[12:50:13] 10\n[12:50:14] 11\n[12:50:15] 12\n[12:50:16] 13\n[12:50:17] 14\n[12:50:18] 15\n[12:50:19] 16\n[12:50:20] 17\n[12:50:21] 18\n[12:50:22] 19\n[12:50:23] 20\n[12:50:24] 21\n[12:50:25] 22\n[12:50:26] 23\n[12:50:27] 24\n[12:50:28] 25\n[12:50:29] 26\n[12:50:30] 27\n[12:50:31] 28\n[12:50:32] 29\n[12:50:33] 30\n[12:50:34] 31\n[12:50:35] 32\n[12:50:36] 33\n[12:50:37] 34\n[12:50:38] 35\n[12:50:39] 36\n[12:50:40] 37\n[12:50:41] 38\n[12:50:42] 39\n[12:50:43] 40\n[12:50:44] 41\n[12:50:45] 42\n[12:50:46] 43\n[12:50:47] 44\n[12:50:48] 45\n[12:50:49] 46\n[12:50:50] 47\n[12:50:51] 48\n[12:50:52] 49\n[12:50:53] 50\n[12:50:54] 51\n[12:50:55] 52\n[12:50:56] 53\n[12:50:57] 54\n[12:50:58] 55\n[12:50:59] 56\n[12:51:00] 57\n[12:51:01] 58\n[12:51:02] 59\n[12:51:03] 60\n[12:51:04] 61\n[12:51:05] 62\n[12:51:06] 63\n[12:51:07] 64\n[12:51:08] 65\n[12:51:09] 66\n[12:51:10] 67\n[12:51:11] 68\n[12:51:12] 69\n[12:51:13] 70\n[12:51:14] 71\n[12:51:15] 72\n[12:51:16] 73\n[12:51:17] 74\n[12:51:18] 75\n[12:51:19] 76\n[12:51:20] 77\n[12:51:21] 78\n[12:51:22] 79\n[12:51:23] 80\n[12:51:24] 81\n[12:51:25] 82\n[12:51:26] 83\n[12:51:27] 84\n[12:51:28] 85\n[12:51:29] 86\n[12:51:30] 87\n[12:51:31] 88\n[12:51:32] 89\n[12:51:33] 90\n[12:51:34] 91\n[12:51:35] 92\n[12:51:36] 93\n[12:51:38] 94\n[12:51:39] 95\n[12:51:40] 96\n[12:51:41] 97\n[12:51:42] 98\n[12:51:43] 99\n[12:51:44] 100\n完成：所有数字 1-100 已打印完毕。\n[2026-04-10-12:51:45] 任务完成\n','logs/task_67/task_67_template_2.log','2026-04-10 12:50:03.057','2026-04-10 12:51:46.160',107,'2026-04-10 12:49:55.665',0,NULL,NULL,NULL),(116,68,2,532,3,'[2026-07-02-22:40:41] 任务开始\n[22:40:41] 1\n[22:40:42] 2\n[22:40:43] 3\n[22:40:44] 4\n[22:40:45] 5\n[22:40:46] 6\n[22:40:47] 7\n[22:40:48] 8\n[22:40:49] 9\n[22:40:51] 10\n[22:40:52] 11\n[22:40:53] 12\n[22:40:54] 13\n[22:40:55] 14\n[22:40:56] 15\n[22:40:57] 16\n[22:40:58] 17\n[22:40:59] 18\n[22:41:00] 19\n[22:41:01] 20\n[22:41:02] 21\n[22:41:03] 22\n[22:41:04] 23\n[22:41:05] 24\n[22:41:06] 25\n[22:41:07] 26\n[22:41:08] 27\n[22:41:09] 28\n[22:41:10] 29\n[22:41:11] 30\n[22:41:12] 31\n[22:41:13] 32\n[22:41:14] 33\n[22:41:15] 34\n[22:41:16] 35\n[22:41:17] 36\n[22:41:18] 37\n[22:41:19] 38\n[22:41:20] 39\n[22:41:21] 40\n[22:41:22] 41\n[22:41:23] 42\n[22:41:24] 43\n[22:41:25] 44\n[22:41:26] 45\n[22:41:27] 46\n[22:41:28] 47\n[22:41:29] 48\n[22:41:30] 49\n[22:41:31] 50\n[22:41:32] 51\n[22:41:33] 52\n[22:41:34] 53\n[22:41:35] 54\n[22:41:36] 55\n[22:41:37] 56\n[22:41:38] 57\n[22:41:39] 58\n[22:41:40] 59\n[22:41:41] 60\n[22:41:42] 61\n[22:41:43] 62\n[22:41:44] 63\n[22:41:45] 64\n[22:41:46] 65\n[22:41:47] 66\n[22:41:48] 67\n[22:41:49] 68\n[22:41:50] 69\n[22:41:51] 70\n[22:41:52] 71\n[22:41:53] 72\n[22:41:54] 73\n[22:41:55] 74\n[22:41:56] 75\n[22:41:57] 76\n[22:41:58] 77\n[22:41:59] 78\n[22:42:00] 79\n[22:42:01] 80\n[22:42:02] 81\n[22:42:03] 82\n[22:42:04] 83\n[22:42:05] 84\n[22:42:06] 85\n[22:42:07] 86\n[22:42:08] 87\n[22:42:09] 88\n[22:42:10] 89\n[22:42:11] 90\n[22:42:12] 91\n[22:42:13] 92\n[22:42:14] 93\n[22:42:15] 94\n[22:42:16] 95\n[22:42:17] 96\n[22:42:18] 97\n[22:42:19] 98\n[22:42:20] 99\n[22:42:21] 100\n完成：所有数字 1-100 已打印完毕。\n[2026-07-02-22:42:22] 任务完成\n','logs/task_68/task_68_template_2.log','2026-07-02 22:40:41.404','2026-07-02 22:42:22.968',101,'2026-07-02 22:40:37.935',0,NULL,NULL,NULL);
/*!40000 ALTER TABLE `task_work` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tool_link`
--

DROP TABLE IF EXISTS `tool_link`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tool_link` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '''主键''',
  `title` longtext NOT NULL COMMENT '''导航标题''',
  `icon` longtext COMMENT '''导航图标''',
  `link` longtext NOT NULL COMMENT '''链接地址''',
  `sort` bigint DEFAULT '0' COMMENT '''排序''',
  `create_time` datetime(3) NOT NULL COMMENT '''创建时间''',
  `update_time` datetime(3) DEFAULT NULL COMMENT '''更新时间''',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tool_link`
--

LOCK TABLES `tool_link` WRITE;
/*!40000 ALTER TABLE `tool_link` DISABLE KEYS */;
INSERT INTO `tool_link` VALUES (2,'百度','http://10.7.16.22:8080/api/v1/upload/20251023/625775000.svg','https://www.baidu.com/',0,'2025-10-23 15:54:08.512','2025-10-23 15:54:08.512'),(3,'aws','http://10.7.16.22:8080/api/v1/upload/20251023/985806000.svg','https://us-west-2.console.aws.amazon.com/eks/clusters?region=us-west-2#',0,'2025-10-23 15:55:59.543','2025-10-23 15:55:59.543'),(4,'美女一号','http://10.7.16.22:8080/api/v1/upload/20251023/646700000.png','https://gitee.com/zhang_fan1024',0,'2025-10-23 15:56:22.231','2025-10-23 15:56:22.231'),(5,'美女二号','http://10.7.16.22:8080/api/v1/upload/20251023/733520000.png','https://demo.spug.cc/',0,'2025-10-23 15:56:48.601','2025-10-23 15:56:48.601'),(6,'凡人修仙传','http://10.7.16.22:8080/api/v1/upload/20251023/771236000.png','http://115.190.10.126/#/dashboard',0,'2025-10-23 15:57:21.676','2025-10-23 15:57:21.676'),(7,'腾讯','http://10.7.16.22:8080/api/v1/upload/20251205/92506000.png','https://cloud.tencent.com/login?s_url=https%3A%2F%2Fconsole.cloud.tencent.com%2Ftke2%2Fcluster%2Fsub%2Flist%2Fbasic%2Finfo%3Frid%3D1%26clusterId%3Dcls-301g0fi0',0,'2025-10-23 15:57:55.559','2025-12-05 20:12:08.966');
/*!40000 ALTER TABLE `tool_link` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tool_service_deploy`
--

DROP TABLE IF EXISTS `tool_service_deploy`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tool_service_deploy` (
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
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tool_service_deploy`
--

LOCK TABLES `tool_service_deploy` WRITE;
/*!40000 ALTER TABLE `tool_service_deploy` DISABLE KEYS */;
INSERT INTO `tool_service_deploy` VALUES (7,'Redis','redis','redis-7.2',501,'8.130.14.34','/opt/data/redis','','','{\"REDIS_MAXMEMORY\":\"2gb\",\"REDIS_PASSWORD\":\"redis123456\",\"REDIS_PORT\":\"6370\"}',1,'[2025-10-30 17:20:01] 开始部署 Redis Redis 7.2\n[2025-10-30 17:20:01] 连接主机 8.130.14.34...\n[2025-10-30 17:20:02] SSH连接成功\n[2025-10-30 17:20:03] 创建安装目录 /opt/data/redis...\n[2025-10-30 17:20:03] 读取模板文件 common/templates/05-redis/versions/redis-7.2-docker-compose.yml...\n[2025-10-30 17:20:03] 生成环境变量配置...\n[2025-10-30 17:20:03] 上传 docker-compose.yml...\n[2025-10-30 17:20:04] 上传 .env...\n[2025-10-30 17:20:04] 检查Docker环境...\n[2025-10-30 17:20:04] 启动服务容器...\n[2025-10-30 17:20:05] 容器启动输出:\n\n[2025-10-30 17:20:05] 验证容器状态...\n[2025-10-30 17:20:08] 容器状态:\nNAME      IMAGE                                                                                      COMMAND                  SERVICE   CREATED         STATUS                            PORTS\nredis72   crpi-aj3vgoxp9kzh2jx4.cn-hangzhou.personal.cr.aliyuncs.com/zhangfan_k8s/redis:7.2-alpine   \"docker-entrypoint.s…\"   redis     4 seconds ago   Up 3 seconds (health: starting)   0.0.0.0:6370->6379/tcp, [::]:6370->6379/tcp\n\n[2025-10-30 17:20:08] 部署完成！\n','2025-10-30 17:20:01.477','2025-10-30 17:20:01.477'),(11,'Java','java','java-17',501,'8.130.14.34','/opt/data/java','','','{\"APP_PORT\":\"8080\",\"JAVA_OPTS\":\"-Xmx512m -Xms256m\"}',1,'[2025-10-31 12:53:21] 开始部署 Java Java 17 LTS\n[2025-10-31 12:53:21] 连接主机 8.130.14.34...\n[2025-10-31 12:53:22] SSH连接成功\n[2025-10-31 12:53:22] 检查Docker环境...\n[2025-10-31 12:53:22] 使用镜像: crpi-aj3vgoxp9kzh2jx4.cn-hangzhou.personal.cr.aliyuncs.com/zhangfan_k8s/openjdk:17-jdk\n[2025-10-31 12:53:22] 拉取镜像...\n[2025-10-31 12:53:23] 镜像拉取成功\n[2025-10-31 12:53:23] 创建临时容器...\n[2025-10-31 12:53:23] 提取文件 /usr/local/openjdk-17 -> /usr/local/java17...\n[2025-10-31 12:53:29] 清理临时容器...\n[2025-10-31 12:53:29] 读取安装脚本 common/templates/02-java/versions/java-17-install.sh...\n[2025-10-31 12:53:29] 上传安装脚本...\n[2025-10-31 12:53:29] 执行安装脚本...\n[2025-10-31 12:53:30] 安装脚本输出:\n===== Java 17 LTS 安装配置 =====\n安装路径: /usr/local/java17\n环境变量文件: /etc/profile.d/java17.sh\n环境变量已配置: /etc/profile.d/java17.sh\n===== 安装成功 =====\n\n[2025-10-31 12:53:30] 验证安装...\n[2025-10-31 12:53:30] 验证结果:\nopenjdk version \"17.0.0.1\" 2024-07-02\nOpenJDK Runtime Environment (build 17.0.0.1+2-3)\nOpenJDK 64-Bit Server VM (build 17.0.0.1+2-3, mixed mode, sharing)\n\n[2025-10-31 12:53:30] 部署完成！\n','2025-10-31 12:53:21.350','2025-10-31 12:53:21.350'),(12,'Elasticsearch','elasticsearch','elasticsearch-8.x',506,'139.9.205.38','/opt/data/elasticsearch','','','{\"ES_HEAP_SIZE\":\"1g\",\"ES_HTTP_PORT\":\"9200\"}',3,'[2025-11-30 22:33:06] 开始部署 Elasticsearch Elasticsearch 8.x\n[2025-11-30 22:33:06] 连接主机 139.9.205.38...\n[2025-11-30 22:33:07] SSH连接成功\n[2025-11-30 22:33:07] 创建安装目录 /opt/data/elasticsearch...\n[2025-11-30 22:33:07] 读取模板文件 common/templates/06-elasticsearch/versions/elasticsearch-8.x-docker-compose.yml...\n[2025-11-30 22:33:07] 生成环境变量配置...\n[2025-11-30 22:33:07] 上传 docker-compose.yml...\n[2025-11-30 22:33:07] 上传 .env...\n[2025-11-30 22:33:07] 检查Docker环境...\n[2025-11-30 22:33:08] 启动服务容器...\n[2025-11-30 22:33:08] 启动失败: Process exited with status 127\n输出: \nSTDERR:\nbash: line 1: docker-compose: command not found\n\n','2025-11-30 22:33:06.149','2025-11-30 22:33:06.149');
/*!40000 ALTER TABLE `tool_service_deploy` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Final view structure for view `db_instance_all`
--

/*!50001 DROP VIEW IF EXISTS `db_instance_all`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`root`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `db_instance_all` AS select `db_instance`.`id` AS `id`,`db_instance`.`code` AS `code`,`db_instance`.`name` AS `name`,(case when (`db_instance`.`type` = 'postgres') then 'postgres' else 'mysql' end) AS `db_type`,`db_instance`.`type` AS `sub_type`,`db_instance`.`host` AS `host`,`db_instance`.`port` AS `port`,`db_instance`.`username` AS `username`,`db_instance`.`password` AS `password`,`db_instance`.`remark` AS `remark`,`db_instance`.`ssh_tunnel_machine_id` AS `ssh_tunnel_machine_id`,`db_instance`.`status` AS `status`,`db_instance`.`create_time` AS `create_time`,`db_instance`.`update_time` AS `update_time`,`db_instance`.`creator` AS `creator`,`db_instance`.`creator_id` AS `creator_id`,`db_instance`.`modifier` AS `modifier`,`db_instance`.`modifier_id` AS `modifier_id`,json_object('network',`db_instance`.`network`,'params',`db_instance`.`params`) AS `connection_config` from `db_instance` union all select `db_redis_instance`.`id` AS `id`,`db_redis_instance`.`code` AS `code`,`db_redis_instance`.`name` AS `name`,'redis' AS `db_type`,`db_redis_instance`.`mode` AS `sub_type`,`db_redis_instance`.`host` AS `host`,`db_redis_instance`.`port` AS `port`,`db_redis_instance`.`username` AS `username`,`db_redis_instance`.`password` AS `password`,`db_redis_instance`.`remark` AS `remark`,`db_redis_instance`.`ssh_tunnel_machine_id` AS `ssh_tunnel_machine_id`,`db_redis_instance`.`status` AS `status`,`db_redis_instance`.`create_time` AS `create_time`,`db_redis_instance`.`update_time` AS `update_time`,`db_redis_instance`.`creator` AS `creator`,`db_redis_instance`.`creator_id` AS `creator_id`,`db_redis_instance`.`modifier` AS `modifier`,`db_redis_instance`.`modifier_id` AS `modifier_id`,json_object('mode',`db_redis_instance`.`mode`,'db',`db_redis_instance`.`db`) AS `connection_config` from `db_redis_instance` union all select `db_mongo_instance`.`id` AS `id`,`db_mongo_instance`.`code` AS `code`,`db_mongo_instance`.`name` AS `name`,'mongodb' AS `db_type`,`db_mongo_instance`.`type` AS `sub_type`,NULL AS `host`,NULL AS `port`,NULL AS `username`,NULL AS `password`,`db_mongo_instance`.`remark` AS `remark`,`db_mongo_instance`.`ssh_tunnel_machine_id` AS `ssh_tunnel_machine_id`,`db_mongo_instance`.`status` AS `status`,`db_mongo_instance`.`create_time` AS `create_time`,`db_mongo_instance`.`update_time` AS `update_time`,`db_mongo_instance`.`creator` AS `creator`,`db_mongo_instance`.`creator_id` AS `creator_id`,`db_mongo_instance`.`modifier` AS `modifier`,`db_mongo_instance`.`modifier_id` AS `modifier_id`,json_object('uri',`db_mongo_instance`.`uri`) AS `connection_config` from `db_mongo_instance` union all select `db_es_instance`.`id` AS `id`,`db_es_instance`.`code` AS `code`,`db_es_instance`.`name` AS `name`,'elasticsearch' AS `db_type`,`db_es_instance`.`protocol` AS `sub_type`,`db_es_instance`.`host` AS `host`,`db_es_instance`.`port` AS `port`,`db_es_instance`.`username` AS `username`,`db_es_instance`.`password` AS `password`,`db_es_instance`.`remark` AS `remark`,`db_es_instance`.`ssh_tunnel_machine_id` AS `ssh_tunnel_machine_id`,`db_es_instance`.`status` AS `status`,`db_es_instance`.`create_time` AS `create_time`,`db_es_instance`.`update_time` AS `update_time`,`db_es_instance`.`creator` AS `creator`,`db_es_instance`.`creator_id` AS `creator_id`,`db_es_instance`.`modifier` AS `modifier`,`db_es_instance`.`modifier_id` AS `modifier_id`,json_object('protocol',`db_es_instance`.`protocol`) AS `connection_config` from `db_es_instance` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-07-04 23:26:31
