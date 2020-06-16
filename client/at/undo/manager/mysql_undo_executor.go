package manager

import (
	"database/sql"
	"fmt"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/sql/schema"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
	"github.com/dk-lockdown/seata-golang/client/at/undo"
)

const (
	InsertSqlTemplate = "INSERT INTO %s (%s) VALUES (%s)"
	DeleteSqlTemplate = "DELETE FROM %s WHERE `%s` = ?"
	UpdateSqlTemplate = "UPDATE %s SET %s WHERE `%s` = ?"
)

type BuildUndoSql func(undoLog undo.SqlUndoLog) string

func DeleteBuildUndoSql(undoLog undo.SqlUndoLog) string {
	beforeImage := undoLog.BeforeImage
	beforeImageRows := beforeImage.Rows

	if beforeImageRows == nil || len(beforeImageRows)==0 {
		panic(errors.New("invalid undo log"))
	}

	row := beforeImageRows[0]
	fields := row.NonPrimaryKeys()
	pkField := row.PrimaryKeys()[0]
	// PK is at last one.
	fields = append(fields,pkField)

	var sb,sb1 strings.Builder
	var size = len(fields)
	for i,field := range fields {
		fmt.Fprintf(&sb,"`%s`", field.Name)
		fmt.Fprint(&sb1,"?")
		if i < size-1 {
			fmt.Fprint(&sb,", ")
			fmt.Fprint(&sb1,", ")
		}
	}
	insertColumns := sb.String()
	insertValues := sb.String()

	return fmt.Sprintf(InsertSqlTemplate,undoLog.TableName,insertColumns,insertValues)
}

func InsertBuildUndoSql(undoLog undo.SqlUndoLog) string {
	afterImage := undoLog.AfterImage
	afterImageRows := afterImage.Rows
	if afterImageRows == nil || len(afterImageRows)==0 {
		panic(errors.New("invalid undo log"))
	}
	row := afterImageRows[0]
	pkField := row.PrimaryKeys()[0]
	return fmt.Sprintf(DeleteSqlTemplate,undoLog.TableName,pkField.Name)
}

func UpdateBuildUndoSql(undoLog undo.SqlUndoLog) string {
	beforeImage := undoLog.BeforeImage
	beforeImageRows := beforeImage.Rows

	if beforeImageRows == nil || len(beforeImageRows)==0 {
		panic(errors.New("invalid undo log"))
	}

	row := beforeImageRows[0]
	nonPkFields := row.NonPrimaryKeys()
	pkField := row.PrimaryKeys()[0]

	var sb strings.Builder
	var size = len(nonPkFields)
	for i,field := range nonPkFields {
		fmt.Fprintf(&sb,"`%s` = ?",field.Name)
		if i < size-1 {
			fmt.Fprint(&sb, ", ")
		}
	}
	updateColumns := sb.String()

	return fmt.Sprintf(UpdateSqlTemplate,undoLog.TableName,updateColumns,pkField.Name)
}

type MysqlUndoExecutor struct {
	sqlUndoLog undo.SqlUndoLog
}

func NewMysqlUndoExecutor(undoLog undo.SqlUndoLog) MysqlUndoExecutor {
	return MysqlUndoExecutor{sqlUndoLog:undoLog}
}

func (executor MysqlUndoExecutor) Execute(tx *sql.Tx) error {
	var undoSql string
	var undoRows schema.TableRecords
	switch executor.sqlUndoLog.SqlType {
	case sqlparser.SQLType_INSERT:
		undoSql = InsertBuildUndoSql(executor.sqlUndoLog)
		undoRows = *executor.sqlUndoLog.AfterImage
		break
	case sqlparser.SQLType_DELETE:
		undoSql = DeleteBuildUndoSql(executor.sqlUndoLog)
		undoRows = *executor.sqlUndoLog.BeforeImage
		break
	case sqlparser.SQLType_UPDATE:
		undoSql = UpdateBuildUndoSql(executor.sqlUndoLog)
		undoRows = *executor.sqlUndoLog.BeforeImage
		break
	default:
		panic(errors.Errorf("unsupport sql type:%s",executor.sqlUndoLog.SqlType.String()))
	}

	// PK is at last one.
	// INSERT INTO a (x, y, z, pk) VALUES (?, ?, ?, ?)
	// UPDATE a SET x=?, y=?, z=? WHERE pk = ?
	// DELETE FROM a WHERE pk = ?
	//todo 后镜数据和当前数据比较，判断是否可以回滚数据
	stmt,err := tx.Prepare(undoSql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _,row := range undoRows.Rows {
		var args = make([]interface{},0)
		var pkValue interface{}

		for _, field := range row.Fields {
			if field.KeyType == schema.PRIMARY_KEY {
				pkValue = field.Value
			} else {
				if executor.sqlUndoLog.SqlType != sqlparser.SQLType_INSERT {
					args = append(args, field.Value)
				}
			}
		}
		args = append(args,pkValue)
		_,err = stmt.Exec(args...)
		if err != nil {
			return err
		}
	}
	return nil
}