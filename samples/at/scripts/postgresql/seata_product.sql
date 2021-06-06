/*
 Navicat PostgreSQL Data Transfer

 Source Server         : postgres
 Source Server Type    : PostgreSQL
 Source Server Version : 100017
 Source Host           : 172.18.150.90:5432
 Source Catalog        : seata_product
 Source Schema         : public

 Target Server Type    : PostgreSQL
 Target Server Version : 100017
 File Encoding         : 65001

 Date: 04/06/2021 17:55:51
*/


-- ----------------------------
-- Sequence structure for undo_log_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."undo_log_id_seq";
CREATE SEQUENCE "public"."undo_log_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 2147483647
START 1
CACHE 1;

-- ----------------------------
-- Table structure for branch_transaction
-- ----------------------------
DROP TABLE IF EXISTS "public"."branch_transaction";
CREATE TABLE "public"."branch_transaction" (
  "sysno" int8 NOT NULL,
  "xid" varchar(128) COLLATE "pg_catalog"."default",
  "branch_id" int8 NOT NULL,
  "args_json" varchar(512) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "state" int4,
  "gmt_create" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gmt_modified" timestamp(6)
)
;

-- ----------------------------
-- Records of branch_transaction
-- ----------------------------

-- ----------------------------
-- Table structure for inventory
-- ----------------------------
DROP TABLE IF EXISTS "public"."inventory";
CREATE TABLE "public"."inventory" (
  "sysno" int8 NOT NULL,
  "product_sysno" int8 NOT NULL,
  "account_qty" int8,
  "available_qty" int8,
  "allocated_qty" int8,
  "adjust_locked_qty" int8
)
;

-- ----------------------------
-- Records of inventory
-- ----------------------------
INSERT INTO "public"."inventory" VALUES (1, 1, 1000000, 1000000, 0, 0);

-- ----------------------------
-- Table structure for product
-- ----------------------------
DROP TABLE IF EXISTS "public"."product";
CREATE TABLE "public"."product" (
  "sysno" int8 NOT NULL,
  "product_name" varchar(32) COLLATE "pg_catalog"."default",
  "product_title" varchar(32) COLLATE "pg_catalog"."default",
  "product_desc" varchar(2000) COLLATE "pg_catalog"."default",
  "product_desc_long" text COLLATE "pg_catalog"."default",
  "default_image_src" varchar(200) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "c3_sysno" int8,
  "barcode" varchar(30) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "length" int8,
  "width" int8,
  "height" int8,
  "weight" float8,
  "merchant_sysno" int8,
  "merchant_productid" varchar(20) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "status" int4 NOT NULL,
  "gmt_create" timestamp(6) NOT NULL,
  "create_user" varchar(32) COLLATE "pg_catalog"."default",
  "modify_user" varchar(32) COLLATE "pg_catalog"."default",
  "gmt_modified" timestamp(6) NOT NULL
)
;

-- ----------------------------
-- Records of product
-- ----------------------------
INSERT INTO "public"."product" VALUES (1, '刺力王', '从小喝到大的刺力王', '好喝好喝好好喝', '', 'https://img10.360buyimg.com/mobilecms/s500x500_jfs/t1/61921/34/1166/131384/5cf60a94E411eee07/1ee010f4142236c3.jpg', 0, '', 15, 5, 5, 5, 1, '', 1, '2019-05-28 03:36:17', '', '', '2019-06-06 01:37:36');

-- ----------------------------
-- Table structure for undo_log
-- ----------------------------
DROP TABLE IF EXISTS "public"."undo_log";
CREATE TABLE "public"."undo_log" (
  "id" int4 NOT NULL DEFAULT nextval('undo_log_id_seq'::regclass),
  "branch_id" int8 NOT NULL,
  "xid" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "context" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "rollback_info" bytea NOT NULL,
  "log_status" int4 NOT NULL,
  "log_created" timestamp(0) NOT NULL,
  "log_modified" timestamp(0) NOT NULL
)
;

-- ----------------------------
-- Records of undo_log
-- ----------------------------

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."undo_log_id_seq"
OWNED BY "public"."undo_log"."id";
SELECT setval('"public"."undo_log_id_seq"', 2, false);

-- ----------------------------
-- Uniques structure for table branch_transaction
-- ----------------------------
ALTER TABLE "public"."branch_transaction" ADD CONSTRAINT "xid" UNIQUE ("xid", "branch_id");

-- ----------------------------
-- Primary Key structure for table branch_transaction
-- ----------------------------
ALTER TABLE "public"."branch_transaction" ADD CONSTRAINT "branch_transaction_pkey" PRIMARY KEY ("sysno");

-- ----------------------------
-- Primary Key structure for table inventory
-- ----------------------------
ALTER TABLE "public"."inventory" ADD CONSTRAINT "inventory_pkey" PRIMARY KEY ("sysno");

-- ----------------------------
-- Primary Key structure for table product
-- ----------------------------
ALTER TABLE "public"."product" ADD CONSTRAINT "product_pkey" PRIMARY KEY ("sysno");

-- ----------------------------
-- Uniques structure for table undo_log
-- ----------------------------
ALTER TABLE "public"."undo_log" ADD CONSTRAINT "ux_undo_log" UNIQUE ("xid", "branch_id");

-- ----------------------------
-- Primary Key structure for table undo_log
-- ----------------------------
ALTER TABLE "public"."undo_log" ADD CONSTRAINT "pk_undo_log" PRIMARY KEY ("id");
