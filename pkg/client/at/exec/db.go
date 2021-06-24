package exec

import (
	"database/sql"
	"strings"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	tx2 "github.com/transaction-wg/seata-golang/pkg/client/at/proxy_tx"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema/cache/mysql"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema/cache/postgresql"
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
	//todo 先根据类型加载对应的表元数据缓存信息
	if constant.POSTGRESQL == newDB.GetDBType() {
		extension.SetTableMetaCache(newDB.GetDBType(), postgresql.NewPostgresqlTableMetaCache(conf.DSN))
	} else {
		extension.SetTableMetaCache(newDB.GetDBType(), mysql.NewMysqlTableMetaCache(conf.DSN))
	}
	dataSourceManager.RegisterResource(newDB)
	return newDB, nil
}

func (db *DB) GetResourceGroupID() string {
	return db.ResourceGroupID
}

func (db *DB) GetDBType() string {
	//DB类型获取，postgres连接指定采用postgres://的形式
	dbType := db.conf.DSN
	//判断是否存在指定字符串
	if strings.Contains(dbType, "postgres://") {
		return constant.POSTGRESQL
	}
	return constant.MYSQL
}

func (db *DB) GetResourceID() string {
	dbType := db.GetDBType()
	if constant.POSTGRESQL == dbType {
		return db.getPGResourceId()
	} else {
		return db.getDefaultResourceId()
	}

}

//pg资源id获取需要考虑schema
func (db *DB) getPGResourceId() string {
	dsn := db.conf.DSN

	if strings.Contains(dsn, "?") {
		var builder strings.Builder
		index := strings.Index(dsn, "?")
		builder.WriteString(dsn[0:index])
		paramUrl := dsn[index+1 : len(dsn)]
		urlParams := strings.Split(paramUrl, "&")
		var paramBuilder strings.Builder
		for _, param := range urlParams {
			if strings.Contains(param, "search_path") {
				paramBuilder.WriteString(param)
				break
			}
		}
		if paramBuilder.Len() > 0 {
			builder.WriteString("?")
			builder.WriteString(paramBuilder.String())
		}
		return builder.String()
	}
	return dsn
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
		DBType:     db.GetDBType(),
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
