package mysql

import (
	"database/sql"
	"github.com/dk-lockdown/seata-golang/client/sqlparser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	"strings"
)

type MysqlSelectForUpdateRecognizer struct {
	originalSQL string
	stmt *sql.Stmt
	selectStmt *ast.SelectStmt
}

func NewMysqlSelectForUpdateRecognizer(originalSQL string,stat *sql.Stmt) *MysqlSelectForUpdateRecognizer {
	recognizer := &MysqlSelectForUpdateRecognizer{
		originalSQL: originalSQL,
		stmt:        stat,
	}

	act,_ := parser.ParseOneStmt(recognizer.originalSQL,"","")
	recognizer.selectStmt,_ = act.(*ast.SelectStmt)
	return recognizer
}

func (recognizer *MysqlSelectForUpdateRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_SELECT
}

func (recognizer *MysqlSelectForUpdateRecognizer) GetTableAlias() string {
	table := recognizer.selectStmt.From.TableRefs.Left.(*ast.TableSource)
	return table.AsName.String()
}

func (recognizer *MysqlSelectForUpdateRecognizer) GetTableName() string {
	var sb strings.Builder
	table := recognizer.selectStmt.From.TableRefs.Left.(*ast.TableSource)
	table.Source.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags,&sb))
	return sb.String()
}

func (recognizer *MysqlSelectForUpdateRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}