package exec

import (
	"database/sql"
	"fmt"
	"github.com/dk-lockdown/seata-golang/client/at/sql/struct/cache"
	sql2 "github.com/dk-lockdown/seata-golang/pkg/sql"
	"github.com/pkg/errors"
	"strings"
)

import (
	"github.com/dk-lockdown/seata-golang/base/mysql"
	_struct "github.com/dk-lockdown/seata-golang/client/at/sql/struct"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
	"github.com/dk-lockdown/seata-golang/client/at/tx"
)

type UpdateExecutor struct {
	tx            *tx.ProxyTx
	sqlRecognizer sqlparser.ISQLUpdateRecognizer
	values        []interface{}
}

func (executor *UpdateExecutor) Execute() (sql.Result, error) {
	beforeImage,err := executor.BeforeImage()
	if err != nil {
		return nil,err
	}
	result, err := executor.tx.Exec(executor.sqlRecognizer.GetOriginalSQL(),executor.values...)
	if err != nil {
		return result,err
	}
	afterImage,err := executor.AfterImage(beforeImage)
	if err != nil {
		return nil,err
	}
	executor.PrepareUndoLog(beforeImage,afterImage)
	return result,err
}

func (executor *UpdateExecutor) PrepareUndoLog(beforeImage, afterImage *_struct.TableRecords) {
	if len(beforeImage.Rows)== 0 && len(afterImage.Rows)== 0 {
		return
	}

	var lockKeyRecords = afterImage

	lockKeys := buildLockKey(lockKeyRecords)
	executor.tx.AppendLockKey(lockKeys)

	sqlUndoLog := buildUndoItem(executor.sqlRecognizer,beforeImage,afterImage)
	executor.tx.AppendUndoLog(sqlUndoLog)
}

func (executor *UpdateExecutor) BeforeImage() (*_struct.TableRecords,error) {
	tableMeta,err := executor.getTableMeta()
	if err != nil {
		return nil,err
	}
	return executor.buildTableRecords(tableMeta)
}

func (executor *UpdateExecutor) AfterImage(beforeImage *_struct.TableRecords) (*_struct.TableRecords,error) {
	tableMeta,err := executor.getTableMeta()
	if err != nil {
		return nil,err
	}
	afterImageSql := executor.buildAfterImageSql(tableMeta,beforeImage)
	var args = make([]interface{},0)
	for _,field := range beforeImage.PkFields() {
		args = append(args,field.Value)
	}
	rows,err := executor.tx.Query(afterImageSql,args...)
	if err != nil {
		return nil,errors.WithStack(err)
	}
	return _struct.BuildRecords(tableMeta,rows),nil
}

func (executor *UpdateExecutor) getTableMeta() (_struct.TableMeta,error) {
	tableMetaCache := cache.GetTableMetaCache()
	return tableMetaCache.GetTableMeta(executor.tx.Tx,executor.sqlRecognizer.GetTableName(),executor.tx.ResourceId)
}

func (executor *UpdateExecutor) buildBeforeImageSql(tableMeta _struct.TableMeta) string {
	var b strings.Builder
	fmt.Fprint(&b,"SELECT ")
	var i = 0
	for _,columnMeta := range tableMeta.AllColumns {
		fmt.Fprint(&b, mysql.CheckAndReplace(columnMeta.ColumnName))
		i = i + 1
		if i != len(tableMeta.AllColumns) {
			fmt.Fprint(&b,",")
		} else {
			fmt.Fprint(&b," ")
		}
	}
	fmt.Fprintf(&b," FROM %s WHERE ",executor.sqlRecognizer.GetTableName())
	fmt.Fprint(&b,executor.sqlRecognizer.GetWhereCondition())
	fmt.Fprint(&b," FOR UPDATE")
	return b.String()
}

func (executor *UpdateExecutor) buildAfterImageSql(tableMeta _struct.TableMeta,beforeImage *_struct.TableRecords) string {
	var b strings.Builder
	fmt.Fprint(&b,"SELECT ")
	var i = 0
	columnCount := len(executor.sqlRecognizer.GetUpdateColumns())
	for _,columnName := range executor.sqlRecognizer.GetUpdateColumns() {
		fmt.Fprint(&b, mysql.CheckAndReplace(columnName))
		i = i + 1
		if i < columnCount {
			fmt.Fprint(&b,",")
		} else {
			fmt.Fprint(&b," ")
		}
	}
	fmt.Fprintf(&b," FROM %s ",executor.sqlRecognizer.GetTableName())
	fmt.Fprintf(&b,"WHERE `%s` IN",tableMeta.GetPkName())
	fmt.Fprint(&b,sql2.AppendInParam(len(beforeImage.PkFields())))
	return b.String()
}

func (executor *UpdateExecutor) buildTableRecords(tableMeta _struct.TableMeta) (*_struct.TableRecords,error) {
	sql := executor.buildBeforeImageSql(tableMeta)
	argsCount := strings.Count(sql,"?")
	rows,err := executor.tx.Query(sql,executor.values[len(executor.values)-argsCount:]...)
	if err != nil {
		return nil,errors.WithStack(err)
	}
	return _struct.BuildRecords(tableMeta,rows),nil
}