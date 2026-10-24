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
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestResolveTableMetaKeyPostgres(t *testing.T) {
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

	cache := &TableMetaCache{db: db}
	cases := []struct {
		ref  types.TableRef
		arg  string
		want types.TableMetaKey
	}{
		{types.TableRef{TableName: "Users"}, "Users", types.TableMetaKey{DBName: "app", Schema: "public", TableName: "users"}},
		{types.TableRef{TableName: "Users", TableNameQuoted: true}, `"Users"`, types.TableMetaKey{DBName: "app", Schema: "public", TableName: "Users"}},
		{types.TableRef{Qualifier: "other", TableName: "Users", TableNameQuoted: true}, `other."Users"`, types.TableMetaKey{DBName: "app", Schema: "other", TableName: "Users"}},
		{types.TableRef{Qualifier: "Other", QualifierQuoted: true, TableName: `user"name`, TableNameQuoted: true}, `"Other"."user""name"`, types.TableMetaKey{DBName: "app", Schema: "Other", TableName: `user"name`}},
	}
	for _, tt := range cases {
		mock.ExpectQuery("to_regclass").WithArgs(tt.arg).WillReturnRows(
			sqlmock.NewRows([]string{"database", "schema", "table"}).AddRow(tt.want.DBName, tt.want.Schema, tt.want.TableName),
		)
		err := conn.Raw(func(raw any) error {
			got, err := cache.ResolveTableMetaKey(context.Background(), raw.(driver.Conn), tt.ref)
			if err != nil {
				return err
			}
			if got != tt.want {
				t.Fatalf("ref %+v: got %+v, want %+v", tt.ref, got, tt.want)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNewTableMetaInstance(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cache := NewTableMetaInstance(db, "public")

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.tableMetaCache)
	assert.Equal(t, db, cache.db)
}

func TestTableMetaCache_GetTableMeta_EmptyTableName(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cache := NewTableMetaInstance(db, "public")
	tableMeta, err := cache.GetTableMeta(context.Background(), types.TableMetaKey{Schema: "public"})
	assert.Error(t, err)
	assert.Nil(t, tableMeta)
	assert.Contains(t, err.Error(), "table name is empty")
}
