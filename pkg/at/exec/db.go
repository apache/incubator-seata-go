package exec

import (
	"database/sql"
	"strings"

	"github.com/transaction-wg/seata-golang/base/meta"
	tx2 "github.com/transaction-wg/seata-golang/pkg/at/proxy_tx"
	"github.com/transaction-wg/seata-golang/pkg/config"
	"github.com/transaction-wg/seata-golang/pkg/context"

	"xorm.io/xorm"
)

type DB struct {
	*sql.DB
	eng             *xorm.Engine
	conf            config.ATConfig
	ResourceGroupId string
}

func NewWithStandard(conf config.ATConfig, stdDB *sql.DB) (*DB, error) {
	db := &DB{
		DB:              stdDB,
		conf:            conf,
		ResourceGroupId: "",
	}
	dataSourceManager.RegisterResource(db)
	return db, nil
}

func NewWithXORM(conf config.ATConfig, eng *xorm.Engine) (*DB, error) {
	db := &DB{
		eng:             eng,
		DB:              eng.DB().DB,
		conf:            conf,
		ResourceGroupId: "",
	}
	dataSourceManager.RegisterResource(db)
	return db, nil
}

func (db *DB) GetResourceGroupId() string {
	return db.ResourceGroupId
}

func (db *DB) GetResourceId() string {
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
		ResourceId: db.GetResourceId(),
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
