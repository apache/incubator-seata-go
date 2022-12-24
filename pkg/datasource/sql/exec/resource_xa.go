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

package exec

import (
	"time"
)

const (
	// TMENDICANT Ends a recovery scan.
	TMENDRSCAN = 0x00800000

	/**
	 * Disassociates the caller and marks the transaction branch
	 * rollback-only.
	 */
	TMFAIL = 0x20000000

	/**
	 * Caller is joining existing transaction branch.
	 */
	TMJOIN = 0x00200000

	/**
	 * Use TMNOFLAGS to indicate no flags value is selected.
	 */
	TMNOFLAGS = 0x00000000

	/**
	 * Caller is using one-phase optimization.
	 */
	TMONEPHASE = 0x40000000

	/**
	 * Caller is resuming association with a suspended
	 * transaction branch.
	 */
	TMRESUME = 0x08000000

	/**
	 * Starts a recovery scan.
	 */
	TMSTARTRSCAN = 0x01000000

	/**
	 * Disassociates caller from a transaction branch.
	 */
	TMSUCCESS = 0x04000000

	/**
	 * Caller is suspending (not ending) its association with
	 * a transaction branch.
	 */
	TMSUSPEND = 0x02000000

	/**
	 * The transaction branch has been read-only and has been committed.
	 */
	XA_RDONLY = 0x00000003

	/**
	 * The transaction work has been prepared normally.
	 */
	XA_OK = 0
)

type XAResource interface {
	Commit(xid string, onePhase bool) error
	End(xid string, flags int) error
	Forget(xid string) error
	GetTransactionTimeout() time.Duration
	IsSameRM(resource XAResource) bool
	XAPrepare(xid string) error
	Recover(flag int) ([]string, error)
	Rollback(xid string) error
	SetTransactionTimeout(duration time.Duration) bool
	Start(xid string, flags int) error
}
