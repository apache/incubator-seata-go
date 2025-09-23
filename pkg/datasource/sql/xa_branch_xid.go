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
	if x.xid != "" {
		return x.xid + branchIdPrefix + strconv.FormatUint(x.branchId, 10)
	}
	return string(x.globalTransactionId) + branchIdPrefix + strconv.FormatUint(x.branchId, 10)
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
		xidBytes := []byte(x.xid)
		if len(xidBytes) > MaxGTRIDLength {
			xidBytes = xidBytes[:MaxGTRIDLength]
		}
		x.globalTransactionId = xidBytes
	} else {
		x.globalTransactionId = []byte("postgresql-default-gtrid")
	}
	
	x.branchQualifier = []byte(strconv.FormatUint(x.branchId, 10))
}

func decodePostgreSQL(x *XABranchXid) {
	if len(x.globalTransactionId) > 0 {
		x.xid = string(x.globalTransactionId)
	}

	if len(x.branchQualifier) > 0 {
		bqualStr := string(x.branchQualifier)
		if branchId, err := strconv.ParseUint(bqualStr, 10, 64); err == nil {
			x.branchId = branchId
		}
	}
}

func ParsePostgreSQLXid(xidStr string) (*XABranchXid, error) {
	if xidStr == "" {
		return nil, fmt.Errorf("empty PostgreSQL XID")
	}

	branchIdIndex := strings.LastIndex(xidStr, branchIdPrefix)
	if branchIdIndex == -1 {
		return NewXABranchXid(
			WithXid(xidStr),
			WithBranchId(0),
			WithDatabaseType(types.DBTypePostgreSQL),
		), nil
	}

	xid := xidStr[:branchIdIndex]
	branchIdStr := xidStr[branchIdIndex+len(branchIdPrefix):]
	
	branchId, err := strconv.ParseUint(branchIdStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid branch ID: %v", err)
	}

	return NewXABranchXid(
		WithXid(xid),
		WithBranchId(branchId),
		WithDatabaseType(types.DBTypePostgreSQL),
	), nil
}
