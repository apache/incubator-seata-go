package mysql

import (
	"database/sql"
	"fmt"
	"testing"
)

import (
	_ "github.com/pingcap/parser/test_driver"
	"github.com/stretchr/testify/assert"
)

func TestMysqlUpdateRecognizer_GetTableName(t *testing.T) {
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?",&sql.Stmt{})
	assert.Equal(t,"`user`",recognizer.GetTableName())
}

func TestMysqlUpdateRecognizer_GetWhereCondition(t *testing.T) {
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?",&sql.Stmt{})
	whereCondition := recognizer.GetWhereCondition()
	fmt.Println(whereCondition)
	assert.NotEmpty(t,whereCondition)
}

func TestMysqlUpdateRecognizer_GetUpdateColumns(t *testing.T) {
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?",&sql.Stmt{})
	columns := recognizer.GetUpdateColumns()
	fmt.Println(columns)
	assert.Equal(t,2,len(columns))
}

func TestMysqlUpdateRecognizer_GetUpdateValues(t *testing.T) {
	recognizer := NewMysqlUpdateRecognizer("update user set gender = ?,age = ? where name = ?",&sql.Stmt{})
	values := recognizer.GetUpdateValues()
	fmt.Println(values)
	assert.Equal(t,2,len(values))
}