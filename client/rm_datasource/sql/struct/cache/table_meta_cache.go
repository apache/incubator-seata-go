package cache

import (
	"database/sql"
	_struct "github.com/dk-lockdown/seata-golang/client/rm_datasource/sql/struct"
)

type ITableMetaCache interface {
	GetTableMeta(db *sql.DB,tableName,resourceId string) (_struct.TableMeta,error)

	Refresh(db *sql.DB,resourceId string)
}
