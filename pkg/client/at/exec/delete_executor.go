package exec

import (
	"database/sql"
	"fmt"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/client/at/proxy_tx"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sqlparser"
	"github.com/transaction-wg/seata-golang/pkg/util/mysql"
	stringUtil "github.com/transaction-wg/seata-golang/pkg/util/string"
)

type DeleteExecutor struct {
	proxyTx       *proxy_tx.ProxyTx
	sqlRecognizer sqlparser.ISQLDeleteRecognizer
	values        []interface{}
}

func (executor *DeleteExecutor) Execute() (sql.Result, error) {
	beforeImage, err := executor.BeforeImage()
	if err != nil {
		return nil, err
	}
	result, err := executor.proxyTx.Exec(executor.sqlRecognizer.GetOriginalSQL(), executor.values...)
	if err != nil {
		return result, err
	}
	afterImage, err := executor.AfterImage()
	if err != nil {
		return nil, err
	}
	executor.PrepareUndoLog(beforeImage, afterImage)
	return result, err
}

func (executor *DeleteExecutor) PrepareUndoLog(beforeImage, afterImage *schema.TableRecords) {
	if len(beforeImage.Rows) == 0 {
		return
	}

	var lockKeyRecords = beforeImage

	lockKeys := buildLockKey(lockKeyRecords)
	executor.proxyTx.AppendLockKey(lockKeys)

	sqlUndoLog := buildUndoItem(executor.sqlRecognizer, beforeImage, afterImage)
	executor.proxyTx.AppendUndoLog(sqlUndoLog)
}

func (executor *DeleteExecutor) BeforeImage() (*schema.TableRecords, error) {
	tableMeta, err := executor.getTableMeta()
	if err != nil {
		return nil, err
	}
	return executor.buildTableRecords(tableMeta)
}

func (executor *DeleteExecutor) AfterImage() (*schema.TableRecords, error) {
	return nil, nil
}

func (executor *DeleteExecutor) getTableMeta() (schema.TableMeta, error) {
	tableMetaCache := extension.GetTableMetaCache(executor.proxyTx.DBType)
	return tableMetaCache.GetTableMeta(executor.proxyTx.Tx, executor.sqlRecognizer.GetTableName(), executor.proxyTx.ResourceID)
}

func (executor *DeleteExecutor) buildBeforeImageSql(tableMeta schema.TableMeta) string {
	var b strings.Builder
	fmt.Fprint(&b, "SELECT ")
	var i = 0
	columnCount := len(tableMeta.Columns)
	for _, column := range tableMeta.Columns {
		fmt.Fprint(&b, mysql.CheckAndReplace(column))
		i = i + 1
		if i < columnCount {
			fmt.Fprint(&b, ",")
		} else {
			fmt.Fprint(&b, " ")
		}
	}
	//todo 先根据不同数据库进行一个if判断
	if executor.proxyTx.DBType == constant.POSTGRESQL {
		fmt.Fprintf(&b, " FROM %s WHERE ", stringUtil.Escape(executor.sqlRecognizer.GetTableName(), "`"))
		fmt.Fprint(&b, executor.sqlRecognizer.GetWhereCondition())
		fmt.Fprint(&b, " FOR UPDATE")
	} else {
		fmt.Fprintf(&b, " FROM %s WHERE ", executor.sqlRecognizer.GetTableName())
		fmt.Fprint(&b, executor.sqlRecognizer.GetWhereCondition())
		fmt.Fprint(&b, " FOR UPDATE")
	}

	return b.String()
}

func (executor *DeleteExecutor) buildTableRecords(tableMeta schema.TableMeta) (*schema.TableRecords, error) {
	rows, err := executor.proxyTx.Query(executor.buildBeforeImageSql(tableMeta), executor.values...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return schema.BuildRecords(tableMeta, rows), nil
}
