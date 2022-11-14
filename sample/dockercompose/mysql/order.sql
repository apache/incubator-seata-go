CREATE database if NOT EXISTS `seata_client` default character set utf8mb4 collate utf8mb4_unicode_ci;
USE `seata_client`;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS  `order_tbl` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) DEFAULT NULL,
  `commodity_code` varchar(255) DEFAULT NULL,
  `count` int(11) DEFAULT '0',
  `money` int(11) DEFAULT '0',
  `descs` varchar(255) DEFAULT '',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO `seata_client`.`order_tbl` (`id`, `user_id`, `commodity_code`, `count`, `money`, `descs`) VALUES (1, 'NO-100001', 'C100000', 100, 10, 'init desc');