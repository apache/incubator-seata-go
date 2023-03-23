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

package xa

import (
	"context"
	"time"
)

const (
	// TMEndRScan ends a recovery scan.
	TMEndRScan = 0x00800000
	// TMFail disassociates the caller and marks the transaction branch
	// rollback-only.
	TMFail = 0x20000000

	// TMJoin joining existing transaction branch.
	TMJoin = 0x00200000

	// TMNoFlags indicate no flags value is selected.
	TMNoFlags = 0x00000000

	// TMOnePhase using one-phase optimization.
	TMOnePhase = 0x40000000

	// TMResume is resuming association with a suspended transaction branch.
	TMResume = 0x08000000

	// TMStartRScan starts a recovery scan.
	TMStartRScan = 0x01000000

	// TMSuccess disassociates caller from a transaction branch.
	TMSuccess = 0x04000000

	// TMSuspend is suspending (not ending) its association with a transaction branch.
	TMSuspend = 0x02000000

	// XAReadOnly the transaction branch has been read-only and has been committed.
	XAReadOnly = 0x00000003

	// XAOk The transaction work has been prepared normally.
	XAOk = 0
)

type XAResource interface {
	Commit(ctx context.Context, xid string, onePhase bool) error
	End(ctx context.Context, xid string, flags int) error
	Forget(ctx context.Context, xid string) error
	GetTransactionTimeout() time.Duration
	IsSameRM(ctx context.Context, resource XAResource) bool
	XAPrepare(ctx context.Context, xid string) error
	Recover(ctx context.Context, flag int) ([]string, error)
	Rollback(ctx context.Context, xid string) error
	SetTransactionTimeout(duration time.Duration) bool
	Start(ctx context.Context, xid string, flags int) error
}
