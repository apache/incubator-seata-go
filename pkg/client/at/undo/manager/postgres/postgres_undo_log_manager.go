package postgres

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/client/at/undo/manager"
	"github.com/transaction-wg/seata-golang/pkg/client/at/undo/manager/mysql"
)

func init() {
	extension.SetUndoLogManager(constant.POSTGRESQL, func() manager.UndoLogManager {
		return PostgresUndoLogManager{}
	})
}

type PostgresUndoLogManager struct {
	mysql.MysqlUndoLogManager
}
