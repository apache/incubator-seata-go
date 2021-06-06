package exec

import (
	"database/sql"
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"

	"strings"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	tx2 "github.com/transaction-wg/seata-golang/pkg/client/at/proxy_tx"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema/cache"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	"github.com/transaction-wg/seata-golang/pkg/client/context"
)

type DB struct {
	*sql.DB
	conf            config.ATConfig
	ResourceGroupID string
}

func NewDB(conf config.ATConfig, db *sql.DB) (*DB, error) {
	newDB := &DB{
		DB:              db,
		conf:            conf,
		ResourceGroupID: "",
	}
	cache.SetTableMetaCache(cache.NewMysqlTableMetaCache(conf.DSN))
	dataSourceManager.RegisterResource(newDB)
	return newDB, nil
}

func (db *DB) GetResourceGroupID() string {
	return db.ResourceGroupID
}

func (db *DB) GetDBType() string {
	dbType := db.conf.DBType
	if dbType == "" {
		db.conf.DBType = constant.MYSQL
	}
	return db.conf.DBType
}

func (db *DB) GetResourceID() string {
	dbType := db.GetDBType()
	if constant.POSTGRESQL == dbType {
		return db.getPGResourceId()
	} else {
		return db.getDefaultResourceId()
	}

}

//id生成准则？：连接信息+库
func (db *DB) getPGResourceId() string {
	return db.conf.DSN
}
func (db *DB) getDefaultResourceId() string {
	fromIndex := strings.Index(db.conf.DSN, "@")
	endIndex := strings.Index(db.conf.DSN, "?")
	return db.conf.DSN[fromIndex+1 : endIndex]
}

func (db *DB) GetBranchType() meta.BranchType {
	return meta.BranchTypeAT
}

func (db *DB) Begin(ctx *context.RootContext) (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	proxyTx := &tx2.ProxyTx{
		Tx:         tx,
		DSN:        db.conf.DSN,
		ResourceID: db.GetResourceID(),
		Context:    tx2.NewTxContext(ctx),
	}
	return &Tx{
		proxyTx:             proxyTx,
		reportRetryCount:    db.conf.ReportRetryCount,
		reportSuccessEnable: db.conf.ReportSuccessEnable,
		lockRetryInterval:   db.conf.LockRetryInterval,
		lockRetryTimes:      db.conf.LockRetryTimes,
	}, nil
}
