package mysql

import (
	"strings"
)

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
)

type MysqlDeleteRecognizer struct {
	originalSQL string
	deleteStmt  *ast.DeleteStmt
}

func NewMysqlDeleteRecognizer(originalSQL string, deleteStmt *ast.DeleteStmt) *MysqlDeleteRecognizer {
	recognizer := &MysqlDeleteRecognizer{
		originalSQL: originalSQL,
		deleteStmt:  deleteStmt,
	}
	return recognizer
}

func (recognizer *MysqlDeleteRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_DELETE
}

func (recognizer *MysqlDeleteRecognizer) GetTableAlias() string {
	return ""
}

func (recognizer *MysqlDeleteRecognizer) GetTableName() string {
	var sb strings.Builder
	recognizer.deleteStmt.TableRefs.TableRefs.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *MysqlDeleteRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *MysqlDeleteRecognizer) GetWhereCondition() string {
	var sb strings.Builder
	recognizer.deleteStmt.Where.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}
