package exec

import (
	"fmt"
	"strings"

	_struct "github.com/xiaobudongzhang/seata-golang/client/at/sql/struct"
	"github.com/xiaobudongzhang/seata-golang/client/at/sqlparser"
	"github.com/xiaobudongzhang/seata-golang/client/at/undo"
)

func buildLockKey(lockKeyRecords *_struct.TableRecords) string {
	if lockKeyRecords.Rows == nil || len(lockKeyRecords.Rows) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, lockKeyRecords.TableName)
	fmt.Fprint(&sb, ":")
	fields := lockKeyRecords.PkFields()
	length := len(fields)
	for i, field := range fields {
		fmt.Fprint(&sb, field.Value)
		if i < length-1 {
			fmt.Fprint(&sb, ",")
		}
	}
	return sb.String()
}

func buildUndoItem(recognizer sqlparser.ISQLRecognizer, beforeImage, afterImage *_struct.TableRecords) *undo.SqlUndoLog {
	sqlType := recognizer.GetSQLType()
	tableName := recognizer.GetTableName()

	sqlUndoLog := &undo.SqlUndoLog{
		SqlType:     sqlType,
		TableName:   tableName,
		BeforeImage: beforeImage,
		AfterImage:  afterImage,
	}
	return sqlUndoLog
}
