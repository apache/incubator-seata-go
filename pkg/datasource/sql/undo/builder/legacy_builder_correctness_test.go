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

package builder

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

// Only database I/O is replaced. Parsing, image building, lock generation
// and after-image SQL all use production code.
type legacyBuilderTestConn struct {
	prepared bool
	query    string
	args     []driver.Value
}

func (c *legacyBuilderTestConn) Prepare(query string) (driver.Stmt, error) {
	c.prepared, c.query = true, query
	return &legacyBuilderTestStmt{conn: c}, nil
}
func (*legacyBuilderTestConn) Close() error { return nil }
func (*legacyBuilderTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected Begin")
}

type legacyBuilderTestStmt struct{ conn *legacyBuilderTestConn }

func (*legacyBuilderTestStmt) Close() error  { return nil }
func (*legacyBuilderTestStmt) NumInput() int { return -1 }
func (*legacyBuilderTestStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected Exec")
}
func (s *legacyBuilderTestStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.conn.args = append([]driver.Value(nil), args...)
	return &legacyBuilderTestRows{}, nil
}

type legacyBuilderTestRows struct{ next int64 }

func (*legacyBuilderTestRows) Columns() []string { return []string{"id"} }
func (*legacyBuilderTestRows) Close() error      { return nil }
func (r *legacyBuilderTestRows) Next(dest []driver.Value) error {
	if r.next == 2 {
		return io.EOF
	}
	if len(dest) != 1 {
		return fmt.Errorf("unexpected destination length: %d", len(dest))
	}
	r.next++
	dest[0] = r.next
	return nil
}

func legacyBuilderTestMeta() types.TableMeta {
	id := types.ColumnMeta{ColumnName: "id", DatabaseTypeString: "BIGINT"}
	return types.TableMeta{
		TableName: "t_user", ColumnNames: []string{"id"},
		Columns: map[string]types.ColumnMeta{"id": id},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{id}},
		},
	}
}

func legacyBuilderTestContext(t *testing.T, query string, conn driver.Conn) *types.ExecContext {
	t.Helper()
	parsed, err := parser.DoParser(query)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy multi builders also read the top-level statement. Populate it to
	// isolate the reported bugs from a separate nil-statement failure.
	if len(parsed.MultiStmt) > 0 {
		parsed.UpdateStmt = parsed.MultiStmt[0].UpdateStmt
		parsed.DeleteStmt = parsed.MultiStmt[0].DeleteStmt
	}
	return &types.ExecContext{
		Query: query, ParseContext: parsed, Conn: conn, DBType: types.DBTypeMySQL,
		TxCtx:       &types.TransactionContext{LockKeys: map[string]struct{}{}},
		MetaDataMap: map[string]types.TableMeta{"t_user": legacyBuilderTestMeta()},
	}
}

func TestLegacyBuildersBeforeImageParameters(t *testing.T) {
	log.Init()
	cases := []struct {
		name    string
		builder undo.UndoLogBuilder
		query   string
		values  []driver.Value
	}{
		{"delete", &MySQLDeleteUndoLogBuilder{},
			"DELETE FROM t_user WHERE id IN (?,?)", []driver.Value{int64(1), int64(2)}},
		{"multi_update", &MySQLMultiUpdateUndoLogBuilder{},
			"UPDATE t_user SET id=? WHERE id=?; UPDATE t_user SET id=? WHERE id=?",
			[]driver.Value{int64(11), int64(1), int64(22), int64(2)}},
		{"multi_delete", &MySQLMultiDeleteUndoLogBuilder{},
			"DELETE FROM t_user WHERE id=?; DELETE FROM t_user WHERE id=?",
			[]driver.Value{int64(1), int64(2)}},
	}
	for _, tc := range cases {
		for _, named := range []bool{true, false} {
			mode := "Values"
			if named {
				mode = "NamedValues"
			}
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				conn := &legacyBuilderTestConn{}
				execCtx := legacyBuilderTestContext(t, tc.query, conn)
				if named {
					for i, value := range tc.values {
						execCtx.NamedValues = append(execCtx.NamedValues,
							driver.NamedValue{Ordinal: i + 1, Value: value})
					}
				} else {
					execCtx.Values = tc.values
				}
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("BeforeImage panicked: %v; reached Prepare=%v", p, conn.prepared)
					}
				}()
				images, err := tc.builder.BeforeImage(context.Background(), execCtx)
				if err != nil {
					t.Fatal(err)
				}
				if len(images) != 1 || len(images[0].Rows) != 2 {
					t.Fatalf("expected one image containing two records, got %#v", images)
				}
				if !conn.prepared || !reflect.DeepEqual(conn.args, []driver.Value{int64(1), int64(2)}) {
					t.Fatalf("expected query arguments [1 2], got prepared=%v args=%v", conn.prepared, conn.args)
				}
			})
		}
	}
}

func TestLegacyDeleteBuildersLockPrimaryKeys(t *testing.T) {
	log.Init()
	cases := []struct {
		name    string
		builder undo.UndoLogBuilder
		query   string
	}{
		{"delete", &MySQLDeleteUndoLogBuilder{}, "DELETE FROM t_user WHERE id IN (1,2)"},
		{"multi_delete", &MySQLMultiDeleteUndoLogBuilder{},
			"DELETE FROM t_user WHERE id=1; DELETE FROM t_user WHERE id=2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &legacyBuilderTestConn{}
			execCtx := legacyBuilderTestContext(t, tc.query, conn)
			execCtx.Values = []driver.Value{} // Avoid the independent NamedValues bug.
			images, err := tc.builder.BeforeImage(context.Background(), execCtx)
			if err != nil {
				t.Fatal(err)
			}
			if len(images) != 1 || len(images[0].Rows) != 2 {
				t.Fatalf("expected one image containing two records, got %#v", images)
			}
			for i, row := range images[0].Rows {
				if len(row.Columns) != 1 || row.Columns[0].Value != int64(i+1) ||
					row.Columns[0].KeyType != types.IndexTypePrimaryKey {
					t.Fatalf("invalid primary key fixture: %#v", row)
				}
			}
			keys := make([]string, 0, len(execCtx.TxCtx.LockKeys))
			for key := range execCtx.TxCtx.LockKeys {
				keys = append(keys, key)
			}
			t.Logf("before image IDs=[1 2]; lock keys=%q", keys)
			if _, ok := execCtx.TxCtx.LockKeys["T_USER:1,2"]; !ok || len(execCtx.TxCtx.LockKeys) != 1 {
				t.Errorf("expected lock T_USER:1,2, got %q", keys)
			}
		})
	}
}

func TestLegacyMultiUpdateAfterImageSQL(t *testing.T) {
	b := &MySQLMultiUpdateUndoLogBuilder{}
	before := &types.RecordImage{TableName: "t_user", Rows: []types.RowImage{
		{Columns: []types.ColumnImage{{ColumnName: "id", Value: int64(1), KeyType: types.IndexTypePrimaryKey}}},
		{Columns: []types.ColumnImage{{ColumnName: "id", Value: int64(2), KeyType: types.IndexTypePrimaryKey}}},
	}}
	query, args := b.buildAfterImageSQL(before, legacyBuilderTestMeta())
	t.Logf("SQL=%q; args=%v", query, args)
	if !reflect.DeepEqual(args, []driver.Value{int64(1), int64(2)}) {
		t.Fatalf("unexpected primary key argument order: %v", args)
	}
	// Control: the expected SQL is accepted by the same MySQL parser.
	if _, err := parser.DoParser("SELECT * FROM t_user WHERE (`id`) IN ((?),(?))"); err != nil {
		t.Fatalf("invalid SQL control: %v", err)
	}
	if !strings.Contains(strings.ToUpper(query), " WHERE ") {
		t.Error("after-image SQL is missing WHERE")
	}
	if _, err := parser.DoParser(query); err != nil {
		t.Errorf("generated SQL cannot be parsed: %v", err)
	}

	conn := &legacyBuilderTestConn{}
	execCtx := legacyBuilderTestContext(t, "UPDATE t_user SET id=11 WHERE id=1", conn)
	images, err := b.AfterImage(context.Background(), execCtx, []*types.RecordImage{before})
	if err != nil {
		t.Fatal(err)
	}
	if !conn.prepared || conn.query != query || !reflect.DeepEqual(conn.args, args) {
		t.Fatalf("expected after-image query %q with args %v, got prepared=%v query=%q args=%v",
			query, args, conn.prepared, conn.query, conn.args)
	}
	if len(images) != 1 || len(images[0].Rows) != 2 {
		t.Fatalf("expected one after image containing two records, got %#v", images)
	}
}

func TestLegacyMultiUpdateAfterImageEmpty(t *testing.T) {
	cases := []struct {
		name   string
		before []*types.RecordImage
	}{
		{"nil_images", nil},
		{"empty_images", []*types.RecordImage{}},
		{"no_rows", []*types.RecordImage{{
			TableName: "t_user", SQLType: types.SQLTypeUpdate, Rows: []types.RowImage{},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &legacyBuilderTestConn{}
			execCtx := legacyBuilderTestContext(t, "UPDATE t_user SET id=11 WHERE id=1", conn)
			images, err := (&MySQLMultiUpdateUndoLogBuilder{}).AfterImage(context.Background(), execCtx, tc.before)
			if err != nil {
				t.Fatal(err)
			}
			if conn.prepared {
				t.Fatalf("empty before image must not query the database, prepared %q", conn.query)
			}
			if !reflect.DeepEqual(images, tc.before) {
				t.Fatalf("expected unchanged empty images, got %#v", images)
			}
		})
	}
}
