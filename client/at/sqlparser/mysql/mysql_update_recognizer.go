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

type MysqlUpdateRecognizer struct {
	originalSQL string
	updateStmt  *ast.UpdateStmt
}

func NewMysqlUpdateRecognizer(originalSQL string, updateStmt *ast.UpdateStmt) *MysqlUpdateRecognizer {
	recognizer := &MysqlUpdateRecognizer{
		originalSQL: originalSQL,
		updateStmt:  updateStmt,
	}
	return recognizer
}

func (recognizer *MysqlUpdateRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_UPDATE
}

func (recognizer *MysqlUpdateRecognizer) GetTableAlias() string {
	return ""
}

func (recognizer *MysqlUpdateRecognizer) GetTableName() string {
	var sb strings.Builder
	recognizer.updateStmt.TableRefs.TableRefs.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *MysqlUpdateRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *MysqlUpdateRecognizer) GetWhereCondition() string {
	var sb strings.Builder
	recognizer.updateStmt.Where.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *MysqlUpdateRecognizer) GetUpdateColumns() []string {
	columns := make([]string, 0)
	for _, assignment := range recognizer.updateStmt.List {
		columns = append(columns, assignment.Column.Name.String())
	}
	return columns
}

func (recognizer *MysqlUpdateRecognizer) GetUpdateValues() []string {
	values := make([]string, 0)
	for _, assignment := range recognizer.updateStmt.List {
		var sb strings.Builder
		assignment.Expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
		values = append(values, sb.String())
	}
	return values
}
