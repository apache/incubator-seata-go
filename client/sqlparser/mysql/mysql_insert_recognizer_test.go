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

func TestNewMysqlInsertRecognizer_GetTableName(t *testing.T) {
	recognizer := NewMysqlInsertRecognizer("insert into user (name,age) values (?,?)",&sql.Stmt{})
	assert.Equal(t,"`user`",recognizer.GetTableName())
}

func TestNewMysqlInsertRecognizer_GetInsertColumns(t *testing.T) {
	recognizer := NewMysqlInsertRecognizer("insert into user (name,age) values (?,?)",&sql.Stmt{})
	columns := recognizer.GetInsertColumns()
	fmt.Println(columns)
	assert.Equal(t,2, len(columns))
}

func TestNewMysqlInsertRecognizer_GetInsertRows(t *testing.T) {
	recognizer := NewMysqlInsertRecognizer("insert into user (name,age) values (?,?)",&sql.Stmt{})
	values := recognizer.GetInsertRows()
	fmt.Println(values)
	assert.Equal(t,1, len(values))
}
