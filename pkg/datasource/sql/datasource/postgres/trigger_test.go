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

package postgres

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

func TestPostgresTrigger_LoadOne(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	columnQuery := regexp.QuoteMeta(
		`SELECT table_name, table_schema, column_name, data_type, udt_name, is_nullable, column_default, is_identity FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`,
	)
	indexQuery := regexp.QuoteMeta(
		`SELECT ic.relname AS index_name, a.attname AS column_name, ix.indisprimary, ix.indisunique FROM pg_class tc JOIN pg_namespace ns ON ns.oid = tc.relnamespace JOIN pg_index ix ON tc.oid = ix.indrelid JOIN pg_class ic ON ic.oid = ix.indexrelid JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS cols(attnum, ordinality) ON TRUE JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = cols.attnum WHERE ns.nspname = $1 AND tc.relname = $2 ORDER BY ic.relname, cols.ordinality`,
	)

	columnRows := sqlmock.NewRows([]string{
		"table_name", "table_schema", "column_name", "data_type", "udt_name", "is_nullable", "column_default", "is_identity",
	}).
		AddRow("users", "public", "id", "bigint", "int8", "NO", "nextval('users_id_seq'::regclass)", "NO").
		AddRow("users", "public", "name", "character varying", "varchar", "YES", nil, "NO")
	indexRows := sqlmock.NewRows([]string{"index_name", "column_name", "indisprimary", "indisunique"}).
		AddRow("users_pkey", "id", true, true)

	mock.ExpectPrepare(columnQuery).
		ExpectQuery().
		WithArgs("public", "users").
		WillReturnRows(columnRows)
	mock.ExpectPrepare(indexQuery).
		ExpectQuery().
		WithArgs("public", "users").
		WillReturnRows(indexRows)

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("failed to get sql connection: %v", err)
	}

	trigger := NewPostgresTrigger()
	meta, err := trigger.LoadOne(context.Background(), types.TableMetaKey{DBName: "seata_go_test", Schema: "public", TableName: "users"}, conn)
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.Equal(t, "public.users", meta.TableName)
	assert.Equal(t, []string{"id", "name"}, meta.ColumnNames)
	assert.True(t, meta.Columns["id"].Autoincrement)
	assert.Equal(t, "users_pkey", meta.Indexs["users_pkey"].Name)
	assert.Equal(t, 1, len(meta.Indexs["users_pkey"].Columns))
}

func TestPostgresTrigger_UsesResolvedSchemaAndTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	key := types.TableMetaKey{DBName: "app", Schema: "tenant", TableName: "Users.Name"}
	mock.ExpectPrepare("FROM information_schema.columns").ExpectQuery().WithArgs(key.Schema, key.TableName).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "table_schema", "column_name", "data_type", "udt_name", "is_nullable", "column_default", "is_identity"}).
			AddRow(key.TableName, key.Schema, "id", "bigint", "int8", "NO", nil, "NO"))
	mock.ExpectPrepare("FROM pg_class tc").ExpectQuery().WithArgs(key.Schema, key.TableName).
		WillReturnRows(sqlmock.NewRows([]string{"index_name", "column_name", "indisprimary", "indisunique"}).
			AddRow("users_pkey", "id", true, true))

	meta, err := NewPostgresTrigger().LoadOne(context.Background(), key, conn)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "tenant.Users.Name", meta.TableName)
	assert.Equal(t, key.Schema, meta.Columns["id"].Schema)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresTrigger_SeparatesSchemaLockKeys(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var lockKeys []string
	for _, schema := range []string{"one", "two"} {
		key := types.TableMetaKey{DBName: "app", Schema: schema, TableName: "users"}
		mock.ExpectPrepare("FROM information_schema.columns").ExpectQuery().WithArgs(schema, "users").
			WillReturnRows(sqlmock.NewRows([]string{"table_name", "table_schema", "column_name", "data_type", "udt_name", "is_nullable", "column_default", "is_identity"}).
				AddRow("users", schema, "id", "bigint", "int8", "NO", nil, "NO"))
		mock.ExpectPrepare("FROM pg_class tc").ExpectQuery().WithArgs(schema, "users").
			WillReturnRows(sqlmock.NewRows([]string{"index_name", "column_name", "indisprimary", "indisunique"}).
				AddRow("users_pkey", "id", true, true))

		meta, err := NewPostgresTrigger().LoadOne(context.Background(), key, conn)
		if err != nil {
			t.Fatal(err)
		}
		image := types.NewEmptyRecordImage(meta, types.SQLTypeUpdate)
		image.Rows = []types.RowImage{{Columns: []types.ColumnImage{{ColumnName: "id", Value: int64(1)}}}}
		lockKeys = append(lockKeys, util.BuildLockKey(image, *meta))
	}

	assert.Equal(t, []string{"ONE.USERS:1", "TWO.USERS:1"}, lockKeys)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
