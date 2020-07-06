package mysql

import (
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	"strings"
)

type MysqlSelectForUpdateRecognizer struct {
	originalSQL string
	selectStmt  *ast.SelectStmt
}

func NewMysqlSelectForUpdateRecognizer(originalSQL string, selectStmt *ast.SelectStmt) *MysqlSelectForUpdateRecognizer {
	recognizer := &MysqlSelectForUpdateRecognizer{
		originalSQL: originalSQL,
		selectStmt:  selectStmt,
	}
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
	table.Source.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *MysqlSelectForUpdateRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *MysqlSelectForUpdateRecognizer) GetWhereCondition() string {
	var sb strings.Builder
	recognizer.selectStmt.Where.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}
