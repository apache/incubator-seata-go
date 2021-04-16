package exec

import (
	"database/sql"
	"time"

	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/client/at/proxy_tx"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema/cache"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sqlparser"
)

type SelectForUpdateExecutor struct {
	proxyTx           *proxy_tx.ProxyTx
	sqlRecognizer     sqlparser.ISQLSelectRecognizer
	values            []interface{}
	dataSourceManager *DataSourceManager
}

func (executor *SelectForUpdateExecutor) Execute(lockRetryInterval time.Duration, lockRetryTimes int) (*sql.Rows, error) {
	tableMeta, err := executor.getTableMeta()
	if err != nil {
		return nil, err
	}
	rows, err := executor.proxyTx.Query(executor.sqlRecognizer.GetOriginalSQL(), executor.values...)
	if err != nil {
		return nil, err
	}
	selectPKRows := schema.BuildRecords(tableMeta, rows)
	lockKeys := buildLockKey(selectPKRows)
	if lockKeys == "" {
		return rows, err
	} else {
		if executor.proxyTx.Context.InGlobalTransaction() {
			var lockable bool
			var err error
			for i := 0; i < lockRetryTimes; i++ {
				lockable, err = executor.dataSourceManager.LockQuery(meta.BranchTypeAT,
					executor.proxyTx.ResourceID, executor.proxyTx.Context.XID, lockKeys)
				if lockable && err == nil {
					break
				}
				time.Sleep(lockRetryInterval)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return rows, err
}

func (executor *SelectForUpdateExecutor) getTableMeta() (schema.TableMeta, error) {
	tableMetaCache := cache.GetTableMetaCache()
	return tableMetaCache.GetTableMeta(executor.proxyTx.Tx, executor.sqlRecognizer.GetTableName(), executor.proxyTx.ResourceID)
}
