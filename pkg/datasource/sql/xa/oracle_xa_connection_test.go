/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xa

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
)

func TestOracleXAConn_Commit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid      string
		onePhase bool
	}

	tests := []struct {
		name    string
		input   args
		wantErr bool
	}{
		{
			name: "normal commit",
			input: args{
				xid:      "xid",
				onePhase: false,
			},
			wantErr: false,
		},
		{
			name: "commit with one phase",
			input: args{
				xid:      "xid",
				onePhase: true,
			},
			wantErr: false,
		},
		{
			name: "xid is nil",
			input: args{
				onePhase: false,
			},
			wantErr: true,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			// check if the xid is nil
			xidSplits := strings.Split(strings.Trim(query, " "), " ")
			if len(xidSplits) < 3 {
				return nil, errors.New("xid is nil")
			}
			if xidSplits[2] == "''" {
				return nil, errors.New("xid is nil")
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: mockConn,
			}
			if err := c.Commit(context.Background(), tt.input.xid, tt.input.onePhase); (err != nil) != tt.wantErr {
				t.Errorf("Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_End(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid   string
		flags int
	}
	tests := []struct {
		name    string
		input   args
		wantErr bool
	}{
		{
			name: "tm success",
			input: args{
				xid:   "xid",
				flags: TMSuccess,
			},
			wantErr: false,
		},
		{
			name: "tm failed",
			input: args{
				xid:   "xid",
				flags: TMFail,
			},
			wantErr: false,
		},
		{
			name: "tm suspend",
			input: args{
				xid:   "xid",
				flags: TMSuspend,
			},
			wantErr: false,
		},
		{
			name: "invalid flag",
			input: args{
				xid:   "xid",
				flags: 999,
			},
			wantErr: true,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: mockConn,
			}
			if err := c.End(context.Background(), tt.input.xid, tt.input.flags); (err != nil) != tt.wantErr {
				t.Errorf("End() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid   string
		flags int
	}
	tests := []struct {
		name    string
		input   args
		wantErr bool
	}{
		{
			name: "normal start",
			input: args{
				xid:   "xid",
				flags: TMNoFlags,
			},
			wantErr: false,
		},
		{
			name: "start with join",
			input: args{
				xid:   "xid",
				flags: TMJoin,
			},
			wantErr: false,
		},
		{
			name: "start with resume",
			input: args{
				xid:   "xid",
				flags: TMResume,
			},
			wantErr: false,
		},
		{
			name: "invalid flag",
			input: args{
				xid:   "xid",
				flags: 999,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := mock.NewMockTestDriverConn(ctrl)
			mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)

			c := &OracleXAConn{
				Conn: mockConn,
			}
			if err := c.Start(context.Background(), tt.input.xid, tt.input.flags); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_XAPrepare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid string
	}
	tests := []struct {
		name    string
		input   args
		wantErr bool
	}{
		{
			name: "normal prepare",
			input: args{
				xid: "xid",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := mock.NewMockTestDriverConn(ctrl)
			mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)

			c := &OracleXAConn{
				Conn: mockConn,
			}
			if err := c.XAPrepare(context.Background(), tt.input.xid); (err != nil) != tt.wantErr {
				t.Errorf("XAPrepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_Rollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid string
	}
	tests := []struct {
		name    string
		input   args
		wantErr bool
	}{
		{
			name: "normal rollback",
			input: args{
				xid: "xid",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := mock.NewMockTestDriverConn(ctrl)
			mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)

			c := &OracleXAConn{
				Conn: mockConn,
			}
			if err := c.Rollback(context.Background(), tt.input.xid); (err != nil) != tt.wantErr {
				t.Errorf("Rollback() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_Forget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid string
	}
	tests := []struct {
		name    string
		input   args
		wantErr bool
	}{
		{
			name: "normal forget",
			input: args{
				xid: "xid",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := mock.NewMockTestDriverConn(ctrl)
			mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)

			c := &OracleXAConn{
				Conn: mockConn,
			}
			if err := c.Forget(context.Background(), tt.input.xid); (err != nil) != tt.wantErr {
				t.Errorf("Forget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_Recover(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		flag int
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "normal recover",
			args: args{
				flag: TMStartRScan | TMEndRScan,
			},
			want:    []string{"xid", "another_xid"},
			wantErr: false,
		},
		{
			name: "invalid flag for recover",
			args: args{
				flag: TMFail,
			},
			wantErr: true,
		},
		{
			name: "valid flag for recover but don't scan",
			args: args{
				flag: TMEndRScan,
			},
			want:    nil,
			wantErr: false,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			rows := &oracleMockRows{}
			rows.data = [][]interface{}{
				{1, 3, 0, "xid"},
				{2, 11, 0, "another_xid"},
			}
			return rows, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: mockConn,
			}
			got, err := c.Recover(context.Background(), tt.args.flag)
			if (err != nil) != tt.wantErr {
				t.Errorf("Recover() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Recover() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOracleXAConn_GetTransactionTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{
			name:    "default timeout",
			timeout: 0,
			want:    0,
		},
		{
			name:    "custom timeout",
			timeout: 30 * time.Second,
			want:    30 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				transactionTimeout: tt.timeout,
			}
			if got := c.GetTransactionTimeout(); got != tt.want {
				t.Errorf("GetTransactionTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOracleXAConn_SetTransactionTimeout(t *testing.T) {
	type args struct {
		duration time.Duration
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "set timeout",
			args: args{
				duration: 30 * time.Second,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{}
			if got := c.SetTransactionTimeout(tt.args.duration); got != tt.want {
				t.Errorf("SetTransactionTimeout() = %v, want %v", got, tt.want)
			}
			if c.GetTransactionTimeout() != tt.args.duration {
				t.Errorf("GetTransactionTimeout() = %v, want %v", c.GetTransactionTimeout(), tt.args.duration)
			}
		})
	}
}

func TestOracleXAConn_IsSameRM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)

	type args struct {
		resource XAResource
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "different resource manager",
			args: args{
				resource: &OracleXAConn{Conn: mockConn},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: mockConn,
			}
			if got := c.IsSameRM(context.Background(), tt.args.resource); got != tt.want {
				t.Errorf("IsSameRM() = %v, want %v", got, tt.want)
			}
		})
	}
}

type oracleMockRows struct {
	idx  int
	data [][]interface{}
}

func (m *oracleMockRows) Columns() []string {
	//TODO implement me
	panic("implement me")
}

func (m *oracleMockRows) Close() error {
	//TODO implement me
	panic("implement me")
}

func (m *oracleMockRows) Next(dest []driver.Value) error {
	if m.idx == len(m.data) {
		return io.EOF
	}

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}
	cnt := min(len(m.data[0]), len(dest))
	for i := 0; i < cnt; i++ {
		dest[i] = m.data[m.idx][i]
	}
	m.idx++
	return nil
}
