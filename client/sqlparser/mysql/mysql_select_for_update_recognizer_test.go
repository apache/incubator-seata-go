package mysql

import (
	"database/sql"
	"testing"
)

import (
	_ "github.com/pingcap/parser/test_driver"
	"github.com/stretchr/testify/assert"
)

func TestMysqlSelectForUpdateRecognizer_GetTableName(t *testing.T) {
	recognizer := NewMysqlSelectForUpdateRecognizer("select * from user u where u.id = ? for update",&sql.Stmt{})
	assert.Equal(t,"`user`",recognizer.GetTableName())
}

func TestMysqlSelectForUpdateRecognizer_GetTableAlias(t *testing.T) {
	recognizer := NewMysqlSelectForUpdateRecognizer("select * from user u where u.id = ? for update",&sql.Stmt{})
	assert.Equal(t,"u",recognizer.GetTableAlias())
}