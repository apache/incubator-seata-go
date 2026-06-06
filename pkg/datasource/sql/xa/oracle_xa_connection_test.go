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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

const oracleBranchXID = "oracle-global-xid-9"

type recordingOracleXAConn struct {
	execQueries   []string
	queryQueries  []string
	queryResults  [][]driver.Value
	queryResultID int
	execErr       error
	queryErr      error
}

func (c *recordingOracleXAConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *recordingOracleXAConn) Close() error {
	return nil
}

func (c *recordingOracleXAConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c *recordingOracleXAConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.execQueries = append(c.execQueries, query)
	return driver.ResultNoRows, c.execErr
}

func (c *recordingOracleXAConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.queryQueries = append(c.queryQueries, query)
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	values := []driver.Value{int64(oracleXAOK)}
	if c.queryResultID < len(c.queryResults) {
		values = c.queryResults[c.queryResultID]
		c.queryResultID++
	}
	return &recordingOracleRows{values: values}, nil
}

type recordingOracleRows struct {
	values []driver.Value
	read   bool
}

func (r *recordingOracleRows) Columns() []string {
	return []string{"RET"}
}

func (r *recordingOracleRows) Close() error {
	return nil
}

func (r *recordingOracleRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	copy(dest, r.values)
	r.read = true
	return nil
}

func TestOracleXAConn_FactoryCreatesResource(t *testing.T) {
	resource, err := CreateXAResource(&recordingOracleXAConn{}, types.DBTypeOracle)
	require.NoError(t, err)
	assert.IsType(t, &OracleXAConn{}, resource)
}

func TestOracleXAConn_LifecycleUsesDBMSXA(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		call     func(*OracleXAConn) error
		wantPart string
	}{
		{
			name: "start",
			op:   "XA_START",
			call: func(c *OracleXAConn) error {
				return c.Start(context.Background(), oracleBranchXID, TMNoFlags)
			},
			wantPart: "DBMS_XA.XA_START(l_xid, DBMS_XA.TMNOFLAGS)",
		},
		{
			name: "end",
			op:   "XA_END",
			call: func(c *OracleXAConn) error {
				return c.End(context.Background(), oracleBranchXID, TMSuccess)
			},
			wantPart: "DBMS_XA.XA_END(l_xid, DBMS_XA.TMSUCCESS)",
		},
		{
			name: "commit",
			op:   "XA_COMMIT",
			call: func(c *OracleXAConn) error {
				return c.Commit(context.Background(), oracleBranchXID, false)
			},
			wantPart: "DBMS_XA.XA_COMMIT(l_xid, FALSE)",
		},
		{
			name: "rollback",
			op:   "XA_ROLLBACK",
			call: func(c *OracleXAConn) error {
				return c.Rollback(context.Background(), oracleBranchXID)
			},
			wantPart: "DBMS_XA.XA_ROLLBACK(l_xid)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &recordingOracleXAConn{}
			c := NewOracleXaConn(conn)

			require.NoError(t, tt.call(c))
			require.Len(t, conn.execQueries, 1)
			assert.Contains(t, conn.execQueries[0], tt.wantPart)
			assert.Contains(t, conn.execQueries[0], "DBMS_XA_XID("+strconv.Itoa(oracleXAFormatID)+",")
			assert.Contains(t, conn.execQueries[0], "DBMS_XA."+tt.op+":ret=")
		})
	}
}

func TestOracleXAConn_FlagsAndValidation(t *testing.T) {
	t.Run("start rejects invalid flag", func(t *testing.T) {
		conn := &recordingOracleXAConn{}
		c := NewOracleXaConn(conn)

		err := c.Start(context.Background(), oracleBranchXID, TMSuspend)
		require.EqualError(t, err, "invalid arguments")
		assert.Empty(t, conn.execQueries)
	})

	t.Run("end rejects tmfail", func(t *testing.T) {
		conn := &recordingOracleXAConn{}
		c := NewOracleXaConn(conn)

		err := c.End(context.Background(), oracleBranchXID, TMFail)
		require.EqualError(t, err, "oracle xa end TMFAIL is not supported")
		assert.Empty(t, conn.execQueries)
	})

	t.Run("invalid xid format", func(t *testing.T) {
		c := NewOracleXaConn(&recordingOracleXAConn{})
		err := c.Commit(context.Background(), "invalid-xid", false)
		require.EqualError(t, err, "invalid xa branch xid: invalid-xid")
	})

	t.Run("xid part too long", func(t *testing.T) {
		c := NewOracleXaConn(&recordingOracleXAConn{})
		globalXid := strings.Repeat("g", oracleXAMaxXIDPartSize+1)
		err := c.Start(context.Background(), globalXid+"-1", TMNoFlags)
		require.EqualError(t, err, "oracle xa gtrid exceeds 64 bytes")
	})
}

func TestOracleXAConn_PrepareUsesReturnCode(t *testing.T) {
	conn := &recordingOracleXAConn{}
	c := NewOracleXaConn(conn)

	require.NoError(t, c.XAPrepare(context.Background(), oracleBranchXID))
	require.Len(t, conn.queryQueries, 1)
	assert.Contains(t, conn.queryQueries[0], "SELECT DBMS_XA.XA_PREPARE(DBMS_XA_XID("+strconv.Itoa(oracleXAFormatID)+",")
	assert.Contains(t, conn.queryQueries[0], "FROM DUAL")
}

func TestOracleXAConn_PrepareReturnsReadOnlySentinel(t *testing.T) {
	conn := &recordingOracleXAConn{queryResults: [][]driver.Value{{int64(XAReadOnly)}}}
	c := NewOracleXaConn(conn)

	err := c.XAPrepare(context.Background(), oracleBranchXID)
	require.ErrorIs(t, err, ErrXAReadOnly)
	require.Len(t, conn.queryQueries, 1)
}

func TestOracleXAConn_PrepareReturnsOracleRetAndOER(t *testing.T) {
	conn := &recordingOracleXAConn{queryResults: [][]driver.Value{{int64(-4)}, {int64(24756)}}}
	c := NewOracleXaConn(conn)

	err := c.XAPrepare(context.Background(), oracleBranchXID)
	require.EqualError(t, err, "DBMS_XA.XA_PREPARE:ret=-4,oer=24756")
	require.Len(t, conn.queryQueries, 2)
	assert.Contains(t, conn.queryQueries[1], "DBMS_XA.XA_GETLASTOER()")
}

func TestOracleXAConn_UnsupportedOperationsReturnExplicitErrors(t *testing.T) {
	c := NewOracleXaConn(&recordingOracleXAConn{})

	xids, err := c.Recover(context.Background(), TMStartRScan)
	require.EqualError(t, err, "oracle xa recover is not supported")
	assert.Nil(t, xids)

	err = c.Forget(context.Background(), oracleBranchXID)
	require.EqualError(t, err, "oracle xa forget is not supported")
}

func TestOracleXAConn_PropagatesDriverErrors(t *testing.T) {
	boom := errors.New("boom")
	conn := &recordingOracleXAConn{execErr: boom}
	c := NewOracleXaConn(conn)

	require.ErrorIs(t, c.Commit(context.Background(), oracleBranchXID, false), boom)
	require.Len(t, conn.execQueries, 1)
}
