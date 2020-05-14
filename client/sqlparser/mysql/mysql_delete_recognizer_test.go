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

func TestMysqlDeleteRecognizer_GetTableName(t *testing.T) {
	recognizer := NewMysqlDeleteRecognizer("delete from user where id > ?",&sql.Stmt{})
	assert.Equal(t,"`user`",recognizer.GetTableName())
}

func TestMysqlDeleteRecognizer_GetWhereCondition(t *testing.T) {
	recognizer := NewMysqlDeleteRecognizer("delete from user where id > ?",&sql.Stmt{})
	whereCondition := recognizer.GetWhereCondition()
	fmt.Println(whereCondition)
	assert.NotEmpty(t,whereCondition)
}