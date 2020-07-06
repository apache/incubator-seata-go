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

func TestMysqlUpdateRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = ?,age = ? where name = ?", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestMysqlUpdateRecognizer_GetWhereCondition(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = ?,age = ? where name = ?", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?", stmt)
	whereCondition := recognizer.GetWhereCondition()
	fmt.Println(whereCondition)
	assert.NotEmpty(t, whereCondition)
}

func TestMysqlUpdateRecognizer_GetUpdateColumns(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = ?,age = ? where name = ?", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?", stmt)
	columns := recognizer.GetUpdateColumns()
	fmt.Println(columns)
	assert.Equal(t, 2, len(columns))
}

func TestMysqlUpdateRecognizer_GetUpdateValues(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = ?,age = ? where name = ?", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?", stmt)
	values := recognizer.GetUpdateValues()
	fmt.Println(values)
	assert.Equal(t, 2, len(values))
}
