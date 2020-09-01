package mysql

import (
	"fmt"
	p "github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"testing"
)

import (
	_ "github.com/pingcap/parser/test_driver"
	"github.com/stretchr/testify/assert"
)

func TestNewMysqlInsertRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("insert into user (name,age) values (?,?)", "", "")
	stmt := act.(*ast.InsertStmt)
	recognizer := NewMysqlInsertRecognizer("insert into user (name,age) values (?,?)", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestNewMysqlInsertRecognizer_GetInsertColumns(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("insert into user (name,age) values (?,?)", "", "")
	stmt := act.(*ast.InsertStmt)
	recognizer := NewMysqlInsertRecognizer("insert into user (name,age) values (?,?)", stmt)
	columns := recognizer.GetInsertColumns()
	fmt.Println(columns)
	assert.Equal(t, 2, len(columns))
}

func TestNewMysqlInsertRecognizer_GetInsertRows(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("insert into user (name,age) values (?,?)", "", "")
	stmt := act.(*ast.InsertStmt)
	recognizer := NewMysqlInsertRecognizer("insert into user (name,age) values (?,?)", stmt)
	values := recognizer.GetInsertRows()
	fmt.Println(values)
	assert.Equal(t, 1, len(values))
}
