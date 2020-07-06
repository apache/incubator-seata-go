package exec

import (
	"database/sql"
	"strings"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	tx2 "github.com/dk-lockdown/seata-golang/client/at/proxy_tx"
	"github.com/dk-lockdown/seata-golang/client/config"
	"github.com/dk-lockdown/seata-golang/client/context"
)

type DB struct {
	*sql.DB
	conf            config.ATConfig
	ResourceGroupId string
}

func NewDB(conf config.ATConfig, db *sql.DB) (*DB, error) {
	newDB := &DB{
		DB:              db,
		conf:            conf,
		ResourceGroupId: "",
	}
	dataSourceManager.RegisterResource(newDB)
	return newDB, nil
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
