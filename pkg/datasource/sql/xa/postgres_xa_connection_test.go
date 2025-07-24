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
		name    string
		input   args
		wantErr bool
	}{
		{
			name: "normal commit (two-phase)",
			input: args{
				xid:      "gtrid1,bqual1,1",
				onePhase: false,
			},
			wantErr: false,
		},
		{
			name: "normal commit (one-phase)",
			input: args{
				xid:      "gtrid2,bqual2,2",
				onePhase: true,
			},
			wantErr: false,
		},
		{
			name: "invalid xid format (missing fields)",
			input: args{
				xid:      "invalid_xid",
				onePhase: false,
			},
			wantErr: true,
		},
		{
			name: "non-numeric formatID in xid",
			input: args{
				xid:      "gtrid3,bqual3,abc",
				onePhase: false,
			},
			wantErr: true,
		},
	}

	// Create mock connection to verify SQL generation
	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			if err := verifySQLContains(query, "XA COMMIT", "gtrid", "bqual"); err != nil {
				return nil, err
			}
			if strings.Contains(query, "ONE PHASE") && !strings.Contains(query, "ONE PHASE") {
				return nil, errors.New("missing ONE PHASE in sql")
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{Conn: mockConn}
			if err := c.Commit(context.Background(), tt.input.xid, tt.input.onePhase); (err != nil) != tt.wantErr {
				t.Errorf("Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostgresqlXAConn_End tests the End method
func TestPostgresqlXAConn_End(t *testing.T) {
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
			name: "normal end (TMSuccess)",
			input: args{
				xid:   "gtrid,e1,1",
				flags: TMSuccess,
			},
			wantErr: false,
		},
		{
			name: "normal end (TMSuspend)",
			input: args{
				xid:   "gtrid,e2,2",
				flags: TMSuspend,
			},
			wantErr: false,
		},
		{
			name: "normal end (TMFail)",
			input: args{
				xid:   "gtrid,e3,3",
				flags: TMFail,
			},
			wantErr: false,
		},
		{
			name: "invalid flags",
			input: args{
				xid:   "gtrid,e4,4",
				flags: 9999,
			},
			wantErr: true,
		},
		{
			name: "invalid xid format",
			input: args{
				xid:   "invalid_xid",
				flags: TMSuccess,
			},
			wantErr: true,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			if err := verifySQLContains(query, "XA END", "gtrid", "e"); err != nil {
				return nil, err
			}
			// Verify SUSPEND flag is correctly added
			if strings.Contains(query, "SUSPEND") && !strings.Contains(query, "SUSPEND") {
				return nil, errors.New("missing SUSPEND in sql")
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{Conn: mockConn}
			if err := c.End(context.Background(), tt.input.xid, tt.input.flags); (err != nil) != tt.wantErr {
				t.Errorf("End() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostgresqlXAConn_Start tests the Start method
func TestPostgresqlXAConn_Start(t *testing.T) {
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
			name: "normal start (no flags)",
			input: args{
				xid:   "g1,s1,1",
				flags: TMNoFlags,
			},
			wantErr: false,
		},
		{
			name: "normal start (TMJoin)",
			input: args{
				xid:   "g2,s2,2",
				flags: TMJoin,
			},
			wantErr: false,
		},
		{
			name: "normal start (TMResume)",
			input: args{
				xid:   "g3,s3,3",
				flags: TMResume,
			},
			wantErr: false,
		},
		{
			name: "invalid flags",
			input: args{
				xid:   "g4,s4,4",
				flags: 8888,
			},
			wantErr: true,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			if err := verifySQLContains(query, "XA START", "g", "s"); err != nil {
				return nil, err
			}
			// Verify JOIN/RESUME flags
			if strings.Contains(query, "JOIN") && !strings.Contains(query, "JOIN") {
				return nil, errors.New("missing JOIN in sql")
			}
			if strings.Contains(query, "RESUME") && !strings.Contains(query, "RESUME") {
				return nil, errors.New("missing RESUME in sql")
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{Conn: mockConn}
			if err := c.Start(context.Background(), tt.input.xid, tt.input.flags); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostgresqlXAConn_XAPrepare tests the XAPrepare method
func TestPostgresqlXAConn_XAPrepare(t *testing.T) {
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
				xid: "gp1,bp1,1",
			},
			wantErr: false,
		},
		{
			name: "invalid xid format",
			input: args{
				xid: "invalid_prepare_xid",
			},
			wantErr: true,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			if err := verifySQLContains(query, "XA PREPARE", "gp1", "bp1"); err != nil {
				return nil, err
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{Conn: mockConn}
			if err := c.XAPrepare(context.Background(), tt.input.xid); (err != nil) != tt.wantErr {
				t.Errorf("XAPrepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostgresqlXAConn_Recover tests the Recover method
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
			want:    []string{"rec_g1,rec_b1,1", "rec_g2,rec_b2,2"},
			wantErr: false,
		},
		{
			name: "invalid flag (non-scanning flag)",
			args: args{
				flag: TMFail,
			},
			wantErr: true,
		},
		{
			name: "TMEndRScan only (no scanning performed)",
			args: args{
				flag: TMEndRScan,
			},
			want:    nil,
			wantErr: false,
		},
	}

	// Mock query result: PostgreSQL's XA RECOVER returns 3 fields (formatID, gtrid, bqual)
	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			if !strings.Contains(query, "XA RECOVER FORMATAS TEXT") {
				return nil, errors.New("recover sql incorrect")
			}

			rows := &pgMockRows{
				data: [][]interface{}{
					{int64(1), "rec_g1", "rec_b1"},
					{int64(2), "rec_g2", "rec_b2"},
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

// TestPostgresqlXAConn_Rollback tests the Rollback method
func TestPostgresqlXAConn_Rollback(t *testing.T) {
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
				xid: "gr1,br1,1",
			},
			wantErr: false,
		},
		{
			name: "invalid xid format",
			input: args{
				xid: "invalid_rollback_xid",
			},
			wantErr: true,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			if err := verifySQLContains(query, "XA ROLLBACK", "gr1", "br1"); err != nil {
				return nil, err
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{Conn: mockConn}
			if err := c.Rollback(context.Background(), tt.input.xid); (err != nil) != tt.wantErr {
				t.Errorf("Rollback() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostgresqlXAConn_Forget tests the Forget method
func TestPostgresqlXAConn_Forget(t *testing.T) {
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
				xid: "fg1,fb1,1",
			},
			wantErr: false,
		},
		{
			name: "invalid xid format",
			input: args{
				xid: "invalid_forget_xid",
			},
			wantErr: true,
		},
	}

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			if err := verifySQLContains(query, "XA FORGET", "fg1", "fb1"); err != nil {
				return nil, err
			}
			return nil, nil
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PostgresqlXAConn{Conn: mockConn}
			if err := c.Forget(context.Background(), tt.input.xid); (err != nil) != tt.wantErr {
				t.Errorf("Forget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// pgMockRows simulates result set returned by PostgreSQL XA RECOVER
type pgMockRows struct {
	idx  int
	data [][]interface{}
}

func (p *pgMockRows) Columns() []string {
	return []string{"formatid", "gtrid", "bqual"}
}

func (p *pgMockRows) Close() error {
	return nil
}

func (p *pgMockRows) Next(dest []driver.Value) error {
	if p.idx >= len(p.data) {
		return io.EOF
	}
	for i := range dest {
		dest[i] = p.data[p.idx][i]
	}
	p.idx++
	return nil
}
