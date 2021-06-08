package cache

import (
	"database/sql"
	"sync"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema"
)

type ITableMetaCache interface {
	GetTableMeta(tx *sql.Tx, tableName, resourceID string) (schema.TableMeta, error)

	Refresh(tx *sql.Tx, resourceID string)
}

var (
	tableMetaCachesMu sync.RWMutex
	tableMetaCaches   = make(map[string]ITableMetaCache)
)

func SetTableMetaCache(dbType string, tbMetaCache ITableMetaCache) {
	tableMetaCachesMu.Lock()
	defer tableMetaCachesMu.Unlock()
	if _, dup := tableMetaCaches[dbType]; dup {
		panic("called twice for TableMetaCache " + dbType)
	}
	tableMetaCaches[dbType] = tbMetaCache
}

func GetTableMetaCache(dbType string) ITableMetaCache {
	tableMetaCachesMu.RLock()
	tbMetaCache := tableMetaCaches[dbType]
	tableMetaCachesMu.RUnlock()
	return tbMetaCache
}
