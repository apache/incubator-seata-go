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
}

func TestXABranchXidBuildWithByte(t *testing.T) {
	xid := []byte("111")
	branchId := []byte(branchIdPrefix + "222")
	x := XaIdBuildWithByte(xid, branchId)
	assert.Equal(t, x.GetGlobalTransactionId(), xid)
	assert.Equal(t, x.GetBranchQualifier(), branchId)

	assert.Equal(t, x.GetGlobalXid(), "111")
	assert.Equal(t, x.GetBranchId(), uint64(222))
}
