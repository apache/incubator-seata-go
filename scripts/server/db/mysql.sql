-- -------------------------------- The script used when storeMode is 'db' --------------------------------

CREATE database if NOT EXISTS `seata` default character set utf8mb4 collate utf8mb4_unicode_ci;
USE `seata`;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;
-- the table to store GlobalSession data
CREATE TABLE IF NOT EXISTS `global_table`
(
  `addressing` varchar(128) NOT NULL,
  `xid` varchar(128) NOT NULL,
  `transaction_id` bigint DEFAULT NULL,
  `transaction_name` varchar(128) DEFAULT NULL,
  `timeout` int DEFAULT NULL,
  `begin_time` bigint DEFAULT NULL,
  `status` tinyint NOT NULL,
  `active` bit(1) NOT NULL,
  `gmt_create` datetime DEFAULT NULL,
  `gmt_modified` datetime DEFAULT NULL,
  PRIMARY KEY (`xid`),
  KEY `idx_gmt_modified_status` (`gmt_modified`,`status`),
  KEY `idx_transaction_id` (`transaction_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8;

-- the table to store BranchSession data
CREATE TABLE IF NOT EXISTS `branch_table`
(
  `addressing` varchar(128) NOT NULL,
  `xid` varchar(128) NOT NULL,
  `branch_id` bigint NOT NULL,
  `transaction_id` bigint DEFAULT NULL,
  `resource_id` varchar(256) DEFAULT NULL,
  `lock_key` VARCHAR(1000),
  `branch_type` varchar(8) DEFAULT NULL,
  `status` tinyint DEFAULT NULL,
  `application_data` varchar(2000) DEFAULT NULL,
  `async_commit` tinyint NOT NULL DEFAULT 0,
  `gmt_create` datetime(6) DEFAULT NULL,
  `gmt_modified` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`branch_id`),
  KEY `idx_xid` (`xid`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8;

-- the table to store lock data
CREATE TABLE IF NOT EXISTS `lock_table`
(
    `row_key`        VARCHAR(256) NOT NULL,
    `xid`            VARCHAR(128) NOT NULL,
    `transaction_id` BIGINT,
    `branch_id`      BIGINT       NOT NULL,
    `resource_id`    VARCHAR(256),
    `table_name`     VARCHAR(64),
    `pk`             VARCHAR(36),
    `gmt_create`     DATETIME,
    `gmt_modified`   DATETIME,
    PRIMARY KEY (`row_key`),
    KEY `idx_branch_id` (`branch_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8;
