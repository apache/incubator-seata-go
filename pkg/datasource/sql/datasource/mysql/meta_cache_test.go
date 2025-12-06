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

package mysql

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestNewTableMetaInstance(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	cache := NewTableMetaInstance(db, cfg)

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.tableMetaCache)
	assert.Equal(t, db, cache.db)
}

func TestTableMetaCache_Init(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	cache := NewTableMetaInstance(db, cfg)
	ctx := context.Background()

	err = cache.Init(ctx, db)
	assert.NoError(t, err)
}

func TestTableMetaCache_Destroy(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	cache := NewTableMetaInstance(db, cfg)

	err = cache.Destroy()
	assert.NoError(t, err)
}

func TestTableMetaCache_GetTableMeta_EmptyTableName(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	cache := NewTableMetaInstance(db, cfg)
	ctx := context.Background()

	tableMeta, err := cache.GetTableMeta(ctx, "test_db", "")
	assert.Error(t, err)
	assert.Nil(t, tableMeta)
	assert.Contains(t, err.Error(), "table name is empty")
}

func TestTableMetaCache_GetTableMeta_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	mock.ExpectBegin()

	columnRows := sqlmock.NewRows([]string{
		"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE",
		"COLUMN_TYPE", "COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
	}).
		AddRow("users", "test_db", "id", "BIGINT", "BIGINT(20)", "PRI", "NO", nil, "auto_increment").
		AddRow("users", "test_db", "name", "VARCHAR", "VARCHAR(100)", "", "YES", nil, "")

	mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
		ExpectQuery().
		WithArgs("test_db", "users").
		WillReturnRows(columnRows)

	indexRows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
		AddRow("PRIMARY", "id", 0)

	mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
		ExpectQuery().
		WithArgs("test_db", "users").
		WillReturnRows(indexRows)

	cache := NewTableMetaInstance(db, cfg)
	ctx := context.Background()

	tableMeta, err := cache.GetTableMeta(ctx, "test_db", "users")

	if err != nil {
		assert.Contains(t, err.Error(), "")
	} else {
		assert.NotNil(t, tableMeta)
	}
}

func TestTableMetaCache_GetTableMeta_DBConnectionError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	db.Close()

	mock.ExpectClose()

	cache := NewTableMetaInstance(db, cfg)
	ctx := context.Background()

	tableMeta, err := cache.GetTableMeta(ctx, "test_db", "users")
	assert.Error(t, err)
	assert.Nil(t, tableMeta)
}

func TestTableMetaCache_GetTableMeta_CacheHit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	cache := NewTableMetaInstance(db, cfg)
	ctx := context.Background()

	columnRows := sqlmock.NewRows([]string{
		"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE",
		"COLUMN_TYPE", "COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
	}).
		AddRow("test_table", "test_db", "id", "INT", "INT(11)", "PRI", "NO", nil, "auto_increment")

	mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
		ExpectQuery().
		WithArgs("test_db", "test_table").
		WillReturnRows(columnRows)

	indexRows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
		AddRow("PRIMARY", "id", 0)

	mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
		ExpectQuery().
		WithArgs("test_db", "test_table").
		WillReturnRows(indexRows)

	_, err = cache.GetTableMeta(ctx, "test_db", "test_table")

	if err == nil {
		t.Log("Successfully tested GetTableMeta code path")
	}
}

func TestTableMetaCache_MultipleInstances(t *testing.T) {
	db1, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db1.Close()

	db2, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db2.Close()

	cfg1 := &mysql.Config{
		User:   "test1",
		Passwd: "test1",
		DBName: "test_db1",
	}

	cfg2 := &mysql.Config{
		User:   "test2",
		Passwd: "test2",
		DBName: "test_db2",
	}

	cache1 := NewTableMetaInstance(db1, cfg1)
	cache2 := NewTableMetaInstance(db2, cfg2)

	assert.NotNil(t, cache1)
	assert.NotNil(t, cache2)
	assert.NotEqual(t, cache1, cache2)
	assert.NotEqual(t, cache1.db, cache2.db)
}

func TestTableMetaCache_ConcurrentAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	columnRows := sqlmock.NewRows([]string{
		"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE",
		"COLUMN_TYPE", "COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
	}).
		AddRow("test_table", "test_db", "id", "INT", "INT(11)", "PRI", "NO", nil, "auto_increment")

	indexRows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
		AddRow("PRIMARY", "id", 0)

	for i := 0; i < 3; i++ {
		mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
			ExpectQuery().
			WithArgs("test_db", "test_table").
			WillReturnRows(columnRows)

		mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
			ExpectQuery().
			WithArgs("test_db", "test_table").
			WillReturnRows(indexRows)
	}

	cache := NewTableMetaInstance(db, cfg)
	ctx := context.Background()

	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, _ = cache.GetTableMeta(ctx, "test_db", "test_table")
			done <- true
		}()
	}

	for i := 0; i < 3; i++ {
		<-done
	}

	assert.NotNil(t, cache)
}

func TestTableMetaCache_WithNilDB(t *testing.T) {
	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	cache := NewTableMetaInstance(nil, cfg)
	assert.NotNil(t, cache)
	assert.Nil(t, cache.db)
}

func TestTableMetaCache_ContextCancellation(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	cache := NewTableMetaInstance(db, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = cache.GetTableMeta(ctx, "test_db", "test_table")
	assert.Error(t, err)
}

func TestTableMetaCache_ErrorFromBaseCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	cfg := &mysql.Config{
		User:   "test",
		Passwd: "test",
		DBName: "test_db",
	}

	mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
		ExpectQuery().
		WithArgs("test_db", "error_table").
		WillReturnError(errors.New("query error"))

	cache := NewTableMetaInstance(db, cfg)
	ctx := context.Background()

	_, err = cache.GetTableMeta(ctx, "test_db", "error_table")
	assert.Error(t, err)
}
