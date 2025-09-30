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

type mockTxWithoutExecer struct{}

func (m *mockTxWithoutExecer) Commit() error   { return nil }
func (m *mockTxWithoutExecer) Rollback() error { return nil }

func verifySQLContains(query string, expected ...string) error {
	for _, exp := range expected {
		if !strings.Contains(query, exp) {
			return errors.New("sql missing expected part: " + exp)
		}
	}
	return nil
}

func TestPostgresqlXAConn_Commit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid      string
		onePhase bool
	}
	tests := []struct {
		name                   string
		input                  args
		wantErr                bool
		mockTxNotSupportExecer bool
		expectExecCall         bool
	}{
		{
			name: "normal commit (two-phase)",
			input: args{
				xid:      "test-xid-1",
				onePhase: false,
			},
			wantErr:                false,
			mockTxNotSupportExecer: false,
			expectExecCall:         true,
		},
		{
			name: "normal commit (one-phase)",
			input: args{
				xid:      "test-xid-2",
				onePhase: true,
			},
			wantErr:                false,
			mockTxNotSupportExecer: false,
			expectExecCall:         true,
		},
		{
			name: "one-phase commit with prepared transaction",
			input: args{
				xid:      "test-prepared-xid",
				onePhase: true,
			},
			wantErr:                false,
			mockTxNotSupportExecer: false,
			expectExecCall:         true,
		},
		{
			name: "invalid xid (empty)",
			input: args{
				xid:      "",
				onePhase: false,
			},
			wantErr:                true,
			mockTxNotSupportExecer: false,
			expectExecCall:         false,
		},
		{
			name: "invalid xid (too long)",
			input: args{
				xid:      strings.Repeat("a", 201),
				onePhase: false,
			},
			wantErr:                true,
			mockTxNotSupportExecer: false,
			expectExecCall:         false,
		},
		{
			name: "invalid xid (dangerous characters)",
			input: args{
				xid:      "test';DROP TABLE users;--",
				onePhase: false,
			},
			wantErr:                true,
			mockTxNotSupportExecer: false,
			expectExecCall:         false,
		},
		{
			name: "invalid xid (unsafe characters)",
			input: args{
				xid:      "test'injection",
				onePhase: false,
			},
			wantErr:                true,
			mockTxNotSupportExecer: false,
			expectExecCall:         false,
		},
		{
			name: "conn does not support ExecerContext",
			input: args{
				xid:      "test-xid-3",
				onePhase: false,
			},
			wantErr:                true,
			mockTxNotSupportExecer: true,
			expectExecCall:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := mock.NewMockTestDriverConn(ctrl)
			mockTx := mock.NewMockTestDriverTx(ctrl)

			if !tt.mockTxNotSupportExecer && tt.expectExecCall {
				mockConn.EXPECT().ExecContext(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).DoAndReturn(func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
					if tt.input.onePhase {
						if err := verifySQLContains(query, "COMMIT"); err != nil {
							return nil, err
						}
					} else {
						if err := verifySQLContains(query, "COMMIT PREPARED", tt.input.xid); err != nil {
							return nil, err
						}
					}
					return &driver.ResultNoRows, nil
				})
			}

			var conn driver.Conn = mockConn
			if tt.mockTxNotSupportExecer {
				conn = &mockConnWithoutExecer{}
			}

			c := &PostgresqlXAConn{Conn: conn, tx: mockTx}
			if tt.input.xid == "test-prepared-xid" {
				c.prepared = true
				c.preparedXid = tt.input.xid
			}
			err := c.Commit(context.Background(), tt.input.xid, tt.input.onePhase)
			if (err != nil) != tt.wantErr {
				t.Errorf("Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostgresqlXAConn_XAPrepare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid string
	}
	tests := []struct {
		name                   string
		input                  args
		wantErr                bool
		mockTxNotSupportExecer bool
		expectExecCall         bool
	}{
		{
			name: "normal prepare",
			input: args{
				xid: "test-prepare-xid",
			},
			wantErr:                false,
			mockTxNotSupportExecer: false,
			expectExecCall:         true,
		},
		{
			name: "invalid xid (empty)",
			input: args{
				xid: "",
			},
			wantErr:                true,
			mockTxNotSupportExecer: false,
			expectExecCall:         false,
		},
		{
			name: "invalid xid (dangerous characters)",
			input: args{
				xid: "test';DROP TABLE users;--",
			},
			wantErr:                true,
			mockTxNotSupportExecer: false,
			expectExecCall:         false,
		},
		{
			name: "tx does not support ExecerContext",
			input: args{
				xid: "test-prepare-xid-2",
			},
			wantErr:                true,
			mockTxNotSupportExecer: true,
			expectExecCall:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := mock.NewMockTestDriverConn(ctrl)

			var tx driver.Tx
			if tt.mockTxNotSupportExecer {
				tx = &mockTxWithoutExecer{}
			} else {
				mockTx := mock.NewMockTestDriverTx(ctrl)
				if tt.expectExecCall {
					mockTx.EXPECT().ExecContext(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).DoAndReturn(func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
						return &driver.ResultNoRows, verifySQLContains(query, "PREPARE TRANSACTION", tt.input.xid)
					})
				}
				tx = mockTx
			}

			c := &PostgresqlXAConn{Conn: mockConn, tx: tx}
			err := c.XAPrepare(context.Background(), tt.input.xid)
			if (err != nil) != tt.wantErr {
				t.Errorf("XAPrepare() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.expectExecCall {
				if !c.prepared || c.preparedXid != tt.input.xid {
					t.Errorf("XAPrepare() should mark transaction as prepared")
				}
			}
		})
	}
}

func TestPostgresqlXAConn_Rollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid string
	}
	tests := []struct {
		name                     string
		input                    args
		wantErr                  bool
		mockConnNotSupportExecer bool
		expectExecCall           bool
	}{
		{
			name: "normal rollback",
			input: args{
				xid: "test-rollback-xid",
			},
			wantErr:                  false,
			mockConnNotSupportExecer: false,
			expectExecCall:           true,
		},
		{
			name: "invalid xid (empty)",
			input: args{
				xid: "",
			},
			wantErr:                  true,
			mockConnNotSupportExecer: false,
			expectExecCall:           false,
		},
		{
			name: "invalid xid (unsafe characters)",
			input: args{
				xid: "test'injection",
			},
			wantErr:                  true,
			mockConnNotSupportExecer: false,
			expectExecCall:           false,
		},
		{
			name: "conn does not support ExecerContext",
			input: args{
				xid: "test-rollback-xid-2",
			},
			wantErr:                  true,
			mockConnNotSupportExecer: true,
			expectExecCall:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTx := mock.NewMockTestDriverTx(ctrl)

			var conn driver.Conn
			if tt.mockConnNotSupportExecer {
				conn = &mockConnWithoutExecer{}
			} else {
				mockConn := mock.NewMockTestDriverConn(ctrl)
				if tt.expectExecCall {
					mockConn.EXPECT().ExecContext(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).DoAndReturn(func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
						return &driver.ResultNoRows, verifySQLContains(query, "ROLLBACK PREPARED", tt.input.xid)
					})
				}
				conn = mockConn
			}

			c := &PostgresqlXAConn{Conn: conn, tx: mockTx, prepared: true, preparedXid: tt.input.xid}
			err := c.Rollback(context.Background(), tt.input.xid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Rollback() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.expectExecCall {
				if c.prepared || c.preparedXid != "" {
					t.Errorf("Rollback() should clear prepared state")
				}
			}
		})
	}
}

func TestPostgresqlXAConn_Recover(t *testing.T) {
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
			name: "normal recover (with TMStartRScan)",
			args: args{
				flag: TMStartRScan,
			},
			want:    []string{"test-xid-1", "test-xid-2"},
			wantErr: false,
		},
		{
			name: "invalid flag",
			args: args{
				flag: TMFail,
			},
			wantErr: true,
		},
		{
			name: "TMEndRScan only (no scanning)",
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
			if !strings.Contains(query, "SELECT gid FROM pg_prepared_xacts") {
				return nil, errors.New("recover sql incorrect")
			}
			rows := &pgMockRows{
				data: [][]interface{}{
					{"test-xid-1"},
					{"test-xid-2"},
				},
			}
			return rows, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{Conn: mockConn}
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

func TestPostgresqlXAConn_Start(t *testing.T) {
	type args struct {
		xid   string
		flags int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal start (TMNoFlags)",
			args: args{
				xid:   "test-start-xid",
				flags: TMNoFlags,
			},
			wantErr: false,
		},
		{
			name: "invalid flags (TMJoin not supported)",
			args: args{
				xid:   "test-start-xid-2",
				flags: TMJoin,
			},
			wantErr: true,
		},
		{
			name: "invalid flags (TMResume not supported)",
			args: args{
				xid:   "test-start-xid-3",
				flags: TMResume,
			},
			wantErr: true,
		},
		{
			name: "invalid xid (empty)",
			args: args{
				xid:   "",
				flags: TMNoFlags,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{}
			err := c.Start(context.Background(), tt.args.xid, tt.args.flags)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostgresqlXAConn_End(t *testing.T) {
	type args struct {
		xid   string
		flags int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal end (TMSuccess)",
			args: args{
				xid:   "test-end-xid",
				flags: TMSuccess,
			},
			wantErr: false,
		},
		{
			name: "normal end (TMFail)",
			args: args{
				xid:   "test-end-xid-2",
				flags: TMFail,
			},
			wantErr: false,
		},
		{
			name: "invalid flags (TMSuspend not supported)",
			args: args{
				xid:   "test-end-xid-3",
				flags: TMSuspend,
			},
			wantErr: true,
		},
		{
			name: "invalid xid (empty)",
			args: args{
				xid:   "",
				flags: TMSuccess,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{}
			err := c.End(context.Background(), tt.args.xid, tt.args.flags)
			if (err != nil) != tt.wantErr {
				t.Errorf("End() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostgresqlXAConn_Forget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		xid string
	}
	tests := []struct {
		name                     string
		input                    args
		wantErr                  bool
		mockConnNotSupportQuery  bool
		mockConnNotSupportExecer bool
		transactionExists        bool
	}{
		{
			name: "normal forget with existing transaction",
			input: args{
				xid: "test-forget-xid",
			},
			wantErr:                  false,
			mockConnNotSupportQuery:  false,
			mockConnNotSupportExecer: false,
			transactionExists:        true,
		},
		{
			name: "forget with non-existing transaction",
			input: args{
				xid: "test-forget-xid-2",
			},
			wantErr:                  false,
			mockConnNotSupportQuery:  false,
			mockConnNotSupportExecer: false,
			transactionExists:        false,
		},
		{
			name: "invalid xid (empty)",
			input: args{
				xid: "",
			},
			wantErr:                  true,
			mockConnNotSupportQuery:  false,
			mockConnNotSupportExecer: false,
			transactionExists:        false,
		},
		{
			name: "conn does not support QueryerContext",
			input: args{
				xid: "test-forget-xid-3",
			},
			wantErr:                  true,
			mockConnNotSupportQuery:  true,
			mockConnNotSupportExecer: false,
			transactionExists:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTx := mock.NewMockTestDriverTx(ctrl)

			var conn driver.Conn
			if tt.mockConnNotSupportQuery {
				conn = &mockConnWithoutExecer{}
			} else {
				mockConn := mock.NewMockTestDriverConn(ctrl)
				if !tt.wantErr || tt.input.xid != "" {
					mockConn.EXPECT().QueryContext(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).DoAndReturn(func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
						if !strings.Contains(query, "SELECT 1 FROM pg_prepared_xacts WHERE gid = $1") {
							return nil, errors.New("query incorrect")
						}
						rows := &pgMockRows{
							data: [][]interface{}{},
						}
						if tt.transactionExists {
							rows.data = [][]interface{}{{1}}
						}
						return rows, nil
					})
				}

				if tt.transactionExists && !tt.wantErr {
					mockConn.EXPECT().ExecContext(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).DoAndReturn(func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
						return &driver.ResultNoRows, verifySQLContains(query, "ROLLBACK PREPARED", tt.input.xid)
					})
				}
				conn = mockConn
			}

			c := &PostgresqlXAConn{Conn: conn, tx: mockTx}
			err := c.Forget(context.Background(), tt.input.xid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Forget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockConnWithoutExecer struct{}

func (m *mockConnWithoutExecer) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (m *mockConnWithoutExecer) Close() error                              { return nil }
func (m *mockConnWithoutExecer) Begin() (driver.Tx, error)                 { return nil, nil }

type pgMockRows struct {
	idx  int
	data [][]interface{}
}

func (p *pgMockRows) Columns() []string {
	return []string{"gid"}
}

func (p *pgMockRows) Close() error {
	return nil
}

func (p *pgMockRows) Next(dest []driver.Value) error {
	if p.idx >= len(p.data) {
		return io.EOF
	}
	for i := range dest {
		if i < len(p.data[p.idx]) {
			dest[i] = p.data[p.idx][i]
		}
	}
	p.idx++
	return nil
}
