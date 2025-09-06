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

package sql

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXABranchXidBuild(t *testing.T) {
	xid := "111"
	branchId := uint64(222)
	x := XaIdBuild(xid, branchId)

	assert.Equal(t, x.GetGlobalXid(), xid)
	assert.Equal(t, x.GetBranchId(), branchId)
	assert.Equal(t, x.GetGlobalTransactionId(), []byte(xid))
	assert.Equal(t, x.GetBranchQualifier(), []byte("-222"))
	assert.Equal(t, x.GetDatabaseType(), DatabaseTypeMySQL)
	assert.Equal(t, x.String(), "111-222")
}

func TestXABranchXidBuildWithByte(t *testing.T) {
	xid := []byte("111")
	branchId := []byte(branchIdPrefix + "222")
	x := XaIdBuildWithByte(xid, branchId)

	assert.Equal(t, x.GetGlobalTransactionId(), xid)
	assert.Equal(t, x.GetBranchQualifier(), branchId)
	assert.Equal(t, x.GetGlobalXid(), "111")
	assert.Equal(t, x.GetBranchId(), uint64(222))
	assert.Equal(t, x.GetDatabaseType(), DatabaseTypeMySQL)
}

func TestXABranchXidBuildForPostgreSQL(t *testing.T) {
	xid := "global-tx-123"
	branchId := uint64(456)
	x := XaIdBuildForPostgreSQL(xid, branchId)

	assert.Equal(t, x.GetGlobalXid(), xid)
	assert.Equal(t, x.GetBranchId(), branchId)
	assert.Equal(t, x.GetDatabaseType(), DatabaseTypePostgreSQL)
	assert.Equal(t, x.GetFormatId(), int32(DefaultFormatId))

	pgFormat := x.ToPostgreSQLFormat()
	assert.NotEmpty(t, pgFormat)
	assert.True(t, strings.HasPrefix(pgFormat, "1_"))

	assert.Equal(t, x.GetGlobalTransactionId(), []byte(xid))
	assert.NotEmpty(t, x.GetBranchQualifier())
}

func TestXABranchXidWithDatabase(t *testing.T) {
	xid := "test-xid"
	branchId := uint64(123)

	mysqlXid := XaIdBuildWithDatabase(xid, branchId, DatabaseTypeMySQL)
	assert.Equal(t, mysqlXid.GetDatabaseType(), DatabaseTypeMySQL)
	assert.Equal(t, mysqlXid.String(), "test-xid-123")

	pgXid := XaIdBuildWithDatabase(xid, branchId, DatabaseTypePostgreSQL)
	assert.Equal(t, pgXid.GetDatabaseType(), DatabaseTypePostgreSQL)
	pgFormat := pgXid.ToPostgreSQLFormat()
	assert.NotEmpty(t, pgFormat)
	assert.Contains(t, pgFormat, hex.EncodeToString([]byte(xid)))
}

func TestXABranchXidStandardBuild(t *testing.T) {
	gtrid := []byte("global-transaction-123")
	bqual := []byte("branch-qualifier-456")
	formatId := int32(2)

	x := XaIdBuildStandard(gtrid, bqual, formatId, DatabaseTypePostgreSQL)

	assert.Equal(t, x.GetGlobalTransactionId(), gtrid)
	assert.Equal(t, x.GetBranchQualifier(), bqual)
	assert.Equal(t, x.GetFormatId(), formatId)
	assert.Equal(t, x.GetDatabaseType(), DatabaseTypePostgreSQL)
	assert.Equal(t, x.GetGlobalXid(), string(gtrid))
}

func TestPostgreSQLXidParsing(t *testing.T) {

	originalXid := XaIdBuildForPostgreSQL("test-global-tx", 789)
	pgFormat := originalXid.ToPostgreSQLFormat()

	parsedXid, err := ParsePostgreSQLXid(pgFormat)
	assert.NoError(t, err)
	assert.NotNil(t, parsedXid)

	assert.Equal(t, parsedXid.GetGlobalTransactionId(), originalXid.GetGlobalTransactionId())
	assert.Equal(t, parsedXid.GetBranchQualifier(), originalXid.GetBranchQualifier())
	assert.Equal(t, parsedXid.GetFormatId(), originalXid.GetFormatId())
	assert.Equal(t, parsedXid.GetDatabaseType(), DatabaseTypePostgreSQL)
}

func TestPostgreSQLXidParsingInvalidFormat(t *testing.T) {
	invalidFormats := []string{
		"",
		"invalid",
		"1_2_3",
		"1_2_3_4",
		"abc_2_3_4_5",
		"1_abc_3_4_5",
		"1_2_abc_4_5",
		"1_2_3_invalidhex_5",
		"1_2_3_616263_invalidhex",
	}

	for _, invalid := range invalidFormats {
		_, err := ParsePostgreSQLXid(invalid)
		assert.Error(t, err, "Expected error for invalid format: %s", invalid)
	}
}

func TestLongXidHandling(t *testing.T) {
	longXid := strings.Repeat("a", MaxGTRIDLength+10)
	x := XaIdBuildForPostgreSQL(longXid, 123)

	assert.NotEmpty(t, x.GetGlobalTransactionId())
	assert.LessOrEqual(t, len(x.GetGlobalTransactionId()), MaxGTRIDLength)

	pgFormat := x.ToPostgreSQLFormat()
	assert.NotEmpty(t, pgFormat)
}

func TestXidStringFormats(t *testing.T) {
	xid := "test-xid"
	branchId := uint64(456)

	mysqlXid := XaIdBuild(xid, branchId)
	assert.Equal(t, mysqlXid.String(), "test-xid-456")

	pgXid := XaIdBuildForPostgreSQL(xid, branchId)
	pgString := pgXid.String()
	assert.NotEqual(t, pgString, "test-xid-456")
	assert.Contains(t, pgString, "1_")
}

func TestEncodeDecodeRoundTrip(t *testing.T) {

	mysqlXid := XaIdBuild("mysql-test", 789)
	mysqlBytes := XaIdBuildWithByte(mysqlXid.GetGlobalTransactionId(), mysqlXid.GetBranchQualifier())
	assert.Equal(t, mysqlXid.GetGlobalXid(), mysqlBytes.GetGlobalXid())
	assert.Equal(t, mysqlXid.GetBranchId(), mysqlBytes.GetBranchId())

	pgXid := XaIdBuildForPostgreSQL("pg-test", 789)
	pgFormat := pgXid.ToPostgreSQLFormat()
	parsedPgXid, err := ParsePostgreSQLXid(pgFormat)
	assert.NoError(t, err)
	assert.Equal(t, pgXid.GetGlobalXid(), parsedPgXid.GetGlobalXid())
}

func TestDatabaseTypeConstants(t *testing.T) {
	assert.Equal(t, DatabaseTypeMySQL, DatabaseType(0))
	assert.Equal(t, DatabaseTypePostgreSQL, DatabaseType(1))
}

func TestDefaultValues(t *testing.T) {
	x := NewXABranchXid()
	assert.Equal(t, x.GetDatabaseType(), DatabaseTypeMySQL)
	assert.Equal(t, x.GetFormatId(), int32(DefaultFormatId))
	assert.Equal(t, DefaultFormatId, int32(1))
}
