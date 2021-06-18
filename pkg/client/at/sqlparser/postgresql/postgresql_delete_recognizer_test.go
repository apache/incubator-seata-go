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

func TestPostgresqlDeleteRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("delete from user where id > $1", "", "")
	stmt := act.(*ast.DeleteStmt)
	recognizer := NewPostgresqlDeleteRecognizer("delete from user where id > $1", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestPostgresqlDeleteRecognizer_GetWhereCondition(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("delete from user where id > $1", "", "")
	stmt := act.(*ast.DeleteStmt)
	recognizer := NewPostgresqlDeleteRecognizer("delete from user where id > $1", stmt)
	whereCondition := recognizer.GetWhereCondition()
	fmt.Println(whereCondition)
	assert.NotEmpty(t, whereCondition)
}
