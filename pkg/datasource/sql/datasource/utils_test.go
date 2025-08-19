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

package datasource

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetScanSlice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []*sqlmock.Column{
		sqlmock.NewColumn("c_varchar").OfType("VARCHAR", ""),
		sqlmock.NewColumn("c_char").OfType("CHAR", ""),
		sqlmock.NewColumn("c_text").OfType("TEXT", sql.NullString{}),
		sqlmock.NewColumn("c_json").OfType("JSON", sql.NullString{}),
		sqlmock.NewColumn("c_decimal").OfType("DECIMAL", ""),
		sqlmock.NewColumn("c_numeric").OfType("NUMERIC", ""),
		sqlmock.NewColumn("c_bigint").OfType("BIGINT", int64(0)),
		sqlmock.NewColumn("c_int").OfType("INT", int32(0)),
		sqlmock.NewColumn("c_smallint").OfType("SMALLINT", int16(0)),
		sqlmock.NewColumn("c_tinyint").OfType("TINYINT", int8(0)),
		sqlmock.NewColumn("c_int_unsigned").OfType("INT UNSIGNED", uint32(0)),
		sqlmock.NewColumn("c_bigint_unsigned").OfType("BIGINT UNSIGNED", uint64(0)),
		sqlmock.NewColumn("c_float").OfType("FLOAT", float32(0)),
		sqlmock.NewColumn("c_double").OfType("DOUBLE", float64(0)),
		sqlmock.NewColumn("c_bit").OfType("BIT", []byte{}),
		sqlmock.NewColumn("c_binary").OfType("BINARY", []byte{}),
		sqlmock.NewColumn("c_blob").OfType("BLOB", []byte{}),
		sqlmock.NewColumn("c_timestamp").OfType("TIMESTAMP", time.Time{}),
		sqlmock.NewColumn("c_datetime").OfType("DATETIME", sql.NullTime{}),
		sqlmock.NewColumn("c_date").OfType("DATE", sql.NullTime{}),
		sqlmock.NewColumn("c_time").OfType("TIME", sql.NullTime{}),
		sqlmock.NewColumn("c_year").OfType("YEAR", int16(0)),
		sqlmock.NewColumn("c_boolean").OfType("BOOLEAN", true),
		sqlmock.NewColumn("c_nullable_int").OfType("INTEGER", sql.NullInt64{}),
		sqlmock.NewColumn("c_nullable_float").OfType("REAL", sql.NullFloat64{}),
		sqlmock.NewColumn("c_nullable_bool").OfType("BOOL", sql.NullBool{}),
		sqlmock.NewColumn("c_raw_bytes").OfType("VARBINARY", sql.RawBytes{}),
		sqlmock.NewColumn("c_unknown_type").OfType("GEOMETRY", new(interface{})),
	}

	rows := sqlmock.NewRowsWithColumnDefinition(columns...)
	mock.ExpectQuery("SELECT \\* FROM comprehensive_table").WillReturnRows(rows)

	res, err := db.Query("SELECT * FROM comprehensive_table")
	if err != nil {
		t.Fatalf("failed to run mock query: %v", err)
	}
	defer res.Close()

	columnTypes, err := res.ColumnTypes()
	if err != nil {
		t.Fatalf("failed to get column types: %v", err)
	}

	scanSlice := GetScanSlice(columnTypes)

	if len(scanSlice) != len(columns) {
		t.Errorf("GetScanSlice() returned a slice of length %d, but expected %d", len(scanSlice), len(columns))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
