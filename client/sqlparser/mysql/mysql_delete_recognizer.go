package mysql

import (
	"database/sql"
	"strings"
)

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
)

import (
	"github.com/dk-lockdown/seata-golang/client/sqlparser"
)

type MysqlDeleteRecognizer struct {
	originalSQL string
	stmt *sql.Stmt
	deleteStmt *ast.DeleteStmt
}

func NewMysqlDeleteRecognizer(originalSQL string,stat *sql.Stmt) *MysqlDeleteRecognizer {
	recognizer := &MysqlDeleteRecognizer{
		originalSQL: originalSQL,
		stmt:        stat,
	}

	act,_ := parser.ParseOneStmt(recognizer.originalSQL,"","")
	recognizer.deleteStmt,_ = act.(*ast.DeleteStmt)
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
	recognizer.deleteStmt.TableRefs.TableRefs.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags,&sb))
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