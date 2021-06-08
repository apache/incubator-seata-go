package cache

import (
	"github.com/patrickmn/go-cache"
)

type PostgresqlTableMetaCache struct {
	MysqlTableMetaCache
	tableMetaCache *cache.Cache
	dsn            string
}

func NewPostgresqlTableMetaCache(dsn string) ITableMetaCache {
	return &PostgresqlTableMetaCache{
		tableMetaCache: cache.New(EXPIRE_TIME, 10*EXPIRE_TIME),
		dsn:            dsn,
	}
}
