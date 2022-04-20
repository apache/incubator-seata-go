-- -------------------------------- The script used when storeMode is 'db' --------------------------------

CREATE database seata;

-- the table to store GlobalSession data
CREATE TABLE IF NOT EXISTS global_table
(
  addressing varchar(128) NOT NULL,
  xid varchar(128) NOT NULL,
  transaction_id bigint DEFAULT NULL,
  transaction_name varchar(128) DEFAULT NULL,
  timeout int DEFAULT NULL,
  begin_time bigint DEFAULT NULL,
  status int NOT NULL,
  active bool NOT NULL,
  gmt_create timestamp DEFAULT NULL,
  gmt_modified timestamp DEFAULT NULL,
  PRIMARY KEY (xid)
);

CREATE INDEX idx_gmt_modified_status ON global_table(gmt_modified, status);
CREATE INDEX idx_transaction_id ON global_table(transaction_id);

-- the table to store BranchSession data
CREATE TABLE IF NOT EXISTS branch_table
(
  addressing varchar(128) NOT NULL,
  xid varchar(128) NOT NULL,
  branch_id bigint NOT NULL,
  transaction_id bigint DEFAULT NULL,
  resource_id varchar(256) DEFAULT NULL,
  lock_key VARCHAR(1000),
  branch_type varchar(8) DEFAULT NULL,
  status int DEFAULT NULL,
  application_data varchar(2000) DEFAULT NULL,
  async_commit tinyint NOT NULL DEFAULT 0,
  gmt_create timestamp DEFAULT NULL,
  gmt_modified timestamp DEFAULT NULL,
  PRIMARY KEY (branch_id)
);

CREATE INDEX idx_xid ON branch_table(xid);

-- the table to store lock data
CREATE TABLE IF NOT EXISTS lock_table
(
    row_key        VARCHAR(256) NOT NULL,
    xid            VARCHAR(128) NOT NULL,
    transaction_id BIGINT,
    branch_id      BIGINT       NOT NULL,
    resource_id    VARCHAR(256),
    table_name     VARCHAR(64),
    pk             VARCHAR(36),
    gmt_create     TIMESTAMP,
    gmt_modified   TIMESTAMP,
    PRIMARY KEY (row_key)
);

CREATE INDEX idx_branch_id ON lock_table(branch_id);
