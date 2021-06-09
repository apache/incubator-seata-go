package extension

import "sync"
import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema/cache"
)

var (
	tableMetaCachesMu sync.RWMutex
	tableMetaCaches   = make(map[string]cache.ITableMetaCache)
)

func SetTableMetaCache(dbType string, tbMetaCache cache.ITableMetaCache) {
	tableMetaCachesMu.Lock()
	defer tableMetaCachesMu.Unlock()
	if _, dup := tableMetaCaches[dbType]; dup {
		panic("called twice for TableMetaCache " + dbType)
	}
	tableMetaCaches[dbType] = tbMetaCache
}

func GetTableMetaCache(dbType string) cache.ITableMetaCache {
	tableMetaCachesMu.RLock()
	tbMetaCache := tableMetaCaches[dbType]
	tableMetaCachesMu.RUnlock()
	return tbMetaCache
}
