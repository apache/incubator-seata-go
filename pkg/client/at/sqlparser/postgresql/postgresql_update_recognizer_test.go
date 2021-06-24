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

func TestPostgresqlUpdateRecognizer_GetTableName(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = $1,age = $2 where name = $3", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewPostgresqlUpdateRecognizer("update user set gender = $1,age = $2 where name = $3", stmt)
	assert.Equal(t, "`user`", recognizer.GetTableName())
}

func TestPostgresqlUpdateRecognizer_GetWhereCondition(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = $1,age = $2 where name = $3", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewPostgresqlUpdateRecognizer("update user set gender = $1,age = $2 where name = $3", stmt)
	whereCondition := recognizer.GetWhereCondition()
	fmt.Println(whereCondition)
	assert.NotEmpty(t, whereCondition)
}

func TestPostgresqlUpdateRecognizer_GetUpdateColumns(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = $1,age = $2 where name = $3", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewPostgresqlUpdateRecognizer("update user set gender = $1,age = $2 where name = $3", stmt)
	columns := recognizer.GetUpdateColumns()
	fmt.Println(columns)
	assert.Equal(t, 2, len(columns))
}

func TestPostgresqlUpdateRecognizer_GetUpdateValues(t *testing.T) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt("update user set gender = $1,age = $2 where name = $3", "", "")
	stmt := act.(*ast.UpdateStmt)
	recognizer := NewPostgresqlUpdateRecognizer("update user set gender = $1,age = $2 where name = $3", stmt)
	values := recognizer.GetUpdateValues()
	fmt.Println(values)
	assert.Equal(t, 2, len(values))
}
