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

package constant

const (
	ActionStartTime = "action-start-time"
	HostName        = "host-name"
	ActionContext   = "actionContext"

	PrepareMethod  = "sys::prepare"
	CommitMethod   = "sys::commit"
	RollbackMethod = "sys::rollback"
	ActionName     = "actionName"

	ActionMethod       = "sys::action"
	CompensationMethod = "sys::compensationAction"

	SeataXidKey     = "SEATA_XID"
	XidKey          = "TX_XID"
	XidKeyLowercase = "tx_xid"
	MdcXidKey       = "X-TX-XID"
	MdcBranchIDKey  = "X-TX-BRANCH-ID"
	BranchTypeKey   = "TX_BRANCH_TYPE"
	GlobalLockKey   = "TX_LOCK"
	SeataFilterKey  = "seataDubboFilter"

	SeataVersion = "1.1.0"

	TccBusinessActionContextParameter  = "tccParam"
	SagaBusinessActionContextParameter = "sagaParam"
)
