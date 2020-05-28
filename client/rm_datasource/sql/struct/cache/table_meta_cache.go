package cache

import (
	"database/sql"
)

import (
	_struct "github.com/dk-lockdown/seata-golang/client/rm_datasource/sql/struct"
)

type ITableMetaCache interface {
	GetTableMeta(tx *sql.Tx,tableName,resourceId string) (_struct.TableMeta,error)

	Refresh(tx *sql.Tx,resourceId string)
}

var tableMetaCache ITableMetaCache

func SetTableMetaCache(tableMetaCache ITableMetaCache) {
	tableMetaCache = tableMetaCache
}

func GetTableMetaCache() ITableMetaCache {
	return tableMetaCache
}