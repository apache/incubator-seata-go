package at

import (
	"database/sql/driver"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMultiDeleteExecutor(t *testing.T) {
	executor := NewMultiDeleteExecutor(nil, nil, nil)
	_, ok := executor.(*multiDeleteExecutor)
	assert.Equalf(t, true, ok, "should be *multiDeleteExecutor")
}

func Test_multiDeleteExecutor_buildBeforeImageSQL(t *testing.T) {
	log.Init()
	var (
		builder = multiDeleteExecutor{}
	)
	tests := []struct {
		name            string
		sourceQuery     []string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     []string{"delete from table_update_executor_test where id = ?", "delete from table_update_executor_test"},
			sourceQueryArgs: []driver.Value{3},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test FOR UPDATE",
			expectQueryArgs: []driver.Value{},
		},
		{
			sourceQuery:     []string{"delete from table_update_executor_test2 where id = ?", "delete from table_update_executor_test2 where id = ?"},
			sourceQueryArgs: []driver.Value{3, 2},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test2 WHERE id=? OR id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{3, 2},
		},
		{
			sourceQuery:     []string{"delete from table_update_executor_test2 where id = ?", "delete from table_update_executor_test2 where name = ? and age = ?"},
			sourceQueryArgs: []driver.Value{3, "seata-go", 4},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test2 WHERE id=? OR name=? AND age=? FOR UPDATE",
			expectQueryArgs: []driver.Value{3, "seata-go", 4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := builder.buildBeforeImageSQL(tt.sourceQuery, util.ValueToNamedValue(tt.sourceQueryArgs))
			assert.Nil(t, err)
			assert.Equal(t, 1, len(query))
			assert.Equal(t, query[0], tt.expectQuery)
			assert.Equal(t, util.ValueToNamedValue(tt.expectQueryArgs), args)
		})
	}
}
