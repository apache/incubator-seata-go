/*
 Navicat PostgreSQL Data Transfer

 Source Server         : postgres
 Source Server Type    : PostgreSQL
 Source Server Version : 100017
 Source Host           : 172.18.150.90:5432
 Source Catalog        : seata
 Source Schema         : public

 Target Server Type    : PostgreSQL
 Target Server Version : 100017
 File Encoding         : 65001

 Date: 10/06/2021 17:08:29
*/


-- ----------------------------
-- Table structure for branch_table
-- ----------------------------
DROP TABLE IF EXISTS "public"."branch_table";
CREATE TABLE "public"."branch_table" (
  "branch_id" int8 NOT NULL,
  "xid" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "transaction_id" int8,
  "resource_group_id" varchar(32) COLLATE "pg_catalog"."default",
  "resource_id" varchar(256) COLLATE "pg_catalog"."default",
  "branch_type" varchar(8) COLLATE "pg_catalog"."default",
  "status" int4,
  "client_id" varchar(64) COLLATE "pg_catalog"."default",
  "application_data" varchar(2000) COLLATE "pg_catalog"."default",
  "gmt_create" timestamp(6),
  "gmt_modified" timestamp(6)
)
;

-- ----------------------------
-- Records of branch_table
-- ----------------------------

-- ----------------------------
-- Table structure for global_table
-- ----------------------------
DROP TABLE IF EXISTS "public"."global_table";
CREATE TABLE "public"."global_table" (
  "xid" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "transaction_id" int8,
  "status" int4 NOT NULL,
  "application_id" varchar(32) COLLATE "pg_catalog"."default",
  "transaction_service_group" varchar(32) COLLATE "pg_catalog"."default",
  "transaction_name" varchar(128) COLLATE "pg_catalog"."default",
  "timeout" int4,
  "begin_time" int8,
  "application_data" varchar(2000) COLLATE "pg_catalog"."default",
  "gmt_create" timestamp(6),
  "gmt_modified" timestamp(6)
)
;

-- ----------------------------
-- Records of global_table
-- ----------------------------

-- ----------------------------
-- Table structure for lock_table
-- ----------------------------
DROP TABLE IF EXISTS "public"."lock_table";
CREATE TABLE "public"."lock_table" (
  "row_key" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "xid" varchar(96) COLLATE "pg_catalog"."default",
  "transaction_id" int8,
  "branch_id" int8 NOT NULL,
  "resource_id" varchar(256) COLLATE "pg_catalog"."default",
  "table_name" varchar(32) COLLATE "pg_catalog"."default",
  "pk" varchar(36) COLLATE "pg_catalog"."default",
  "gmt_create" timestamp(6),
  "gmt_modified" timestamp(6)
)
;

-- ----------------------------
-- Records of lock_table
-- ----------------------------

-- ----------------------------
-- Indexes structure for table branch_table
-- ----------------------------
CREATE INDEX "idx_xid" ON "public"."branch_table" USING btree (
  "xid" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table branch_table
-- ----------------------------
ALTER TABLE "public"."branch_table" ADD CONSTRAINT "branch_table_pkey" PRIMARY KEY ("branch_id");

-- ----------------------------
-- Indexes structure for table global_table
-- ----------------------------
CREATE INDEX "idx_gmt_modified_status" ON "public"."global_table" USING btree (
  "status" "pg_catalog"."int4_ops" ASC NULLS LAST,
  "gmt_modified" "pg_catalog"."timestamp_ops" ASC NULLS LAST
);
CREATE INDEX "idx_transaction_id" ON "public"."global_table" USING btree (
  "transaction_id" "pg_catalog"."int8_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table global_table
-- ----------------------------
ALTER TABLE "public"."global_table" ADD CONSTRAINT "global_table_pkey" PRIMARY KEY ("xid");

-- ----------------------------
-- Indexes structure for table lock_table
-- ----------------------------
CREATE INDEX "idx_branch_id" ON "public"."lock_table" USING btree (
  "branch_id" "pg_catalog"."int8_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table lock_table
-- ----------------------------
ALTER TABLE "public"."lock_table" ADD CONSTRAINT "lock_table_pkey" PRIMARY KEY ("row_key");
