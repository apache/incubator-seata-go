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
	"github.com/dk-lockdown/seata-golang/base/mysql"
	_struct "github.com/dk-lockdown/seata-golang/client/at/sql/struct"
	"github.com/dk-lockdown/seata-golang/client/at/sql/struct/cache"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
	"github.com/dk-lockdown/seata-golang/client/at/tx"
)

type DeleteExecutor struct {
	tx            *tx.ProxyTx
	sqlRecognizer sqlparser.ISQLDeleteRecognizer
	values        []interface{}
}

func (executor *DeleteExecutor) Execute() (sql.Result, error) {
	beforeImage,err := executor.BeforeImage()
	if err != nil {
		return nil,err
	}
	result, err := executor.tx.Exec(executor.sqlRecognizer.GetOriginalSQL(),executor.values...)
	if err != nil {
		return result,err
	}
	afterImage,err := executor.AfterImage()
	if err != nil {
		return nil,err
	}
	executor.PrepareUndoLog(beforeImage,afterImage)
	return result,err
}

func (executor *DeleteExecutor) PrepareUndoLog(beforeImage, afterImage *_struct.TableRecords) {
	if len(beforeImage.Rows)== 0 {
		return
	}

	var lockKeyRecords = beforeImage

	lockKeys := buildLockKey(lockKeyRecords)
	executor.tx.AppendLockKey(lockKeys)

	sqlUndoLog := buildUndoItem(executor.sqlRecognizer,beforeImage,afterImage)
	executor.tx.AppendUndoLog(sqlUndoLog)
}

func (executor *DeleteExecutor) BeforeImage() (*_struct.TableRecords,error) {
	tableMeta,err := executor.getTableMeta()
	if err != nil {
		return nil,err
	}
	return executor.buildTableRecords(tableMeta)
}

func (executor *DeleteExecutor) AfterImage() (*_struct.TableRecords,error) {
	return nil,nil
}

func (executor *DeleteExecutor) getTableMeta() (_struct.TableMeta,error) {
	tableMetaCache := cache.GetTableMetaCache()
	return tableMetaCache.GetTableMeta(executor.tx.Tx,executor.sqlRecognizer.GetTableName(),executor.tx.ResourceId)
}

func (executor *DeleteExecutor) buildBeforeImageSql(tableMeta _struct.TableMeta) string {
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

func (executor *DeleteExecutor) buildTableRecords(tableMeta _struct.TableMeta) (*_struct.TableRecords,error) {
	rows,err := executor.tx.Query(executor.buildBeforeImageSql(tableMeta),executor.values...)
	if err != nil {
		return nil,errors.WithStack(err)
	}
	return _struct.BuildRecords(tableMeta,rows),nil
}