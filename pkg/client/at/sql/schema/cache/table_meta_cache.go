package cache

import (
	"database/sql"
	"time"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema"
)

var EXPIRE_TIME = 15 * time.Minute

type ITableMetaCache interface {
	GetTableMeta(tx *sql.Tx, tableName, resourceID string) (schema.TableMeta, error)

	Refresh(tx *sql.Tx, resourceID string)
}
