package sql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestXABranchXidBuild(t *testing.T) {
	xid := "111"
	branchId := uint64(222)

	x := NewXABranchXid(WithXid(xid), WithBranchId(branchId))

	assert.Equal(t, x.GetGlobalXid(), xid)
	assert.Equal(t, x.GetBranchId(), branchId)
	assert.Equal(t, x.GetGlobalTransactionId(), []byte(xid))
	assert.Equal(t, x.GetBranchQualifier(), []byte("-222"))
	assert.Equal(t, x.GetDatabaseType(), types.DBTypeMySQL)
	assert.Equal(t, x.String(), "111-222")
}

func TestXABranchXidBuildWithByte(t *testing.T) {
	xidByte := []byte("111")
	branchIdByte := []byte(branchIdPrefix + "222")
	x := NewXABranchXid(
		WithGlobalTransactionId(xidByte),
		WithBranchQualifier(branchIdByte),
	)

	assert.Equal(t, x.GetGlobalTransactionId(), xidByte)
	assert.Equal(t, x.GetBranchQualifier(), branchIdByte)
	assert.Equal(t, x.GetGlobalXid(), "111")
	assert.Equal(t, x.GetBranchId(), uint64(222))
	assert.Equal(t, x.GetDatabaseType(), types.DBTypeMySQL)
}

func TestXABranchXidBuildForPostgreSQL(t *testing.T) {
	xid := "global-tx-123"
	branchId := uint64(456)
	x := NewXABranchXid(
		WithXid(xid),
		WithBranchId(branchId),
		WithDatabaseType(types.DBTypePostgreSQL),
	)

	assert.Equal(t, x.GetGlobalXid(), xid)
	assert.Equal(t, x.GetBranchId(), branchId)
	assert.Equal(t, x.GetDatabaseType(), types.DBTypePostgreSQL)
	assert.Equal(t, x.GetFormatId(), int32(DefaultFormatId))

	pgFormat := x.ToPostgreSQLFormat()
	assert.NotEmpty(t, pgFormat)
	assert.True(t, strings.HasPrefix(pgFormat, xid+branchIdPrefix))
	assert.Equal(t, x.GetGlobalTransactionId(), []byte(xid))
	assert.NotEmpty(t, x.GetBranchQualifier())
}

func TestXABranchXidWithDatabase(t *testing.T) {
	xid := "test-xid"
	branchId := uint64(123)

	mysqlXid := NewXABranchXid(
		WithXid(xid),
		WithBranchId(branchId),
		WithDatabaseType(types.DBTypeMySQL),
	)
	assert.Equal(t, mysqlXid.GetDatabaseType(), types.DBTypeMySQL)
	assert.Equal(t, mysqlXid.String(), "test-xid-123")

	pgXid := NewXABranchXid(
		WithXid(xid),
		WithBranchId(branchId),
		WithDatabaseType(types.DBTypePostgreSQL),
	)
	assert.Equal(t, pgXid.GetDatabaseType(), types.DBTypePostgreSQL)
	pgFormat := pgXid.ToPostgreSQLFormat()
	assert.NotEmpty(t, pgFormat)
	assert.Contains(t, pgFormat, xid)
}

func TestXABranchXidStandardBuild(t *testing.T) {
	gtrid := []byte("global-transaction-123")
	bqual := []byte("branch-qualifier-456")
	formatId := int32(2)

	x := NewXABranchXid(
		WithGlobalTransactionId(gtrid),
		WithBranchQualifier(bqual),
		WithFormatId(formatId),
		WithDatabaseType(types.DBTypePostgreSQL),
	)

	assert.Equal(t, x.GetGlobalTransactionId(), gtrid)
	assert.Equal(t, x.GetBranchQualifier(), bqual)
	assert.Equal(t, x.GetFormatId(), formatId)
	assert.Equal(t, x.GetDatabaseType(), types.DBTypePostgreSQL)
	assert.Equal(t, x.GetGlobalXid(), string(gtrid))
}

func TestPostgreSQLXidParsing(t *testing.T) {
	originalXid := NewXABranchXid(
		WithXid("test-global-tx"),
		WithBranchId(789),
		WithDatabaseType(types.DBTypePostgreSQL),
	)
	pgFormat := originalXid.ToPostgreSQLFormat()

	parsedXid, err := ParsePostgreSQLXid(pgFormat)
	assert.NoError(t, err)
	assert.NotNil(t, parsedXid)

	assert.Equal(t, parsedXid.GetGlobalTransactionId(), originalXid.GetGlobalTransactionId())
	assert.Equal(t, parsedXid.GetBranchQualifier(), originalXid.GetBranchQualifier())
	assert.Equal(t, parsedXid.GetFormatId(), originalXid.GetFormatId())
	assert.Equal(t, parsedXid.GetDatabaseType(), types.DBTypePostgreSQL)
}

func TestPostgreSQLXidParsingInvalidFormat(t *testing.T) {
	invalidFormats := []string{
		"",
	}

	for _, invalid := range invalidFormats {
		_, err := ParsePostgreSQLXid(invalid)
		assert.Error(t, err, "Expected error for invalid format: %s", invalid)
	}

	validFormats := []string{
		"invalid",
		"1_2_3",
		"1_2_3_4",
		"abc_2_3_4_5",
		"1_abc_3_4_5",
		"1_2_abc_4_5",
		"1_2_3_invalidhex_5",
		"1_2_3_616263_invalidhex",
		"test-xid-123",
		"simple_xid",
	}

	for _, valid := range validFormats {
		_, err := ParsePostgreSQLXid(valid)
		assert.NoError(t, err, "Expected no error for valid format: %s", valid)
	}
}

func TestLongXidHandling(t *testing.T) {
	longXid := strings.Repeat("a", MaxGTRIDLength+10)
	x := NewXABranchXid(
		WithXid(longXid),
		WithBranchId(123),
		WithDatabaseType(types.DBTypePostgreSQL),
	)

	assert.NotEmpty(t, x.GetGlobalTransactionId())
	assert.LessOrEqual(t, len(x.GetGlobalTransactionId()), MaxGTRIDLength)

	pgFormat := x.ToPostgreSQLFormat()
	assert.NotEmpty(t, pgFormat)
}

func TestXidStringFormats(t *testing.T) {
	xid := "test-xid"
	branchId := uint64(456)

	mysqlXid := NewXABranchXid(WithXid(xid), WithBranchId(branchId))
	assert.Equal(t, mysqlXid.String(), "test-xid-456")

	pgXid := NewXABranchXid(
		WithXid(xid),
		WithBranchId(branchId),
		WithDatabaseType(types.DBTypePostgreSQL),
	)
	pgString := pgXid.String()
	assert.Equal(t, pgString, "test-xid-456")
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	mysqlXid := NewXABranchXid(WithXid("mysql-test"), WithBranchId(789))
	mysqlBytes := NewXABranchXid(
		WithGlobalTransactionId(mysqlXid.GetGlobalTransactionId()),
		WithBranchQualifier(mysqlXid.GetBranchQualifier()),
	)
	assert.Equal(t, mysqlXid.GetGlobalXid(), mysqlBytes.GetGlobalXid())
	assert.Equal(t, mysqlXid.GetBranchId(), mysqlBytes.GetBranchId())

	pgXid := NewXABranchXid(
		WithXid("pg-test"),
		WithBranchId(789),
		WithDatabaseType(types.DBTypePostgreSQL),
	)
	pgFormat := pgXid.ToPostgreSQLFormat()
	parsedPgXid, err := ParsePostgreSQLXid(pgFormat)
	assert.NoError(t, err)
	assert.Equal(t, pgXid.GetGlobalXid(), parsedPgXid.GetGlobalXid())
}

func TestDefaultValues(t *testing.T) {
	x := NewXABranchXid()
	assert.Equal(t, x.GetDatabaseType(), types.DBTypeMySQL)
	assert.Equal(t, x.GetFormatId(), int32(DefaultFormatId))
	assert.Equal(t, DefaultFormatId, int32(1))
}
