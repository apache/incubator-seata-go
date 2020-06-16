package exec

import (
	"database/sql"
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/client/at/sql/schema/cache"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/proxy_tx"
	"github.com/dk-lockdown/seata-golang/client/at/sql/schema"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
)

type SelectForUpdateExecutor struct {
	proxyTx       *proxy_tx.ProxyTx
	sqlRecognizer sqlparser.ISQLSelectRecognizer
	values        []interface{}
}

func (executor *SelectForUpdateExecutor) Execute() (*sql.Rows, error) {
	tableMeta,err := executor.getTableMeta()
	if err != nil {
		return nil,err
	}
	rows,err := executor.proxyTx.Query(executor.sqlRecognizer.GetOriginalSQL(),executor.values...)
	if err != nil {
		return nil,err
	}
	selectPKRows := schema.BuildRecords(tableMeta,rows)
	lockKeys := buildLockKey(selectPKRows)
	if lockKeys == "" {
		return rows,err
	} else {
		if executor.proxyTx.Context.InGlobalTransaction() {
			lockable,err := dataSourceManager.LockQuery(meta.BranchTypeAT,
				executor.proxyTx.ResourceId,executor.proxyTx.Context.Xid,lockKeys)
			if !lockable && err != nil {
				return nil,err
			}
		}
	}
	return rows,err
}

func (executor *SelectForUpdateExecutor) getTableMeta() (schema.TableMeta,error) {
	tableMetaCache := cache.GetTableMetaCache()
	return tableMetaCache.GetTableMeta(executor.proxyTx.Tx,executor.sqlRecognizer.GetTableName(),executor.proxyTx.ResourceId)
}