package postgresql

import (
	"strings"
)

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/sqlparser"
	"github.com/transaction-wg/seata-golang/pkg/util/sql"
)

type PostgresqlUpdateRecognizer struct {
	originalSQL string
	updateStmt  *ast.UpdateStmt
}

func NewPostgresqlUpdateRecognizer(originalSQL string, updateStmt *ast.UpdateStmt) *PostgresqlUpdateRecognizer {
	recognizer := &PostgresqlUpdateRecognizer{
		originalSQL: originalSQL,
		updateStmt:  updateStmt,
	}
	return recognizer
}

func (recognizer *PostgresqlUpdateRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_UPDATE
}

func (recognizer *PostgresqlUpdateRecognizer) GetTableAlias() string {
	return ""
}

func (recognizer *PostgresqlUpdateRecognizer) GetTableName() string {
	var sb strings.Builder
	recognizer.updateStmt.TableRefs.TableRefs.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *PostgresqlUpdateRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *PostgresqlUpdateRecognizer) GetWhereCondition() string {
	var sb strings.Builder
	recognizer.updateStmt.Where.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	condition := strings.Replace(sb.String(), "`", "", -1)
	return sql.ConvertParamOrder(condition)
}

func (recognizer *PostgresqlUpdateRecognizer) GetUpdateColumns() []string {
	columns := make([]string, 0)
	for _, assignment := range recognizer.updateStmt.List {
		columns = append(columns, assignment.Column.Name.String())
	}
	return columns
}

func (recognizer *PostgresqlUpdateRecognizer) GetUpdateValues() []string {
	values := make([]string, 0)
	for _, assignment := range recognizer.updateStmt.List {
		var sb strings.Builder
		assignment.Expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
		values = append(values, sb.String())
	}
	return values
}
