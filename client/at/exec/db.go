package exec

import (
	"database/sql"
	tx2 "github.com/dk-lockdown/seata-golang/client/at/tx"
	"github.com/dk-lockdown/seata-golang/client/config"
	"github.com/dk-lockdown/seata-golang/client/context"
	"strings"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
)

type DB struct{
	*sql.DB
	conf        config.ATConfig
	ResourceGroupId string
}

func NewDB(conf config.ATConfig) (*DB,error) {
	db, err := sql.Open("mysql",conf.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(conf.Active)
	db.SetMaxIdleConns(conf.Idle)
	db.SetConnMaxLifetime(conf.IdleTimeout)
	newDB := &DB{
		DB:              db,
		conf:            conf,
		ResourceGroupId: "",
	}
	dataSourceManager.RegisterResource(newDB)
	return newDB,nil
}

func (db *DB) GetResourceGroupId() string {
	return db.ResourceGroupId
}

func (db *DB) GetResourceId() string {
	fromIndex := strings.Index(db.conf.DSN,"@")
	endIndex := strings.Index(db.conf.DSN,"?")
	return db.conf.DSN[fromIndex:endIndex]
}

func (db *DB) GetBranchType() meta.BranchType {
	return meta.BranchTypeAT
}

func (db *DB) Begin(ctx *context.RootContext) (*Tx,error) {
	tx,err := db.DB.Begin()
	if err != nil {
		return nil,err
	}
	proxyTx := &tx2.ProxyTx{
		Tx:         tx,
		DSN:        db.conf.DSN,
		ResourceId: db.GetResourceId(),
		Context:    tx2.NewTxContext(ctx),
	}
	return &Tx{
		tx: proxyTx,
		reportRetryCount: db.conf.ReportRetryCount,
		reportSuccessEnable: db.conf.ReportSuccessEnable,
	},nil
}