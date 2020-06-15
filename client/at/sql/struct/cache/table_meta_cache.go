package cache

import (
	"database/sql"

	_struct "github.com/xiaobudongzhang/seata-golang/client/at/sql/struct"
)

type ITableMetaCache interface {
	GetTableMeta(tx *sql.Tx, tableName, resourceId string) (_struct.TableMeta, error)

	Refresh(tx *sql.Tx, resourceId string)
}

var tableMetaCache ITableMetaCache

func SetTableMetaCache(tbMetaCache ITableMetaCache) {
	tableMetaCache = tbMetaCache
}

func GetTableMetaCache() ITableMetaCache {
	return tableMetaCache
}
