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
	"strconv"
	"strings"
)

const (
	branchIdPrefix = "-"
)

type XABranchXid struct {
	xid                 string
	branchId            uint64
	globalTransactionId []byte
	branchQualifier     []byte
}

type Option func(*XABranchXid)

func NewXABranchXid(opt ...Option) *XABranchXid {
	xABranchXid := &XABranchXid{}

	for _, fn := range opt {
		fn(xABranchXid)
	}

	// encode
	if (xABranchXid.xid != "" || xABranchXid.branchId != 0) &&
		len(xABranchXid.globalTransactionId) == 0 &&
		len(xABranchXid.branchQualifier) == 0 {
		encode(xABranchXid)
	}

	// decode
	if xABranchXid.xid == "" && xABranchXid.branchId == 0 &&
		(len(xABranchXid.globalTransactionId) > 0 || len(xABranchXid.branchQualifier) > 0) {
		decode(xABranchXid)
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

func (x *XABranchXid) String() string {
	return x.xid + branchIdPrefix + strconv.FormatUint(x.branchId, 10)
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
