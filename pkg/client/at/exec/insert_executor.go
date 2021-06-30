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
	sql2 "github.com/transaction-wg/seata-golang/pkg/util/sql"
	stringUtil "github.com/transaction-wg/seata-golang/pkg/util/string"
)

type InsertExecutor struct {
	proxyTx       *proxy_tx.ProxyTx
	sqlRecognizer sqlparser.ISQLInsertRecognizer
	values        []interface{}
}

func (executor *InsertExecutor) Execute() (sql.Result, error) {
	beforeImage, err := executor.BeforeImage()
	if err != nil {
		return nil, err
	}
	result, err := executor.proxyTx.Exec(executor.sqlRecognizer.GetOriginalSQL(), executor.values...)
	if err != nil {
		return result, err
	}

	afterImage, err := executor.AfterImage(result)

	if err != nil {
		return nil, err
	}
	executor.PrepareUndoLog(beforeImage, afterImage)
	return result, err
}

func (executor *InsertExecutor) PrepareUndoLog(beforeImage, afterImage *schema.TableRecords) {
	if len(afterImage.Rows) == 0 {
		return
	}

	var lockKeyRecords = afterImage

	lockKeys := buildLockKey(lockKeyRecords)
	executor.proxyTx.AppendLockKey(lockKeys)

	sqlUndoLog := buildUndoItem(executor.sqlRecognizer, beforeImage, afterImage)
	executor.proxyTx.AppendUndoLog(sqlUndoLog)
}

func (executor *InsertExecutor) BeforeImage() (*schema.TableRecords, error) {
	return nil, nil
}

func (executor *InsertExecutor) AfterImage(result sql.Result) (*schema.TableRecords, error) {
	var afterImage *schema.TableRecords
	var err error
	pkValues := executor.GetPkValuesByColumn()
	if executor.GetPkIndex() >= 0 {
		afterImage, err = executor.BuildTableRecords(pkValues)
	} else {
		pk, _ := result.LastInsertId()
		afterImage, err = executor.BuildTableRecords([]interface{}{pk})
	}
	if err != nil {
		return nil, err
	}
	return afterImage, nil
}

func (executor *InsertExecutor) BuildTableRecords(pkValues []interface{}) (*schema.TableRecords, error) {
	tableMeta, err := executor.getTableMeta()
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	fmt.Fprint(&sb, "SELECT ")
	var i = 0
	columnCount := len(tableMeta.Columns)
	for _, column := range tableMeta.Columns {
		fmt.Fprint(&sb, mysql.CheckAndReplace(column))
		i = i + 1
		if i < columnCount {
			fmt.Fprint(&sb, ",")
		} else {
			fmt.Fprint(&sb, " ")
		}
	}
	//todo 先根据不同数据库进行一个if判断
	if executor.proxyTx.DBType == constant.POSTGRESQL {
		fmt.Fprintf(&sb, "FROM %s ", stringUtil.Escape(executor.sqlRecognizer.GetTableName(), "`"))
		fmt.Fprintf(&sb, " WHERE %s IN ", tableMeta.GetPkName())
		fmt.Fprint(&sb, sql2.AppendInParamPostgres(len(pkValues)))
	} else {
		fmt.Fprintf(&sb, "FROM %s ", executor.sqlRecognizer.GetTableName())
		fmt.Fprintf(&sb, " WHERE %s IN ", tableMeta.GetPkName())
		fmt.Fprint(&sb, sql2.AppendInParam(len(pkValues)))
	}

	rows, err := executor.proxyTx.Query(sb.String(), pkValues...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return schema.BuildRecords(tableMeta, rows), nil
}

func (executor *InsertExecutor) GetPkValuesByColumn() []interface{} {
	pkValues := make([]interface{}, 0)
	columnLen := executor.GetColumnLen()
	pkIndex := executor.GetPkIndex()
	for i, value := range executor.values {
		if i%columnLen == pkIndex {
			pkValues = append(pkValues, value)
		}
	}
	return pkValues
}

func (executor *InsertExecutor) GetPkIndex() int {
	insertColumns := executor.sqlRecognizer.GetInsertColumns()
	tableMeta, _ := executor.getTableMeta()

	if insertColumns != nil && len(insertColumns) > 0 {
		for i, columnName := range insertColumns {
			if strings.EqualFold(tableMeta.GetPkName(), columnName) {
				return i
			}
		}
	} else {
		allColumns := tableMeta.Columns
		var idx = 0
		for _, column := range allColumns {
			if strings.EqualFold(tableMeta.GetPkName(), column) {
				return idx
			}
			idx = idx + 1
		}
	}
	return -1
}

func (executor *InsertExecutor) GetColumnLen() int {
	insertColumns := executor.sqlRecognizer.GetInsertColumns()
	if insertColumns != nil {
		return len(insertColumns)
	}
	tableMeta, _ := executor.getTableMeta()

	return len(tableMeta.Columns)
}

func (executor *InsertExecutor) getTableMeta() (schema.TableMeta, error) {
	tableMetaCache := extension.GetTableMetaCache(executor.proxyTx.DBType)
	return tableMetaCache.GetTableMeta(executor.proxyTx.Tx, executor.sqlRecognizer.GetTableName(), executor.proxyTx.ResourceID)
}
