/*
 Navicat PostgreSQL Data Transfer

 Source Server         : postgres
 Source Server Type    : PostgreSQL
 Source Server Version : 100017
 Source Host           : 172.18.150.90:5432
 Source Catalog        : seata_order
 Source Schema         : public

 Target Server Type    : PostgreSQL
 Target Server Version : 100017
 File Encoding         : 65001

 Date: 04/06/2021 17:55:34
*/


-- ----------------------------
-- Sequence structure for undo_log_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."undo_log_id_seq";
CREATE SEQUENCE "public"."undo_log_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for undo_log_id_seq1
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."undo_log_id_seq1";
CREATE SEQUENCE "public"."undo_log_id_seq1" 
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
  "xid" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "branch_id" int8 NOT NULL,
  "args_json" varchar(512) COLLATE "pg_catalog"."default",
  "state" int2,
  "gmt_create" timestamp(6) NOT NULL,
  "gmt_modified" timestamp(6)
)
;

-- ----------------------------
-- Records of branch_transaction
-- ----------------------------

-- ----------------------------
-- Table structure for so_item
-- ----------------------------
DROP TABLE IF EXISTS "public"."so_item";
CREATE TABLE "public"."so_item" (
  "sysno" int8 NOT NULL,
  "so_sysno" int8,
  "product_sysno" int8,
  "product_name" varchar(64) COLLATE "pg_catalog"."default",
  "cost_price" numeric(16,6) DEFAULT NULL::numeric,
  "original_price" numeric(16,6) DEFAULT NULL::numeric,
  "deal_price" numeric(16,6) DEFAULT NULL::numeric,
  "quantity" int8
)
;

-- ----------------------------
-- Records of so_item
-- ----------------------------

-- ----------------------------
-- Table structure for so_master
-- ----------------------------
DROP TABLE IF EXISTS "public"."so_master";
CREATE TABLE "public"."so_master" (
  "sysno" int8 NOT NULL,
  "so_id" varchar(20) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "buyer_user_sysno" int8,
  "seller_company_code" varchar(20) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "receive_division_sysno" int8 NOT NULL,
  "receive_address" varchar(200) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "receive_zip" varchar(20) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "receive_contact" varchar(20) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "receive_contact_phone" varchar(100) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "stock_sysno" int8,
  "payment_type" int4,
  "so_amt" numeric(16,6) DEFAULT NULL::numeric,
  "status" int4,
  "order_date" timestamp(6),
  "paymemt_date" timestamp(6),
  "delivery_date" timestamp(6),
  "receive_date" timestamp(6),
  "appid" varchar(64) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "memo" varchar(255) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "create_user" varchar(255) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "gmt_create" timestamp(6),
  "modify_user" varchar(255) COLLATE "pg_catalog"."default" DEFAULT NULL::character varying,
  "gmt_modified" timestamp(6)
)
;

-- ----------------------------
-- Records of so_master
-- ----------------------------

-- ----------------------------
-- Table structure for undo_log
-- ----------------------------
DROP TABLE IF EXISTS "public"."undo_log";
CREATE TABLE "public"."undo_log" (
  "id" int4 NOT NULL DEFAULT nextval('undo_log_id_seq1'::regclass),
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
SELECT setval('"public"."undo_log_id_seq"', 2, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."undo_log_id_seq1"
OWNED BY "public"."undo_log"."id";
SELECT setval('"public"."undo_log_id_seq1"', 2, false);

-- ----------------------------
-- Indexes structure for table branch_transaction
-- ----------------------------
CREATE UNIQUE INDEX "xid" ON "public"."branch_transaction" USING btree (
  "xid" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "branch_id" "pg_catalog"."int8_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table branch_transaction
-- ----------------------------
ALTER TABLE "public"."branch_transaction" ADD CONSTRAINT "branch_transaction_pkey" PRIMARY KEY ("sysno");

-- ----------------------------
-- Primary Key structure for table so_item
-- ----------------------------
ALTER TABLE "public"."so_item" ADD CONSTRAINT "so_item_pkey" PRIMARY KEY ("sysno");

-- ----------------------------
-- Primary Key structure for table so_master
-- ----------------------------
ALTER TABLE "public"."so_master" ADD CONSTRAINT "so_master_pkey" PRIMARY KEY ("sysno");

-- ----------------------------
-- Uniques structure for table undo_log
-- ----------------------------
ALTER TABLE "public"."undo_log" ADD CONSTRAINT "ux_undo_log" UNIQUE ("xid", "branch_id");

-- ----------------------------
-- Primary Key structure for table undo_log
-- ----------------------------
ALTER TABLE "public"."undo_log" ADD CONSTRAINT "pk_undo_log" PRIMARY KEY ("id");
