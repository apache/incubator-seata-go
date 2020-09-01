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

func TestMysqlDeleteRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("delete from user where id > ?", "", "")
	stmt := act.(*ast.DeleteStmt)
	recognizer := NewMysqlDeleteRecognizer("delete from user where id > ?", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestMysqlDeleteRecognizer_GetWhereCondition(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("delete from user where id > ?", "", "")
	stmt := act.(*ast.DeleteStmt)
	recognizer := NewMysqlDeleteRecognizer("delete from user where id > ?", stmt)
	whereCondition := recognizer.GetWhereCondition()
	fmt.Println(whereCondition)
	assert.NotEmpty(t, whereCondition)
}
