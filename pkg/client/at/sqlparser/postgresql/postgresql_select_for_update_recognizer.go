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

type PostgresqlSelectForUpdateRecognizer struct {
	originalSQL string
	selectStmt  *ast.SelectStmt
}

func NewPostgresqlSelectForUpdateRecognizer(originalSQL string, selectStmt *ast.SelectStmt) *PostgresqlSelectForUpdateRecognizer {
	recognizer := &PostgresqlSelectForUpdateRecognizer{
		originalSQL: originalSQL,
		selectStmt:  selectStmt,
	}
	return recognizer
}

func (recognizer *PostgresqlSelectForUpdateRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_SELECT
}

func (recognizer *PostgresqlSelectForUpdateRecognizer) GetTableAlias() string {
	table := recognizer.selectStmt.From.TableRefs.Left.(*ast.TableSource)
	return table.AsName.String()
}

func (recognizer *PostgresqlSelectForUpdateRecognizer) GetTableName() string {
	var sb strings.Builder
	table := recognizer.selectStmt.From.TableRefs.Left.(*ast.TableSource)
	table.Source.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *PostgresqlSelectForUpdateRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *PostgresqlSelectForUpdateRecognizer) GetWhereCondition() string {
	var sb strings.Builder
	recognizer.selectStmt.Where.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	condition := strings.Replace(sb.String(), "`", "", -1)
	return sql.ConvertParamOrder(condition)
}
