package exec

import (
	"database/sql"

	"github.com/xiaobudongzhang/seata-golang/base/meta"
	"github.com/xiaobudongzhang/seata-golang/client/at/sql/struct/cache"

	_struct "github.com/xiaobudongzhang/seata-golang/client/at/sql/struct"
	"github.com/xiaobudongzhang/seata-golang/client/at/sqlparser"
	"github.com/xiaobudongzhang/seata-golang/client/at/tx"
)

type SelectForUpdateExecutor struct {
	tx            *tx.ProxyTx
	sqlRecognizer sqlparser.ISQLSelectRecognizer
	values        []interface{}
}

func (executor *SelectForUpdateExecutor) Execute() (*sql.Rows, error) {
	tableMeta, err := executor.getTableMeta()
	if err != nil {
		return nil, err
	}
	rows, err := executor.tx.Query(executor.sqlRecognizer.GetOriginalSQL(), executor.values...)
	if err != nil {
		return nil, err
	}
	selectPKRows := _struct.BuildRecords(tableMeta, rows)
	lockKeys := buildLockKey(selectPKRows)
	if lockKeys == "" {
		return rows, err
	} else {
		if executor.tx.Context.InGlobalTransaction() {
			lockable, err := dataSourceManager.LockQuery(meta.BranchTypeAT,
				executor.tx.ResourceId, executor.tx.Context.Xid, lockKeys)
			if !lockable && err != nil {
				return nil, err
			}
		}
	}
	return rows, err
}

func (executor *SelectForUpdateExecutor) getTableMeta() (_struct.TableMeta, error) {
	tableMetaCache := cache.GetTableMetaCache()
	return tableMetaCache.GetTableMeta(executor.tx.Tx, executor.sqlRecognizer.GetTableName(), executor.tx.ResourceId)
}
