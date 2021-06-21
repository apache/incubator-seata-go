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

type UpdateExecutor struct {
	proxyTx       *proxy_tx.ProxyTx
	sqlRecognizer sqlparser.ISQLUpdateRecognizer
	values        []interface{}
}

func (executor *UpdateExecutor) Execute() (sql.Result, error) {
	beforeImage, err := executor.BeforeImage()
	if err != nil {
		return nil, err
	}
	result, err := executor.proxyTx.Exec(executor.sqlRecognizer.GetOriginalSQL(), executor.values...)
	if err != nil {
		return result, err
	}
	afterImage, err := executor.AfterImage(beforeImage)
	if err != nil {
		return nil, err
	}
	executor.PrepareUndoLog(beforeImage, afterImage)
	return result, err
}

func (executor *UpdateExecutor) PrepareUndoLog(beforeImage, afterImage *schema.TableRecords) {
	if len(beforeImage.Rows) == 0 &&
		(afterImage == nil || len(afterImage.Rows) == 0) {
		return
	}

	var lockKeyRecords = afterImage

	lockKeys := buildLockKey(lockKeyRecords)
	executor.proxyTx.AppendLockKey(lockKeys)

	sqlUndoLog := buildUndoItem(executor.sqlRecognizer, beforeImage, afterImage)
	executor.proxyTx.AppendUndoLog(sqlUndoLog)
}

func (executor *UpdateExecutor) BeforeImage() (*schema.TableRecords, error) {
	tableMeta, err := executor.getTableMeta()
	if err != nil {
		return nil, err
	}
	return executor.buildTableRecords(tableMeta)
}

func (executor *UpdateExecutor) AfterImage(beforeImage *schema.TableRecords) (*schema.TableRecords, error) {
	if beforeImage.Rows == nil || len(beforeImage.Rows) == 0 {
		return nil, nil
	}

	tableMeta, err := executor.getTableMeta()
	if err != nil {
		return nil, err
	}
	afterImageSql := executor.buildAfterImageSql(tableMeta, beforeImage)
	var args = make([]interface{}, 0)
	for _, field := range beforeImage.PkFields() {
		args = append(args, field.Value)
	}
	rows, err := executor.proxyTx.Query(afterImageSql, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return schema.BuildRecords(tableMeta, rows), nil
}

func (executor *UpdateExecutor) getTableMeta() (schema.TableMeta, error) {
	tableMetaCache := extension.GetTableMetaCache(executor.proxyTx.DBType)
	return tableMetaCache.GetTableMeta(executor.proxyTx.Tx, executor.sqlRecognizer.GetTableName(), executor.proxyTx.ResourceID)
}

func (executor *UpdateExecutor) buildBeforeImageSql(tableMeta schema.TableMeta) string {
	var b strings.Builder
	fmt.Fprint(&b, "SELECT ")
	var i = 0
	columnCount := len(tableMeta.Columns)
	for _, column := range tableMeta.Columns {
		fmt.Fprint(&b, mysql.CheckAndReplace(column))
		i = i + 1
		if i != columnCount {
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

func (executor *UpdateExecutor) buildAfterImageSql(tableMeta schema.TableMeta, beforeImage *schema.TableRecords) string {
	var b strings.Builder
	fmt.Fprint(&b, "SELECT ")
	var i = 0
	columnCount := len(tableMeta.Columns)
	for _, columnName := range tableMeta.Columns {
		fmt.Fprint(&b, mysql.CheckAndReplace(columnName))
		i = i + 1
		if i < columnCount {
			fmt.Fprint(&b, ",")
		} else {
			fmt.Fprint(&b, " ")
		}
	}
	//todo 先根据不同数据库进行一个if判断
	if executor.proxyTx.DBType == constant.POSTGRESQL {
		fmt.Fprintf(&b, " FROM %s ", stringUtil.Escape(executor.sqlRecognizer.GetTableName(), "`"))
		fmt.Fprintf(&b, "WHERE %s IN", tableMeta.GetPkName())
		fmt.Fprint(&b, sql2.AppendInParamPostgres(len(beforeImage.PkFields())))
	} else {
		fmt.Fprintf(&b, " FROM %s ", executor.sqlRecognizer.GetTableName())
		fmt.Fprintf(&b, "WHERE `%s` IN", tableMeta.GetPkName())
		fmt.Fprint(&b, sql2.AppendInParam(len(beforeImage.PkFields())))
	}
	return b.String()
}

func (executor *UpdateExecutor) buildTableRecords(tableMeta schema.TableMeta) (*schema.TableRecords, error) {
	sql := executor.buildBeforeImageSql(tableMeta)
	var argsCount int
	if strings.Contains(sql, "?") {
		argsCount = strings.Count(sql, "?")
	} else {
		argsCount = strings.Count(sql, "$")
	}
	rows, err := executor.proxyTx.Query(sql, executor.values[len(executor.values)-argsCount:]...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return schema.BuildRecords(tableMeta, rows), nil
}
