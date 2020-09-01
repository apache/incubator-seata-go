package mysql

import (
	p "github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"testing"
)

import (
	_ "github.com/pingcap/parser/test_driver"
	"github.com/stretchr/testify/assert"
)

func TestMysqlSelectForUpdateRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("select * from user u where u.id = ? for update", "", "")
	stmt := act.(*ast.SelectStmt)
	recognizer := NewMysqlSelectForUpdateRecognizer("select * from user u where u.id = ? for update", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestMysqlSelectForUpdateRecognizer_GetTableAlias(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("select * from user u where u.id = ? for update", "", "")
	stmt := act.(*ast.SelectStmt)
	recognizer := NewMysqlSelectForUpdateRecognizer("select * from user u where u.id = ? for update", stmt)
	assert.Equal(t, "u", recognizer.GetTableAlias())
}
