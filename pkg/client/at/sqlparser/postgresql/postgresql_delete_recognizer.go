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

type PostgresDeleteRecognizer struct {
	originalSQL string
	deleteStmt  *ast.DeleteStmt
}

func NewPostgresqlDeleteRecognizer(originalSQL string, deleteStmt *ast.DeleteStmt) *PostgresDeleteRecognizer {
	recognizer := &PostgresDeleteRecognizer{
		originalSQL: originalSQL,
		deleteStmt:  deleteStmt,
	}
	return recognizer
}

func (recognizer *PostgresDeleteRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_DELETE
}

func (recognizer *PostgresDeleteRecognizer) GetTableAlias() string {
	return ""
}

func (recognizer *PostgresDeleteRecognizer) GetTableName() string {
	var sb strings.Builder
	recognizer.deleteStmt.TableRefs.TableRefs.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *PostgresDeleteRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *PostgresDeleteRecognizer) GetWhereCondition() string {
	var sb strings.Builder
	recognizer.deleteStmt.Where.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	condition := strings.Replace(sb.String(), "`", "", -1)
	return sql.ConvertParamOrder(condition)
}
