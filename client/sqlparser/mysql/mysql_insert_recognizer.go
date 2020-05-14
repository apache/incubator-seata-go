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

type MysqlInsertRecognizer struct {
	originalSQL string
	stmt *sql.Stmt
	insertStmt *ast.InsertStmt
}

func NewMysqlInsertRecognizer(originalSQL string,stat *sql.Stmt) *MysqlInsertRecognizer {
	recognizer := &MysqlInsertRecognizer{
		originalSQL: originalSQL,
		stmt:        stat,
	}

	act,_ := parser.ParseOneStmt(recognizer.originalSQL,"","")
	recognizer.insertStmt,_ = act.(*ast.InsertStmt)
	return recognizer
}

func (recognizer *MysqlInsertRecognizer) GetSQLType() sqlparser.SQLType {
	return sqlparser.SQLType_INSERT
}

func (recognizer *MysqlInsertRecognizer) GetTableAlias() string {
	return ""
}

func (recognizer *MysqlInsertRecognizer) GetTableName() string {
	var sb strings.Builder
	recognizer.insertStmt.Table.TableRefs.Left.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags,&sb))
	return sb.String()
}

func (recognizer *MysqlInsertRecognizer) GetOriginalSQL() string {
	return recognizer.originalSQL
}

func (recognizer *MysqlInsertRecognizer) GetInsertColumns() []string {
	result := make([]string,0)
	for _,col := range recognizer.insertStmt.Columns {
		result = append(result,col.Name.String())
	}
	return result
}

func (recognizer *MysqlInsertRecognizer) GetInsertRows() [][]string {
	var rows = make([][]string,0)
	for _,dataRow := range recognizer.insertStmt.Lists {
		var row = make([]string,0)
		for _,dataField := range dataRow {
			var sb strings.Builder
			dataField.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags,&sb))
			row = append(row,sb.String())
		}
		rows = append(rows,row)
	}

	return rows
}
