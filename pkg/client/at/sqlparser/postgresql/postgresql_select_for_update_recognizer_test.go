package postgresql

import (
	"testing"
)

import (
	p "github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/parser/test_driver"
	"github.com/stretchr/testify/assert"
)

func TestPostgresqlSelectForUpdateRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("select * from user u where u.id = $1 for update", "", "")
	stmt := act.(*ast.SelectStmt)
	recognizer := NewPostgresqlSelectForUpdateRecognizer("select * from user u where u.id = $2 for update", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestPostgresqlSelectForUpdateRecognizer_GetTableAlias(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("select * from user u where u.id = $1 for update", "", "")
	stmt := act.(*ast.SelectStmt)
	recognizer := NewPostgresqlSelectForUpdateRecognizer("select * from user u where u.id = $2 for update", stmt)
	assert.Equal(t, "u", recognizer.GetTableAlias())
}
