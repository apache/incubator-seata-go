package cache

import (
	"database/sql"
)

import (
	_struct "github.com/dk-lockdown/seata-golang/client/at/sql/struct"
)

type ITableMetaCache interface {
	GetTableMeta(tx *sql.Tx,tableName,resourceId string) (_struct.TableMeta,error)

	Refresh(tx *sql.Tx,resourceId string)
}

var tableMetaCache ITableMetaCache

func SetTableMetaCache(tbMetaCache ITableMetaCache) {
	tableMetaCache = tbMetaCache
}

func GetTableMetaCache() ITableMetaCache {
	return tableMetaCache
}