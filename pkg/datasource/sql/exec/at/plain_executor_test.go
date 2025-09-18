// pkg/datasource/sql/exec/at/plain_executor_test.go（修改后）
/*
 * 版权声明（保持不变）
 */

package at

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	mock "seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

// TestNewPlainExecutor
func TestNewPlainExecutor(t *testing.T) {
	executor := NewPlainExecutor(nil, nil)
	_, ok := executor.(*plainExecutor)
	assert.Equalf(t, true, ok, "should be *plainExecutor")
}

// TestPlainExecutor_ExecContext
func TestPlainExecutor_ExecContext(t *testing.T) {
	tests := []struct {
		name    string
		f       exec.CallbackWithNamedValue
		wantVal types.ExecResult
		wantErr error
	}{
		{
			name: "test1",
			f: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return mock.NewMockInsertResult(int64(1), int64(2)), nil
			},
			wantVal: mock.NewMockInsertResult(int64(1), int64(2)),
			wantErr: nil,
		},
		{
			name: "test2",
			f: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return nil, fmt.Errorf("test error")
			},
			wantVal: nil,
			wantErr: fmt.Errorf("test error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &plainExecutor{execContext: &types.ExecContext{}}
			val, err := u.ExecContext(context.Background(), tt.f)
			assert.Equalf(t, tt.wantVal, val, "")
			assert.Equalf(t, tt.wantErr, err, "")
		})
	}
}
