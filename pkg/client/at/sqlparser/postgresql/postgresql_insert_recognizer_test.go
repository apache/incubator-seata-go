package postgresql

import (
	"fmt"
	"testing"
)

import (
	p "github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/parser/test_driver"
	"github.com/stretchr/testify/assert"
)

func TestNewPostgresqlInsertRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("insert into user (name,age) values ($1,$2)", "", "")
	stmt := act.(*ast.InsertStmt)
	recognizer := NewPostgresqlInsertRecognizer("insert into user (name,age) values ($1,$2)", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestNewPostgresqlInsertRecognizer_GetInsertColumns(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("insert into user (name,age) values ($1,$2)", "", "")
	stmt := act.(*ast.InsertStmt)
	recognizer := NewPostgresqlInsertRecognizer("insert into user (name,age) values ($1,$2)", stmt)
	columns := recognizer.GetInsertColumns()
	fmt.Println(columns)
	assert.Equal(t, 2, len(columns))
}

func TestNewPostgresqlInsertRecognizer_GetInsertRows(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("insert into user (name,age) values ($1,$2)", "", "")
	stmt := act.(*ast.InsertStmt)
	recognizer := NewPostgresqlInsertRecognizer("insert into user (name,age) values ($1,$2)", stmt)
	values := recognizer.GetInsertRows()
	fmt.Println(values)
	assert.Equal(t, 1, len(values))
}
