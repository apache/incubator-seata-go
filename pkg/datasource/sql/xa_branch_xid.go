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
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

const (
	branchIdPrefix        = "-"
	MaxGTRIDLength        = 64
	MaxBQUALLength        = 64
	DefaultFormatId int32 = 1
)

type XABranchXid struct {
	xid                 string
	branchId            uint64
	globalTransactionId []byte
	branchQualifier     []byte

	formatId int32
	dbType   types.DBType
}

type Option func(*XABranchXid)

func NewXABranchXid(opt ...Option) *XABranchXid {
	xABranchXid := &XABranchXid{
		formatId: DefaultFormatId,
		dbType:   types.DBTypeMySQL,
	}

	for _, fn := range opt {
		fn(xABranchXid)
	}

	switch xABranchXid.dbType {
	case types.DBTypePostgreSQL:
		if (xABranchXid.xid != "" || xABranchXid.branchId != 0) &&
			len(xABranchXid.globalTransactionId) == 0 &&
			len(xABranchXid.branchQualifier) == 0 {
			encodePostgreSQL(xABranchXid)
		} else if xABranchXid.xid == "" && xABranchXid.branchId == 0 &&
			(len(xABranchXid.globalTransactionId) > 0 || len(xABranchXid.branchQualifier) > 0) {
			decodePostgreSQL(xABranchXid)
		}
	default:
		if (xABranchXid.xid != "" || xABranchXid.branchId != 0) &&
			len(xABranchXid.globalTransactionId) == 0 &&
			len(xABranchXid.branchQualifier) == 0 {
			encode(xABranchXid)
		}
		if xABranchXid.xid == "" && xABranchXid.branchId == 0 &&
			(len(xABranchXid.globalTransactionId) > 0 || len(xABranchXid.branchQualifier) > 0) {
			decode(xABranchXid)
		}
	}

	return xABranchXid
}

func (x *XABranchXid) GetGlobalXid() string {
	return x.xid
}

func (x *XABranchXid) GetBranchId() uint64 {
	return x.branchId
}

func (x *XABranchXid) GetGlobalTransactionId() []byte {
	return x.globalTransactionId
}

func (x *XABranchXid) GetBranchQualifier() []byte {
	return x.branchQualifier
}

func (x *XABranchXid) GetFormatId() int32 {
	return x.formatId
}

func (x *XABranchXid) GetDatabaseType() types.DBType {
	return x.dbType
}

func (x *XABranchXid) String() string {
	switch x.dbType {
	case types.DBTypePostgreSQL:
		return x.ToPostgreSQLFormat()
	default:
		return x.xid + branchIdPrefix + strconv.FormatUint(x.branchId, 10)
	}
}

func (x *XABranchXid) ToPostgreSQLFormat() string {
	gtridStr := string(x.globalTransactionId)
	bqualStr := string(x.branchQualifier)
	formatID := x.formatId

	if gtridStr == "" || bqualStr == "" {
		return ""
	}
	
	return fmt.Sprintf("%s,%s,%d", gtridStr, bqualStr, formatID)
}

func WithXid(xid string) Option {
	return func(x *XABranchXid) {
		x.xid = xid
	}
}

func WithBranchId(branchId uint64) Option {
	return func(x *XABranchXid) {
		x.branchId = branchId
	}
}

func WithGlobalTransactionId(globalTransactionId []byte) Option {
	return func(x *XABranchXid) {
		x.globalTransactionId = globalTransactionId
	}
}

func WithBranchQualifier(branchQualifier []byte) Option {
	return func(x *XABranchXid) {
		x.branchQualifier = branchQualifier
	}
}

func WithFormatId(formatId int32) Option {
	return func(x *XABranchXid) {
		x.formatId = formatId
	}
}

func WithDatabaseType(dbType types.DBType) Option {
	return func(x *XABranchXid) {
		x.dbType = dbType
	}
}

func encode(x *XABranchXid) {
	if x.xid != "" {
		x.globalTransactionId = []byte(x.xid)
	}

	if x.branchId != 0 {
		x.branchQualifier = []byte(branchIdPrefix + strconv.FormatUint(x.branchId, 10))
	}
}

func decode(x *XABranchXid) {
	if len(x.globalTransactionId) > 0 {
		x.xid = string(x.globalTransactionId)
	}

	if len(x.branchQualifier) > 0 {
		branchId := strings.TrimLeft(string(x.branchQualifier), branchIdPrefix)
		x.branchId, _ = strconv.ParseUint(branchId, 10, 64)
	}
}

func encodePostgreSQL(x *XABranchXid) {
	if x.xid != "" {
		gtrid := x.xid
		if len(gtrid) > MaxGTRIDLength {
			hash := md5.Sum([]byte(gtrid))
			gtrid = hex.EncodeToString(hash[:])
		}
		x.globalTransactionId = []byte(gtrid)
	} else {
		x.globalTransactionId = []byte("postgresql-default-gtrid")
	}

	branchBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(branchBytes, x.branchId)
	bqual := hex.EncodeToString(branchBytes)
	if len(bqual) > MaxBQUALLength {
		bqual = bqual[:MaxBQUALLength]
	}
	x.branchQualifier = []byte(bqual)

	if len(x.globalTransactionId) == 0 {
		x.globalTransactionId = []byte("postgresql-default-gtrid")
	}
	if len(x.branchQualifier) == 0 {
		x.branchQualifier = []byte("0000000000000000")
	}
}

func decodePostgreSQL(x *XABranchXid) {
	if len(x.globalTransactionId) > 0 {
		x.xid = string(x.globalTransactionId)
	}

	if len(x.branchQualifier) > 0 {
		bqualStr := string(x.branchQualifier)
		if branchBytes, err := hex.DecodeString(bqualStr); err == nil && len(branchBytes) >= 8 {
			x.branchId = binary.BigEndian.Uint64(branchBytes[:8])
		}
	}
}

func ParsePostgreSQLXid(xidStr string) (*XABranchXid, error) {
	parts := strings.Split(xidStr, "_")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid PostgreSQL XID format")
	}

	formatId, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid format ID: %v", err)
	}

	gtridLen, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid gtrid length: %v", err)
	}

	bqualLen, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid bqual length: %v", err)
	}

	gtrid, err := hex.DecodeString(parts[3])
	if err != nil || len(gtrid) != gtridLen {
		return nil, fmt.Errorf("invalid gtrid: %v", err)
	}

	bqual, err := hex.DecodeString(parts[4])
	if err != nil || len(bqual) != bqualLen {
		return nil, fmt.Errorf("invalid bqual: %v", err)
	}

	return NewXABranchXid(
		WithGlobalTransactionId(gtrid),
		WithBranchQualifier(bqual),
		WithFormatId(int32(formatId)),
		WithDatabaseType(types.DBTypePostgreSQL),
	), nil
}
