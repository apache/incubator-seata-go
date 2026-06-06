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
	"time"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
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

	tx                 driver.Tx
	xaResource         xa.XAResource
	xaErrorClassifier  xa.XAErrorClassifier
	xaBranchXid        *XABranchXid
	xaActive           bool
	rollBacked         bool
	branchRegisterTime time.Time
	prepareTime        time.Time
	isConnKept         bool
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

// ResetSession is called by database/sql before a pooled connection is reused.
func (c *XAConn) ResetSession(ctx context.Context) error {
	if c.shouldDetachFromPool() {
		return c.detachHeldConnection(ctx)
	}
	c.resetWrapperState()
	return c.Conn.ResetSession(ctx)
}

func (c *XAConn) detachHeldConnection(ctx context.Context) error {
	if c.Conn == nil || c.res == nil || c.res.connector == nil || c.xaBranchXid == nil {
		return driver.ErrBadConn
	}

	replacementConn, err := c.res.connector.Connect(ctx)
	if err != nil {
		return err
	}

	c.storeHeldConnectionCopy()
	c.Conn.targetConn = replacementConn
	c.resetWrapperState()
	return nil
}

func (c *XAConn) resetWrapperState() {
	c.tx = nil
	c.xaResource = nil
	c.xaErrorClassifier = nil
	c.xaBranchXid = nil
	c.xaActive = false
	c.rollBacked = false
	c.branchRegisterTime = time.Time{}
	c.prepareTime = time.Time{}
	c.isConnKept = false
	c.autoCommit = true
	c.txCtx = types.NewTxCtx()
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

	ret, err := c.createNewTxOnExecIfNeed(ctx, func() (types.ExecResult, error) {
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

	ret, err := c.createNewTxOnExecIfNeed(ctx, func() (types.ExecResult, error) {
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

	previousAutoCommit := c.autoCommit
	previousTxCtx := c.txCtx
	c.autoCommit = false

	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts
	c.txCtx.ResourceID = c.res.resourceID
	c.txCtx.XID = tm.GetXID(ctx)
	c.txCtx.TransactionMode = types.XAMode

	var tx driver.Tx
	var err error
	if c.dbType == types.DBTypeOracle {
		tx, err = c.Conn.BeginTx(ctx, opts)
		if err != nil {
			c.restoreXABeginState(previousAutoCommit, previousTxCtx)
			return nil, err
		}
		c.tx = tx
	} else {
		// Keep a sentinel target in Tx so any accidental fallback to the generic
		// driver.Tx path fails fast instead of silently masking XA lifecycle bugs.
		branchTx := xaBranchTx{}
		c.tx = branchTx

		tx, err = newTx(
			withDriverConn(c.Conn),
			withTxCtx(c.txCtx),
			withOriginTx(branchTx),
			withXAConn(c),
		)
		if err != nil {
			c.restoreXABeginState(previousAutoCommit, previousTxCtx)
			return nil, err
		}
	}

	if !c.autoCommit {
		if c.xaActive {
			return nil, errors.New("should NEVER happen: setAutoCommit from true to false while xa branch is active")
		}

		baseTx, ok := tx.(*Tx)
		if !ok {
			err := fmt.Errorf("start xa %s transaction failure for the tx is a wrong type", c.txCtx.XID)
			return nil, c.cleanXABeginFailure(tx, err, previousAutoCommit, previousTxCtx)
		}

		baseTx.xaConn = c

		c.branchRegisterTime = time.Now()
		if err := baseTx.register(c.txCtx); err != nil {
			err = fmt.Errorf("failed to register xa branch %s, err:%w", c.txCtx.XID, err)
			return nil, c.cleanXABeginFailure(tx, err, previousAutoCommit, previousTxCtx)
		}

		c.xaBranchXid = XaIdBuild(c.txCtx.XID, c.txCtx.BranchID)
		c.keepIfNecessary()

		if err = c.start(ctx); err != nil {
			err = fmt.Errorf("failed to start xa branch xid:%s err:%w", c.txCtx.XID, err)
			return nil, c.cleanXABeginFailure(tx, err, previousAutoCommit, previousTxCtx)
		}
		c.xaActive = true
	}

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

func (c *XAConn) createNewTxOnExecIfNeed(ctx context.Context, f func() (types.ExecResult, error)) (types.ExecResult, error) {
	var (
		tx  driver.Tx
		err error
	)

	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			log.Errorf("conn xa rollback recoverErr:%v", recoverErr)
			if tx != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Errorf("conn xa rollback error:%v", rollbackErr)
				}
				return
			}
			if c.tx != nil {
				if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
					log.Errorf("conn xa rollback error:%v", rollbackErr)
				}
			}
		}
	}()

	currentAutoCommit := c.autoCommit
	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		tx, err = c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			return nil, err
		}
	}

	// execute SQL
	ret, err := f()
	if err != nil {
		if tx != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Errorf("failed to rollback xa branch of :%s, err:%v", c.txCtx.XID, rollbackErr)
			}
		} else {
			if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
				log.Errorf("failed to rollback xa branch of :%s, err:%v", c.txCtx.XID, rollbackErr)
			}
		}
		return nil, err
	}

	if tx != nil && currentAutoCommit {
		// Commit through XATx so phase-one reporting stays coupled to driver.Tx lifecycle.
		if err = tx.Commit(); err != nil {
			log.Errorf("xa connection proxy commit failure xid:%s, err:%v", c.txCtx.XID, err)
			return nil, err
		}
	}

	return ret, nil
}

func (c *XAConn) keepIfNecessary() {
	if c.ShouldBeHeld() {
		if err := c.res.Hold(c.xaBranchXid.String(), c); err == nil {
			c.isConnKept = true
		}
	}
}

func (c *XAConn) releaseIfNecessary() {
	if c.xaBranchXid == nil {
		return
	}
	if c.ShouldBeHeld() && c.xaBranchXid.String() != "" {
		if c.isConnKept {
			c.res.Release(c.xaBranchXid.String())
			c.isConnKept = false
		}
	}
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
		c.xaResource.End(ctx, c.xaBranchXid.String(), c.rollbackEndFlag())
		c.XaRollback(ctx, c.xaBranchXid)
		return err
	}
	return err
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
	branchStatus, err := branchStatus(xaBranchXid)
	if err != nil {
		c.releaseIfNecessary()
		return fmt.Errorf("failed xa branch [%v] the global transaction has finish, branch status: [%v]", c.txCtx.XID, branchStatus)
	}
	return nil
}

func (c *XAConn) cleanXABranchContext() {
	h, _ := time.ParseDuration("-1000h")
	c.branchRegisterTime = time.Now().Add(h)
	c.prepareTime = time.Now().Add(h)
	c.xaActive = false
	if !c.isConnKept {
		c.xaBranchXid = nil
	}
}

func (c *XAConn) restoreXABeginState(autoCommit bool, txCtx *types.TransactionContext) {
	c.autoCommit = autoCommit
	c.txCtx = txCtx
	c.tx = nil
}

func (c *XAConn) cleanXABeginFailure(tx driver.Tx, cause error, autoCommit bool, txCtx *types.TransactionContext) error {
	defer c.restoreXABeginState(autoCommit, txCtx)

	c.releaseIfNecessary()
	c.cleanXABranchContext()

	baseTx, ok := tx.(*Tx)
	if !ok || baseTx.target == nil {
		return cause
	}
	if _, sentinel := baseTx.target.(xaBranchTx); sentinel {
		return cause
	}
	if err := tx.Rollback(); err != nil {
		return errors.Join(cause, fmt.Errorf("failed to rollback target transaction after XA begin failure: %w", err))
	}
	return cause
}

func (c *XAConn) Rollback(ctx context.Context) error {
	if c.autoCommit {
		return nil
	}

	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT rollback on an inactive session")
	}

	if !c.rollBacked {
		endFlag := c.rollbackEndFlag()
		if err := c.xaResource.End(ctx, c.xaBranchXid.String(), endFlag); err != nil {
			// Handle XAER_RMFAIL exception - check if it's already ended
			//expected error: Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the  IDLE state
			if c.xaErrorClassifier.IsAlreadyEnded(err) {
				// If already ended, continue with rollback
				log.Infof("XA branch already ended, continuing with rollback for xid: %s", c.txCtx.XID)
			} else {
				return c.rollbackEndErrorHandle(endFlag, err)
			}
		}

		// Then perform XA rollback
		if c.XaRollback(ctx, c.xaBranchXid) != nil {
			c.cleanXABranchContext()
			return c.rollbackErrorHandle(endFlag)
		}
		c.rollBacked = true
	}
	c.cleanXABranchContext()
	return nil
}

func (c *XAConn) rollbackEndFlag() int {
	if c.xaDBType() == types.DBTypeOracle {
		return xa.TMSuccess
	}
	return xa.TMFail
}

func (c *XAConn) rollbackErrorHandle(endFlag int) error {
	return fmt.Errorf("failed to end(%s) xa branch on [%v] - [%v]", xaEndFlagName(endFlag), c.txCtx.XID, c.xaBranchXid.GetBranchId())
}

func (c *XAConn) rollbackEndErrorHandle(endFlag int, endErr error) error {
	err := errors.Join(c.rollbackErrorHandle(endFlag), endErr)
	if c.xaDBType() != types.DBTypeOracle {
		return err
	}
	if targetErr := c.finishTargetTx(false); targetErr != nil {
		err = errors.Join(err, fmt.Errorf("failed to rollback target transaction after XA end failure: %w", targetErr))
	}
	c.releaseIfNecessary()
	c.cleanXABranchContext()
	return err
}

func xaEndFlagName(flag int) string {
	switch flag {
	case xa.TMSuccess:
		return "TMSUCCESS"
	case xa.TMFail:
		return "TMFAIL"
	case xa.TMSuspend:
		return "TMSUSPEND"
	default:
		return fmt.Sprintf("%d", flag)
	}
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
		if errors.Is(err, xa.ErrXAReadOnly) {
			c.finishCompletedTargetTx(true)
			setBranchStatus(c.xaBranchXid.String(), branch.BranchStatusPhasetwoCommitted)
			c.cleanXABranchContext()
			return nil
		}
		return c.commitErrorHandle(ctx, err)
	}

	c.prepareTime = time.Now()
	c.storeHeldConnectionCopy()
	return nil
}

func (c *XAConn) commitErrorHandle(ctx context.Context, cause error) error {
	if err := c.XaRollback(ctx, c.xaBranchXid); err != nil {
		cause = errors.Join(cause,
			fmt.Errorf("failed to rollback XA branch after commit-failure xid:%s, err:%w", c.txCtx.XID, err))
	}
	c.cleanXABranchContext()
	return cause
}

func (c *XAConn) ShouldBeHeld() bool {
	return c.res != nil && c.res.IsShouldBeHeld()
}

func (c *XAConn) shouldDetachFromPool() bool {
	return c.isConnKept && c.ShouldBeHeld() && c.xaBranchXid != nil && !c.prepareTime.IsZero()
}

func (c *XAConn) checkTimeout(ctx context.Context, now time.Time) error {
	if now.Sub(c.branchRegisterTime) > xaConnTimeout {
		c.XaRollback(ctx, c.xaBranchXid)
		return fmt.Errorf("XA branch timeout error xid:%s", c.txCtx.XID)
	}
	return nil
}

func (c *XAConn) Close() error {
	c.rollBacked = false
	if c.isConnKept && c.ShouldBeHeld() {
		return nil
	}
	c.cleanXABranchContext()
	if err := c.Conn.Close(); err != nil {
		return err
	}
	return nil
}

func (c *XAConn) CloseForce() error {
	if err := c.Conn.Close(); err != nil {
		return err
	}
	c.rollBacked = false
	c.cleanXABranchContext()
	c.releaseIfNecessary()
	return nil
}

func (c *XAConn) XaCommit(ctx context.Context, xaXid XAXid) error {
	err := c.xaResource.Commit(ctx, xaXid.String(), false)
	if c.isBranchCommitted(err) {
		c.finishCompletedTargetTx(true)
	} else if c.isBranchRollbacked(err) {
		c.finishCompletedTargetTx(false)
	}
	return err
}

func (c *XAConn) XaRollbackByBranchId(ctx context.Context, xaXid XAXid) error {
	return c.XaRollback(ctx, xaXid)
}

func (c *XAConn) XaRollback(ctx context.Context, xaXid XAXid) error {
	err := c.xaResource.Rollback(ctx, xaXid.String())
	if c.isBranchRollbacked(err) {
		c.finishCompletedTargetTx(false)
	} else if c.isBranchCommitted(err) {
		c.finishCompletedTargetTx(true)
	}
	return err
}

func (c *XAConn) isBranchCommitted(err error) bool {
	if err == nil {
		return true
	}
	if classifier, ok := c.xaErrorClassifier.(xa.XACommitErrorClassifier); ok {
		return classifier.IsAlreadyCommitted(err)
	}
	return c.isBranchFinished(err)
}

func (c *XAConn) isBranchRollbacked(err error) bool {
	if err == nil {
		return true
	}
	if classifier, ok := c.xaErrorClassifier.(xa.XARollbackErrorClassifier); ok {
		return classifier.IsAlreadyRollbacked(err)
	}
	return c.isBranchFinished(err)
}

func (c *XAConn) isBranchFinished(err error) bool {
	return err == nil || (c.xaErrorClassifier != nil && c.xaErrorClassifier.IsAlreadyEnded(err))
}

func (c *XAConn) finishTargetTx(commit bool) error {
	if c.xaDBType() != types.DBTypeOracle || c.tx == nil {
		return nil
	}

	tx, ok := c.tx.(*Tx)
	if !ok || tx.target == nil {
		return nil
	}

	var err error
	if commit {
		err = tx.target.Commit()
	} else {
		err = tx.target.Rollback()
	}
	c.tx = nil
	c.autoCommit = true
	return err
}

func (c *XAConn) finishCompletedTargetTx(commit bool) {
	if err := c.finishTargetTx(commit); err != nil {
		log.Errorf("finish target transaction after xa branch completed failed, xid:%v, err:%v", c.xaBranchXid, err)
	}
	c.releaseIfNecessary()
}

func (c *XAConn) storeHeldConnectionCopy() {
	if !c.isConnKept || c.Conn == nil || c.res == nil || c.xaBranchXid == nil {
		return
	}

	heldBaseConn := *c.Conn
	heldXAConn := *c
	heldXAConn.Conn = &heldBaseConn
	c.res.keeper.Store(c.xaBranchXid.String(), &heldXAConn)
}

func (c *XAConn) xaDBType() types.DBType {
	if c.Conn == nil {
		return types.DBTypeUnknown
	}
	if c.Conn.dbType != 0 && c.Conn.dbType != types.DBTypeUnknown {
		return c.Conn.dbType
	}
	if c.Conn.res != nil {
		return c.Conn.res.GetDbType()
	}
	return types.DBTypeUnknown
}
