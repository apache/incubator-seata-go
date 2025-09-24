package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func newTestConn(t *testing.T) (*sql.Conn, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)

	cleanup := func() {
		conn.Close()
		db.Close()
	}

	return conn, mock, cleanup
}

func TestPostgresqlTrigger_LoadOne_Success(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	dbName := "test_db"
	tableName := "test_table"

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.columns").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_name", "table_catalog", "column_name", "data_type", "column_type", "is_nullable", "column_default", "extra",
		}).AddRow(tableName, dbName, "id", "integer", "integer", "NO", nil, "auto_increment").
			AddRow(tableName, dbName, "name", "text", "text", "YES", nil, ""))

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.table_constraints").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

	mock.ExpectPrepare("SELECT .* FROM .*pg_catalog.pg_index").
		ExpectQuery().
		WithArgs(tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"index_name", "column_name", "non_unique",
		}).AddRow("idx_id", "id", int64(0)))

	tr := NewPostgresqlTrigger()
	meta, err := tr.LoadOne(context.Background(), dbName, tableName, conn)

	assert.NoError(t, err)
	assert.Equal(t, tableName, meta.TableName)
	assert.Contains(t, meta.Columns, "id")
	assert.Equal(t, "PRI", meta.Columns["id"].ColumnKey)
	assert.Contains(t, meta.Indexs, "idx_id")
}

func TestPostgresqlTrigger_LoadOne_NoIndex(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	dbName := "test_db"
	tableName := "no_index"

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.columns").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_name", "table_catalog", "column_name", "data_type", "column_type", "is_nullable", "column_default", "extra",
		}).AddRow(tableName, dbName, "id", "integer", "integer", "NO", nil, "auto_increment"))

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.table_constraints").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

	mock.ExpectPrepare("SELECT .* FROM .*pg_catalog.pg_index").
		ExpectQuery().
		WithArgs(tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"index_name", "column_name", "non_unique",
		})) // no indexes

	tr := NewPostgresqlTrigger()
	meta, err := tr.LoadOne(context.Background(), dbName, tableName, conn)
	assert.Error(t, err)
	assert.Nil(t, meta)
}

func TestPostgresqlTrigger_LoadOne_ScanColumnFail(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	dbName := "test_db"
	tableName := "fail_column"

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.columns").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_name", "table_catalog", "column_name", "data_type", "column_type", "is_nullable", "column_default", "extra",
		}).AddRow("bad", nil, nil, nil, nil, nil, nil, nil)) // Scan will fail

	tr := NewPostgresqlTrigger()
	_, err := tr.LoadOne(context.Background(), dbName, tableName, conn)
	assert.Error(t, err)
}

func TestPostgresqlTrigger_LoadOne_EmptyColumns(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	dbName := "test_db"
	tableName := "empty_columns"

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.columns").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_name", "table_catalog", "column_name", "data_type", "column_type", "is_nullable", "column_default", "extra",
		})) // no rows

	tr := NewPostgresqlTrigger()
	_, err := tr.LoadOne(context.Background(), dbName, tableName, conn)
	assert.Error(t, err)
}

func TestPostgresqlTrigger_LoadOne_QueryColumnFail(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	dbName := "test_db"
	tableName := "bad_table"

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.columns").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnError(errors.New("query failed"))

	tr := NewPostgresqlTrigger()
	_, err := tr.LoadOne(context.Background(), dbName, tableName, conn)
	assert.Error(t, err)
}

func TestPostgresqlTrigger_LoadOne_QueryIndexFail(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	dbName := "test_db"
	tableName := "index_fail"

	// Column query OK
	mock.ExpectPrepare("SELECT .* FROM .*information_schema.columns").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_name", "table_catalog", "column_name", "data_type", "column_type", "is_nullable", "column_default", "extra",
		}).AddRow(tableName, dbName, "id", "int", "int", "NO", nil, ""))

	// Primary key OK
	mock.ExpectPrepare("SELECT .* FROM .*information_schema.table_constraints").
		ExpectQuery().
		WithArgs(dbName, tableName, "").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

	// Index fails
	mock.ExpectPrepare("SELECT .* FROM .*pg_catalog.pg_index").
		ExpectQuery().
		WithArgs(tableName, "").
		WillReturnError(errors.New("index query failed"))

	tr := NewPostgresqlTrigger()
	_, err := tr.LoadOne(context.Background(), dbName, tableName, conn)
	assert.Error(t, err)
}

func TestPostgresqlTrigger_LoadAll_MultipleTables(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	dbName := "test_db"
	table := "t1"

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.columns").
		ExpectQuery().
		WithArgs(dbName, table, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_name", "table_catalog", "column_name", "data_type", "column_type", "is_nullable", "column_default", "extra",
		}).AddRow(table, dbName, "id", "integer", "integer", "NO", nil, "auto_increment"))

	mock.ExpectPrepare("SELECT .* FROM .*information_schema.table_constraints").
		ExpectQuery().
		WithArgs(dbName, table, "").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

	mock.ExpectPrepare("SELECT .* FROM .*pg_catalog.pg_index").
		ExpectQuery().
		WithArgs(table, "").
		WillReturnRows(sqlmock.NewRows([]string{"index_name", "column_name", "non_unique"}).AddRow("idx_id", "id", int64(0)))

	tr := NewPostgresqlTrigger()
	metas, err := tr.LoadAll(context.Background(), dbName, conn, table, "bad_table")
	assert.NoError(t, err)
	assert.Len(t, metas, 1)
	assert.Equal(t, table, metas[0].TableName)
}

func TestPostgresqlTrigger_getIndexes_MultiColumnIndex(t *testing.T) {
	conn, mock, cleanup := newTestConn(t)
	defer cleanup()

	table := "test_table"

	mock.ExpectPrepare("SELECT .* FROM .*pg_catalog.pg_index").
		ExpectQuery().
		WithArgs(table, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"index_name", "column_name", "non_unique",
		}).AddRow("idx_composite", "col1", int64(0)).
			AddRow("idx_composite", "col2", int64(0))) // same index, multiple cols

	tr := NewPostgresqlTrigger()
	indexes, err := tr.getIndexes(context.Background(), "test_db", table, conn)
	assert.NoError(t, err)
	assert.Len(t, indexes, 2) // returned as separate entries, will be merged in LoadOne
}
