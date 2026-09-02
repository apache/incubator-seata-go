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
	"context"
	gosql "database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"time"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

var xaConnTimeout time.Duration

var errXABranchLifecycleManaged = errors.New("xa branch lifecycle is managed by XATx or XAConn")

// XAConn Database connection proxy object under XA transaction model
// Conn is assumed to be stateful.
type XAConn struct {
	*Conn

	tx                  driver.Tx
	xaResource          xa.XAResource
	xaErrorClassifier   xa.XAErrorClassifier
	xaBranchXid         *XABranchXid
	xaActive            bool
	rollBacked          bool
	branchRegisterTime  time.Time
	prepareTime         time.Time
	keeperMu            sync.Mutex
	keepInKeeper        bool
	shouldBeHeld        bool
	isConnKept          bool
	preparedForPhaseTwo bool
	poolDiscarded       bool
	physicalClosed      bool
}

// xaBranchTx is a sentinel driver.Tx used to satisfy database/sql wiring while
// the real XA branch lifecycle is driven by XATx/XAConn through XA START/END/PREPARE.
// Any direct Commit/Rollback on this placeholder indicates the caller bypassed the XA flow.
type xaBranchTx struct{}

func (xaBranchTx) Commit() error {
	return errXABranchLifecycleManaged
}

func (xaBranchTx) Rollback() error {
	return errXABranchLifecycleManaged
}

func (c *XAConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	//ret, err := c.createNewTxOnExecIfNeed(ctx, func() (types, error) {
	//	ret, err := c.Conn.PrepareContext(ctx, query)
	//	if err != nil {
	//		return nil, err
	//	}
	//	return types.NewResult(types.WithRows(ret)), nil
	//})

	return c.Conn.PrepareContext(ctx, query)
}

// QueryContext exec xa sql
func (c *XAConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	ret, err := c.createNewTxOnExecIfNeed(ctx, true, query, args, func() (types.ExecResult, error) {
		ret, err := c.Conn.QueryContext(ctx, query, args)
		if err != nil {
			return nil, err
		}
		return types.NewResult(types.WithRows(ret)), nil
	})
	if err != nil {
		return nil, err
	}
	return ret.GetRows(), nil
}

func (c *XAConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	ret, err := c.createNewTxOnExecIfNeed(ctx, false, query, args, func() (types.ExecResult, error) {
		ret, err := c.Conn.ExecContext(ctx, query, args)
		if err != nil {
			return nil, err
		}
		return types.NewResult(types.WithResult(ret)), nil
	})

	if err != nil {
		return nil, err
	}

	return ret.GetResult(), nil
}

// BeginTx like common transaction. but it just exec XA START
func (c *XAConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if !tm.IsGlobalTx(ctx) {
		tx, err := c.Conn.BeginTx(ctx, opts)
		return tx, err
	}

	c.autoCommit = false

	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts
	c.txCtx.ResourceID = c.res.resourceID
	c.txCtx.XID = tm.GetXID(ctx)
	c.txCtx.TransactionMode = types.XAMode

	// Keep a sentinel target in Tx so any accidental fallback to the generic
	// driver.Tx path fails fast instead of silently masking XA lifecycle bugs.
	branchTx := xaBranchTx{}
	c.tx = branchTx

	tx, err := newTx(
		withDriverConn(c.Conn),
		withTxCtx(c.txCtx),
		withOriginTx(branchTx),
		withXAConn(c),
	)
	if err != nil {
		return nil, err
	}

	if c.xaActive {
		return nil, errors.New("should NEVER happen: setAutoCommit from true to false while xa branch is active")
	}

	baseTx, ok := tx.(*Tx)
	if !ok {
		return nil, fmt.Errorf("start xa %s transaction failure for the tx is a wrong type", c.txCtx.XID)
	}

	baseTx.xaConn = c

	c.branchRegisterTime = time.Now()
	if err := baseTx.register(c.txCtx); err != nil {
		c.cleanXABranchContext()
		return nil, fmt.Errorf("failed to register xa branch %s, err:%w", c.txCtx.XID, err)
	}

	c.xaBranchXid = XaIdBuild(c.txCtx.XID, c.txCtx.BranchID)
	c.configureConnectionHold(ctx)
	c.keepIfNecessary()

	if err = c.start(ctx); err != nil {
		c.releaseIfNecessary()
		c.cleanXABranchContext()
		return nil, fmt.Errorf("failed to start xa branch xid:%s err:%w", c.txCtx.XID, err)
	}
	c.xaActive = true

	return &XATx{tx: tx.(*Tx)}, nil
}

func (c *XAConn) createOnceTxContext(ctx context.Context) bool {
	onceTx := tm.IsGlobalTx(ctx) && c.autoCommit

	if onceTx {
		c.txCtx = types.NewTxCtx()
		c.txCtx.DBType = c.res.dbType
		c.txCtx.ResourceID = c.res.resourceID
		c.txCtx.XID = tm.GetXID(ctx)
		c.txCtx.TransactionMode = types.XAMode
		c.txCtx.GlobalLockRequire = true
	}

	return onceTx
}

func (c *XAConn) createNewTxOnExecIfNeed(ctx context.Context, isQuery bool, query string, args []driver.NamedValue, f func() (types.ExecResult, error)) (types.ExecResult, error) {
	var (
		tx           driver.Tx
		err          error
		xaRollbacked bool // Track if XA rollback was already done to avoid duplicate rollback
	)

	defer func() {
		recoverErr := recover()
		// Check if error is ErrSkip - don't rollback for this special error
		isErrSkip := err != nil && errors.Is(err, driver.ErrSkip)

		if (err != nil && !isErrSkip) || recoverErr != nil {
			// Prefer XATx.Rollback so a registered branch reports phase-1 failure to
			// the TC; fall back to the raw connection rollback for non-autoCommit paths.
			if !xaRollbacked {
				if tx != nil {
					if rollbackErr := tx.Rollback(); rollbackErr != nil {
						log.Errorf("defer rollback xa branch error:%v", rollbackErr)
					}
					xaRollbacked = true
				} else if c.xaActive {
					if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
						log.Errorf("defer rollback xa branch error:%v", rollbackErr)
					}
					xaRollbacked = true
				}
			}
		}
	}()

	currentAutoCommit := c.autoCommit

	// For global transactions in autoCommit mode, each statement is a complete XA branch
	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		tx, err = c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			return nil, err
		}
	}

	// execute SQL
	ret, err := f()
	if err != nil && errors.Is(err, driver.ErrSkip) {
		// driver.ErrSkip is not a real failure: with the default go-sql-driver DSN
		// (interpolateParams=false) the direct Execer/Queryer answers ErrSkip for any
		// statement carrying bind parameters, asking database/sql to retry it through
		// the Prepare+Exec fallback path.
		if tx == nil {
			// No XA branch opened for this statement - safe to hand the retry back to
			// database/sql and let it run its own Prepare+Exec fallback.
			return nil, err
		}
		// We already opened an XA branch (XA START) for this statement. We cannot hand
		// the retry back to database/sql: it would run on a *different* pooled
		// connection (autoCommit is now false and txCtx has been reset), leaving this
		// branch registered-but-never-prepared (a leak) and letting the retried write
		// escape the global transaction. Instead run the Prepare+Exec fallback
		// OURSELVES on this same physical connection, which still holds XA START open,
		// so the retried statement stays inside the branch and the normal XA END +
		// XA PREPARE commit below applies unchanged. These helpers never return ErrSkip.
		if isQuery {
			ret, err = c.queryPreparedInBranch(ctx, query, args)
		} else {
			ret, err = c.execPreparedInBranch(ctx, query, args)
		}
	}
	if err != nil {
		// On real error, rollback the entire branch. Prefer XATx.Rollback so the
		// already-registered branch reports phase-1 failure to the TC; fall back to
		// the raw connection rollback for non-autoCommit paths.
		if tx != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Errorf("failed to rollback xa branch of :%s, err:%v", c.txCtx.XID, rollbackErr)
			}
		} else if c.xaActive {
			if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
				log.Errorf("failed to rollback xa branch of :%s, err:%v", c.txCtx.XID, rollbackErr)
			}
		}
		xaRollbacked = true // Mark that rollback was handled
		return nil, err
	}

	// For autoCommit mode with global transaction, commit the branch now:
	// XA END + XA PREPARE + report phase-1 success to TC.
	if tx != nil && currentAutoCommit {
		// A query statement returns an open result set that still occupies this
		// connection's read buffer. Running XA END + XA PREPARE here - before the
		// caller has drained/closed the rows - issues a new command on top of that
		// unread result set, which go-sql-driver rejects as a "busy buffer" /
		// "commands out of sync" error and database/sql surfaces as
		// "driver: bad connection", forcing the whole transaction to roll back
		// (issue #904, e.g. SELECT ... FOR UPDATE followed by UPDATE). Defer the
		// branch commit until the caller closes the rows, mirroring AT mode's
		// RowsCommitOnClose handling. xaDeferredCommitTx keeps the inline path's
		// rollback-on-commit-failure semantics so a failed deferred commit never
		// leaves a prepared branch holding locks. (GetRows must only be called on
		// a query result - it panics on a write result - hence the isQuery gate.)
		if isQuery {
			if dr := ret.GetRows(); dr != nil {
				return types.NewResult(types.WithRows(&RowsCommitOnClose{
					rows: dr,
					tx:   xaDeferredCommitTx{tx: tx},
				})), nil
			}
		}
		if err = tx.Commit(); err != nil {
			log.Errorf("xa transaction commit failure xid:%s, err:%v", c.txCtx.XID, err)
			// XA End & Rollback
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Errorf("xa transaction rollback failure xid:%s, err:%v", c.txCtx.XID, rollbackErr)
			}
			xaRollbacked = true
			return nil, err
		}
	}

	return ret, nil
}

// execPreparedInBranch runs an ExecContext statement through the driver's
// Prepare+Exec path on the XAConn's own physical connection, which is still inside
// the open XA branch (XA START has been issued and not yet ended). It exists so a
// statement that answers driver.ErrSkip on the direct Execer path - the default
// go-sql-driver behavior for parameterized statements - can still be executed
// without handing the retry back to database/sql, which would run it on a different
// connection outside the branch. Unlike the direct path this never returns
// driver.ErrSkip: it either produces a concrete result or a concrete error.
func (c *XAConn) execPreparedInBranch(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
	preparer, ok := c.Conn.targetConn.(driver.ConnPrepareContext)
	if !ok {
		return nil, fmt.Errorf("xa branch %s: driver connection does not support PrepareContext, cannot recover from ErrSkip", c.txCtx.XID)
	}
	stmt, err := preparer.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	execer, ok := stmt.(driver.StmtExecContext)
	if !ok {
		return nil, fmt.Errorf("xa branch %s: prepared statement does not support ExecContext, cannot recover from ErrSkip", c.txCtx.XID)
	}
	res, err := execer.ExecContext(ctx, args)
	if err != nil {
		return nil, err
	}
	return types.NewResult(types.WithResult(res)), nil
}

// queryPreparedInBranch is the QueryContext counterpart of execPreparedInBranch.
// The prepared statement must outlive the result set, so it is wrapped in
// rowsWithStmt, which closes the statement when the rows are closed (this composes
// with RowsCommitOnClose: draining the rows closes both the driver rows and the
// statement and then runs the deferred XA END + XA PREPARE).
func (c *XAConn) queryPreparedInBranch(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
	preparer, ok := c.Conn.targetConn.(driver.ConnPrepareContext)
	if !ok {
		return nil, fmt.Errorf("xa branch %s: driver connection does not support PrepareContext, cannot recover from ErrSkip", c.txCtx.XID)
	}
	stmt, err := preparer.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	queryer, ok := stmt.(driver.StmtQueryContext)
	if !ok {
		_ = stmt.Close()
		return nil, fmt.Errorf("xa branch %s: prepared statement does not support QueryContext, cannot recover from ErrSkip", c.txCtx.XID)
	}
	rows, err := queryer.QueryContext(ctx, args)
	if err != nil {
		_ = stmt.Close()
		return nil, err
	}
	return types.NewResult(types.WithRows(util.NewRowsWithStmt(rows, stmt))), nil
}

// xaDeferredCommitTx wraps an XA branch tx whose commit is deferred until the
// query's rows are closed (see createNewTxOnExecIfNeed / RowsCommitOnClose).
// It mirrors the inline exec path: if the deferred XA END + XA PREPARE (or the
// phase-1 report to the TC) fails, the branch is rolled back so it does not stay
// prepared and hold locks.
type xaDeferredCommitTx struct {
	tx driver.Tx
}

func (t xaDeferredCommitTx) Commit() error {
	if err := t.tx.Commit(); err != nil {
		log.Errorf("deferred xa branch commit failed, rolling back branch: %v", err)
		if rollbackErr := t.tx.Rollback(); rollbackErr != nil {
			log.Errorf("deferred xa branch rollback failed: %v", rollbackErr)
		}
		return err
	}
	return nil
}

func (t xaDeferredCommitTx) Rollback() error {
	return t.tx.Rollback()
}

// ResetSession is called by database/sql before reusing a pooled connection.
// XAConn.xaActive lives on the XA wrapper, so the embedded *Conn.ResetSession
// cannot clear it; without this override a connection whose previous autoCommit
// branch already completed phase-1 would keep xaActive=true and the next
// statement's BeginTx would fail the "xa branch is active" guard. Clearing it
// here (in addition to XAConn.Commit) is a defensive backstop for any path that
// leaves a stale flag. xaBranchXid is intentionally left untouched so a held
// branch remains available for phase-2.
func (c *XAConn) ResetSession(ctx context.Context) error {
	c.keeperMu.Lock()
	c.xaActive = false
	if c.physicalClosed || c.isConnKept {
		c.poolDiscarded = true
		c.keeperMu.Unlock()
		return driver.ErrBadConn
	}
	c.keeperMu.Unlock()

	return c.Conn.ResetSession(ctx)
}

func (c *XAConn) isKept() bool {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()
	return c.isConnKept
}

func (c *XAConn) configureConnectionHold(ctx context.Context) {
	c.keepInKeeper = c.res.keepInKeeper
	c.shouldBeHeld = c.res.IsShouldBeHeld()
	if !c.res.probeXADetachOnPrepare {
		return
	}

	detachOnPrepare, err := selectMySQLXADetachOnPrepare(ctx, c.Conn.targetConn)
	if err != nil {
		// The safe fallback is to retain the actual owner connection.
		c.shouldBeHeld = true
		log.Errorf("probe xa_detach_on_prepare on XA connection: %v", err)
		return
	}
	c.shouldBeHeld = !detachOnPrepare
}

func (c *XAConn) keepIfNecessary() {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()

	if c.xaBranchXid == nil {
		return
	}
	if c.keepInKeeper {
		if err := c.res.Hold(c.xaBranchXid.String(), c); err == nil {
			c.isConnKept = true
		}
	}
}

func (c *XAConn) releaseIfNecessary() {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()
	c.releaseIfNecessaryLocked()
}

func (c *XAConn) releaseIfNecessaryLocked() {
	if !c.isConnKept || c.xaBranchXid == nil {
		return
	}

	c.res.Release(c.xaBranchXid.String())
	c.isConnKept = false
}

func (c *XAConn) start(ctx context.Context) error {
	xaResource, err := xa.CreateXAResource(c.Conn.targetConn, c.dbType)
	if err != nil {
		return fmt.Errorf("create xa xid:%s resoruce err:%w", c.txCtx.XID, err)
	}
	c.xaResource = xaResource
	c.xaErrorClassifier = xa.CreateErrorClassifier(c.dbType)

	if err := c.xaResource.Start(ctx, c.xaBranchXid.String(), xa.TMNoFlags); err != nil {
		return fmt.Errorf("xa xid %s resource connection start err:%w", c.txCtx.XID, err)
	}

	if err := c.termination(c.xaBranchXid.String()); err != nil {
		c.xaResource.End(ctx, c.xaBranchXid.String(), xa.TMFail)
		c.XaRollback(ctx, c.xaBranchXid)
		return err
	}
	return nil
}

func (c *XAConn) end(ctx context.Context, flags int) error {
	err := c.xaResource.End(ctx, c.xaBranchXid.String(), flags)
	if err != nil {
		return err
	}
	err = c.termination(c.xaBranchXid.String())
	if err != nil {
		return err
	}
	return nil
}

func (c *XAConn) termination(xaBranchXid string) error {
	status, err := branchStatus(xaBranchXid)
	if err != nil {
		return fmt.Errorf("get XA branch status for [%s]: %w", xaBranchXid, err)
	}

	switch status {
	case branch.BranchStatusPhasetwoCommitted, branch.BranchStatusPhasetwoRollbacked:
		c.releaseIfNecessary()
		return fmt.Errorf(
			"XA branch [%v] has already terminated, branch status: [%v]",
			c.txCtx.XID,
			status,
		)
	}

	return nil
}

func (c *XAConn) cleanXABranchContext() {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()
	c.cleanXABranchContextLocked()
}

func (c *XAConn) cleanXABranchContextLocked() {
	h, _ := time.ParseDuration("-1000h")
	c.branchRegisterTime = time.Now().Add(h)
	c.prepareTime = time.Now().Add(h)
	c.preparedForPhaseTwo = false
	c.xaActive = false
	if !c.isConnKept {
		c.xaBranchXid = nil
	}
}

func (c *XAConn) Rollback(ctx context.Context) error {
	if c.autoCommit {
		return nil
	}

	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT rollback on an inactive session")
	}

	if !c.rollBacked {
		// First end the XA branch with TMFail
		if err := c.xaResource.End(ctx, c.xaBranchXid.String(), xa.TMFail); err != nil {
			// Handle XAER_RMFAIL exception - check if it's already ended
			if c.xaErrorClassifier.IsAlreadyEnded(err) {
				log.Infof("XA branch already ended, continuing with rollback for xid: %s", c.txCtx.XID)
				// Already ended, continue with rollback
			} else {
				return c.rollbackErrorHandle()
			}
		}

		// Then perform XA rollback
		if c.XaRollback(ctx, c.xaBranchXid) != nil {
			c.cleanXABranchContext()
			return c.rollbackErrorHandle()
		}
		c.rollBacked = true
	}
	c.cleanXABranchContext()

	return nil
}

func (c *XAConn) rollbackErrorHandle() error {
	return fmt.Errorf("failed to end(TMFAIL) xa branch on [%v] - [%v]", c.txCtx.XID, c.xaBranchXid.GetBranchId())
}

func (c *XAConn) Commit(ctx context.Context) error {
	if c.autoCommit {
		return nil
	}

	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT commit on an inactive session")
	}

	now := time.Now()

	if err := c.end(ctx, xa.TMSuccess); err != nil {
		return c.commitErrorHandle(ctx, err)
	}

	if err := c.checkTimeout(ctx, now); err != nil {
		return c.commitErrorHandle(ctx, err)
	}

	if err := c.xaResource.XAPrepare(ctx, c.xaBranchXid.String()); err != nil {
		return c.commitErrorHandle(ctx, err)
	}

	c.keeperMu.Lock()
	c.prepareTime = time.Now()
	c.preparedForPhaseTwo = true

	// Phase-1 is done: this session no longer has an in-flight XA branch. Clear
	// only the session-active flag so a subsequent autoCommit statement on a
	// connection that was not transferred to the keeper can open a fresh branch.
	// Held and temporarily cached branches remain addressable for phase two.
	c.xaActive = false
	c.keeperMu.Unlock()

	return nil
}

func (c *XAConn) commitErrorHandle(ctx context.Context, cause error) error {
	err := cause
	if rollbackErr := c.XaRollback(ctx, c.xaBranchXid); rollbackErr != nil {
		err = errors.Join(
			cause,
			fmt.Errorf("failed to rollback XA branch after commit failure xid:%s, err:%w", c.txCtx.XID, rollbackErr),
		)
	}
	c.cleanXABranchContext()
	return err
}

func (c *XAConn) ShouldBeHeld() bool {
	return c.shouldBeHeld || c.res.GetDbType() == types.DBTypeUnknown
}

func (c *XAConn) phaseTwoTimeoutSnapshot() (prepared bool, holdUntilPhaseTwo bool, preparedAt time.Time) {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()
	return c.preparedForPhaseTwo, c.shouldBeHeld, c.prepareTime
}

func (c *XAConn) checkTimeout(ctx context.Context, now time.Time) error {
	if now.Sub(c.branchRegisterTime) > xaConnTimeout {
		c.XaRollback(ctx, c.xaBranchXid)
		return fmt.Errorf("XA branch timeout error xid:%s", c.txCtx.XID)
	}
	return nil
}

func (c *XAConn) Close() error {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()

	c.rollBacked = false
	if c.isConnKept {
		c.poolDiscarded = true
		return nil
	}
	c.cleanXABranchContextLocked()
	return c.closePhysicalLocked()
}

func (c *XAConn) CloseForce() error {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()

	err := c.closePhysicalLocked()
	c.rollBacked = false
	c.releaseIfNecessaryLocked()
	c.cleanXABranchContextLocked()
	return err
}

func (c *XAConn) XaCommit(ctx context.Context, xaXid XAXid) error {
	return c.xaCommit(ctx, xaXid, nil)
}

func (c *XAConn) xaCommit(ctx context.Context, xaXid XAXid, beforeRelease func()) error {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()

	if c.physicalClosed {
		return driver.ErrBadConn
	}
	err := c.xaResource.Commit(ctx, xaXid.String(), false)
	if err == nil {
		if beforeRelease != nil {
			beforeRelease()
		}
		c.completePhaseTwoLocked()
	}
	return err
}

func (c *XAConn) XaRollbackByBranchId(ctx context.Context, xaXid XAXid) error {
	return c.XaRollback(ctx, xaXid)
}

func (c *XAConn) XaRollback(ctx context.Context, xaXid XAXid) error {
	return c.xaRollback(ctx, xaXid, nil)
}

func (c *XAConn) xaRollback(ctx context.Context, xaXid XAXid, beforeRelease func()) error {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()

	if c.physicalClosed {
		return driver.ErrBadConn
	}
	err := c.xaResource.Rollback(ctx, xaXid.String())
	if err == nil {
		if beforeRelease != nil {
			beforeRelease()
		}
		c.completePhaseTwoLocked()
	}
	return err
}

func (c *XAConn) completePhaseTwo() {
	c.completePhaseTwoWith(nil)
}

func (c *XAConn) completePhaseTwoWith(beforeRelease func()) {
	c.keeperMu.Lock()
	defer c.keeperMu.Unlock()
	if beforeRelease != nil {
		beforeRelease()
	}
	c.completePhaseTwoLocked()
}

func (c *XAConn) completePhaseTwoLocked() {
	c.releaseIfNecessaryLocked()
	c.preparedForPhaseTwo = false
	if c.poolDiscarded {
		if err := c.closePhysicalLocked(); err != nil {
			log.Errorf("close discarded XA connection after phase two: %v", err)
		}
	}
}

func (c *XAConn) closePhysicalLocked() error {
	if c.physicalClosed {
		return nil
	}
	c.physicalClosed = true
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}
