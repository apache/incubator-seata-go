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

	"github.com/golang/mock/gomock"

	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
)

func TestMysqlXAConn_Commit(t *testing.T) {
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
			if len(xidSplits) != 3 {
				return nil, errors.New("xid is nil")
			}
			if xidSplits[2] == "''" {
				return nil, errors.New("xid is nil")
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MysqlXAConn{
				Conn: mockConn,
			}
			if err := c.Commit(context.Background(), tt.input.xid, tt.input.onePhase); (err != nil) != tt.wantErr {
				t.Errorf("Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMysqlXAConn_End(t *testing.T) {
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
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MysqlXAConn{
				Conn: mockConn,
			}
			if err := c.End(context.Background(), tt.input.xid, tt.input.flags); (err != nil) != tt.wantErr {
				t.Errorf("End() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMysqlXAConn_Start(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := mock.NewMockTestDriverConn(ctrl)
			mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)

			c := &MysqlXAConn{
				Conn: mockConn,
			}
			if err := c.Start(context.Background(), tt.input.xid, tt.input.flags); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMysqlXAConn_XAPrepare(t *testing.T) {
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

			c := &MysqlXAConn{
				Conn: mockConn,
			}
			if err := c.XAPrepare(context.Background(), tt.input.xid); (err != nil) != tt.wantErr {
				t.Errorf("XAPrepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMysqlXAConn_Recover(t *testing.T) {
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
			rows := &mysqlMockRows{}
			rows.data = [][]interface{}{
				{1, 3, 0, "xid"},
				{2, 11, 0, "another_xid"},
			}
			return rows, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MysqlXAConn{
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

type mysqlMockRows struct {
	idx  int
	data [][]interface{}
}

func (m *mysqlMockRows) Columns() []string {
	//TODO implement me
	panic("implement me")
}

func (m *mysqlMockRows) Close() error {
	//TODO implement me
	panic("implement me")
}

func (m *mysqlMockRows) Next(dest []driver.Value) error {
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
