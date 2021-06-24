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
)

type PostgresqlInsertRecognizer struct {
	originalSQL string
	insertStmt  *ast.InsertStmt
}

func NewPostgresqlInsertRecognizer(originalSQL string, insertStmt *ast.InsertStmt) *PostgresqlInsertRecognizer {
	recognizer := &PostgresqlInsertRecognizer{
		originalSQL: originalSQL,
		insertStmt:  insertStmt,
	}
	return recognizer
}

func (recognizer *PostgresqlInsertRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_INSERT
}

func (recognizer *PostgresqlInsertRecognizer) GetTableAlias() string {
	return ""
}

func (recognizer *PostgresqlInsertRecognizer) GetTableName() string {
	var sb strings.Builder
	recognizer.insertStmt.Table.TableRefs.Left.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	return sb.String()
}

func (recognizer *PostgresqlInsertRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *PostgresqlInsertRecognizer) GetInsertColumns() []string {
	result := make([]string, 0)
	for _, col := range recognizer.insertStmt.Columns {
		result = append(result, col.Name.String())
	}
	return result
}

func (recognizer *PostgresqlInsertRecognizer) GetInsertRows() [][]string {
	var rows = make([][]string, 0)
	for _, dataRow := range recognizer.insertStmt.Lists {
		var row = make([]string, 0)
		for _, dataField := range dataRow {
			var sb strings.Builder
			dataField.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
			row = append(row, sb.String())
		}
		rows = append(rows, row)
	}

	return rows
}
