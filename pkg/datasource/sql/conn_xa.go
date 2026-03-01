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
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

var xaConnTimeout time.Duration

// XAConn Database connection proxy object under XA transaction model
// Conn is assumed to be stateful.
type XAConn struct {
	*Conn

	tx                 driver.Tx
	xaResource         xa.XAResource
	xaBranchXid        *XABranchXid
	xaActive           bool
	rollBacked         bool
	branchRegisterTime time.Time
	prepareTime        time.Time
	isConnKept         bool
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
	xid := tm.GetXID(ctx)
	log.Infof("[XA-DEBUG] QueryContext called, xid: %s, query: %s, autoCommit: %v", xid, query, c.autoCommit)

	if c.createOnceTxContext(ctx) {
		log.Infof("[XA-DEBUG] createOnceTxContext returned true, registered defer to reset txCtx")
		defer func() {
			log.Infof("[XA-DEBUG] QueryContext defer: resetting txCtx")
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
		log.Errorf("[XA-DEBUG] QueryContext failed: %v", err)
		return nil, err
	}
	log.Infof("[XA-DEBUG] QueryContext succeeded")
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
	xid := tm.GetXID(ctx)
	log.Infof("[XA-DEBUG] BeginTx called, xid: %s, IsGlobalTx: %v, autoCommit: %v", xid, tm.IsGlobalTx(ctx), c.autoCommit)

	if !tm.IsGlobalTx(ctx) {
		tx, err := c.Conn.BeginTx(ctx, opts)
		return tx, err
	}

	// Save the original autoCommit state before modifying it
	wasAutoCommit := c.autoCommit
	c.autoCommit = false

	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts
	c.txCtx.ResourceID = c.res.resourceID
	c.txCtx.XID = tm.GetXID(ctx)
	c.txCtx.TransactionMode = types.XAMode
	// Store the original autoCommit state for later use in commit logic
	c.txCtx.GlobalLockRequire = wasAutoCommit

	log.Infof("[XA-DEBUG] BeginTx: txCtx created, XID: %s, GlobalLockRequire: %v (wasAutoCommit: %v)",
		c.txCtx.XID, c.txCtx.GlobalLockRequire, wasAutoCommit)

	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		log.Errorf("[XA-DEBUG] Conn.BeginTx failed: %v", err)
		return nil, err
	}
	c.tx = tx

	if !c.autoCommit {
		if c.xaActive {
			return nil, errors.New("should NEVER happen: setAutoCommit from true to false while xa branch is active")
		}

		baseTx, ok := tx.(*Tx)
		if !ok {
			return nil, fmt.Errorf("start xa %s transaction failure for the tx is a wrong type", c.txCtx.XID)
		}

		c.branchRegisterTime = time.Now()
		if err := baseTx.register(c.txCtx); err != nil {
			c.cleanXABranchContext()
			return nil, fmt.Errorf("failed to register xa branch %s, err:%w", c.txCtx.XID, err)
		}

		c.xaBranchXid = XaIdBuild(c.txCtx.XID, c.txCtx.BranchID)
		c.keepIfNecessary()

		if err = c.start(ctx); err != nil {
			c.cleanXABranchContext()
			return nil, fmt.Errorf("failed to start xa branch xid:%s err:%w", c.txCtx.XID, err)
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

	xid := tm.GetXID(ctx)
	log.Infof("[XA-DEBUG] createNewTxOnExecIfNeed called, xid: %s, autoCommit: %v, txCtx.TransactionMode: %v, txCtx.XID: %s",
		xid, c.autoCommit, c.txCtx.TransactionMode, c.txCtx.XID)

	defer func() {
		recoverErr := recover()
		// Check if error is ErrSkip - don't rollback for this special error
		isErrSkip := err != nil && (err == driver.ErrSkip || err.Error() == "driver: skip fast-path; continue as if unimplemented")

		if (err != nil && !isErrSkip) || recoverErr != nil {
			log.Errorf("[XA-DEBUG] defer triggered, err: %v, recoverErr: %v", err, recoverErr)
			// Don't try to rollback if tx is nil
			if c.tx != nil {
				log.Infof("[XA-DEBUG] calling c.tx.Rollback() in defer")
				rollbackErr := c.tx.Rollback()
				if rollbackErr != nil {
					log.Errorf("conn at rollback error:%v", rollbackErr)
				}
			}
		} else {
			log.Infof("[XA-DEBUG] defer: no error or ErrSkip, skipping rollback")
		}
	}()

	currentAutoCommit := c.autoCommit
	registry := getXARegistry()

	// For global transactions in autoCommit mode, create/reuse XA branch
	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		// Check if we already have an active XA branch for this transaction
		if registry.canReuse(xid) {
			// Reuse existing XA branch - execute SQL directly
			log.Infof("Reusing existing XA branch for xid: %s", xid)
			ret, err := f()
			if err != nil {
				// On error (but not ErrSkip), rollback the entire branch
				isErrSkip := err == driver.ErrSkip || err.Error() == "driver: skip fast-path; continue as if unimplemented"
				if !isErrSkip && c.xaActive {
					if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
						log.Errorf("failed to rollback xa branch of :%s, err:%v", c.txCtx.XID, rollbackErr)
					}
				}
				return nil, err
			}
			return ret, nil
		}

		// Create new XA branch
		log.Infof("[XA-DEBUG] creating new XA branch with BeginTx")
		tx, err = c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			log.Errorf("[XA-DEBUG] BeginTx failed: %v", err)
			return nil, err
		}

		log.Infof("[XA-DEBUG] BeginTx succeeded, xaActive: %v, xaBranchXid: %v", c.xaActive, c.xaBranchXid)

		// Register the XA branch in the registry for reuse by subsequent SQL statements
		if c.xaBranchXid != nil {
			registry.register(xid, c.xaBranchXid.String(), c.res.resourceID, c)
		}
	} else {
		log.Infof("[XA-DEBUG] skipping XA branch creation, conditions: TransactionMode=%v, IsGlobalTx=%v, autoCommit=%v",
			c.txCtx.TransactionMode, tm.IsGlobalTx(ctx), c.autoCommit)
	}

	// execute SQL
	log.Infof("[XA-DEBUG] executing SQL")
	ret, err := f()
	log.Infof("[XA-DEBUG] SQL executed, err: %v", err)
	if err != nil {
		// Check if this is driver.ErrSkip - not a real error, just means use fallback path
		// In this case, don't rollback the XA branch, just return the error
		// The database/sql package will handle the retry
		isErrSkip := err == driver.ErrSkip || err.Error() == "driver: skip fast-path; continue as if unimplemented"
		if isErrSkip {
			log.Infof("[XA-DEBUG] Got ErrSkip, returning without rollback - database/sql will use fallback")
			return nil, err
		}
		// On real error, rollback the entire branch
		if c.xaActive {
			if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
				log.Errorf("failed to rollback xa branch of :%s, err:%v", c.txCtx.XID, rollbackErr)
			}
		}
		return nil, err
	}

	// For autoCommit mode with global transaction, call Commit()
	// The Commit method will skip actual XA END/PREPARE if GlobalLockRequire is true
	// This allows multiple SQL statements to use the same XA branch
	if tx != nil && currentAutoCommit {
		log.Infof("[XA-DEBUG] calling Commit, tx != nil: %v, currentAutoCommit: %v", tx != nil, currentAutoCommit)
		if err = c.Commit(ctx); err != nil {
			log.Errorf("xa connection proxy commit failure xid:%s, err:%v", c.txCtx.XID, err)
			// XA End & Rollback
			if err := c.Rollback(ctx); err != nil {
				log.Errorf("xa connection proxy rollback failure xid:%s, err:%v", c.txCtx.XID, err)
			}
			return nil, err
		}
		log.Infof("[XA-DEBUG] Commit succeeded")
	} else {
		log.Infof("[XA-DEBUG] skipping Commit call, tx != nil: %v, currentAutoCommit: %v", tx != nil, currentAutoCommit)
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

	if err := c.xaResource.Start(ctx, c.xaBranchXid.String(), xa.TMNoFlags); err != nil {
		return fmt.Errorf("xa xid %s resource connection start err:%w", c.txCtx.XID, err)
	}

	// For multi-statement XA transactions (originally in autoCommit mode),
	// skip the termination check. The check will be done during Phase 2 commit/rollback.
	if c.txCtx.GlobalLockRequire {
		log.Infof("Skipping termination check for autoCommit-based XA transaction, xid: %s", c.txCtx.XID)
		return nil
	}

	// For explicit transactions (BeginTx mode), do the normal termination check
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

func (c *XAConn) Rollback(ctx context.Context) error {
	log.Infof("[XA-DEBUG] Rollback called, txCtx.XID: %s, GlobalLockRequire: %v, autoCommit: %v, xaActive: %v, rollBacked: %v",
		c.txCtx.XID, c.txCtx.GlobalLockRequire, c.autoCommit, c.xaActive, c.rollBacked)

	// For autoCommit mode (multi-statement transactions), check if rollback is needed
	if c.autoCommit && !c.xaActive {
		log.Infof("[XA-DEBUG] Rollback skipped: autoCommit=true and xaActive=false")
		return nil
	}

	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT rollback on an inactive session")
	}

	if !c.rollBacked {
		// First end the XA branch with TMFail
		if err := c.xaResource.End(ctx, c.xaBranchXid.String(), xa.TMFail); err != nil {
			// Handle XAER_RMFAIL exception - check if it's already ended
			//expected error: Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the  IDLE state
			if isXAER_RMFAILAlreadyEnded(err) {
				// If already ended, continue with rollback
				log.Infof("XA branch already ended, continuing with rollback for xid: %s", c.txCtx.XID)
			} else {
				return c.rollbackErrorHandle()
			}
		}

		// Then perform XA rollback
		if c.XaRollback(ctx, c.xaBranchXid) != nil {
			c.cleanXABranchContext()
			return c.rollbackErrorHandle()
		}
		if err := c.tx.Rollback(); err != nil {
			c.cleanXABranchContext()
			return fmt.Errorf("failed to report XA branch commit-failure on xid:%s err:%w", c.txCtx.XID, err)
		}
		c.rollBacked = true
	}
	c.cleanXABranchContext()

	// Clean up registry on rollback
	registry := getXARegistry()
	registry.unregister(c.txCtx.XID)

	return nil
}

func (c *XAConn) rollbackErrorHandle() error {
	return fmt.Errorf("failed to end(TMFAIL) xa branch on [%v] - [%v]", c.txCtx.XID, c.xaBranchXid.GetBranchId())
}

func (c *XAConn) Commit(ctx context.Context) error {
	log.Infof("[XA-DEBUG] Commit called, txCtx.XID: %s, GlobalLockRequire: %v, autoCommit: %v, xaActive: %v",
		c.txCtx.XID, c.txCtx.GlobalLockRequire, c.autoCommit, c.xaActive)

	// If this XA branch was created in autoCommit mode (multi-statement transaction),
	// don't do the actual XA commit here. The TC will handle it in Phase 2.
	// We detect this by checking GlobalLockRequire which we set to wasAutoCommit in BeginTx
	if c.txCtx.GlobalLockRequire {
		log.Infof("Skipping XA commit for autoCommit mode, will be committed by TC in Phase 2, xid: %s", c.txCtx.XID)
		return nil
	}

	if c.autoCommit {
		return nil
	}

	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT commit on an inactive session")
	}

	now := time.Now()

	// Update registry state to ENDED before XA END
	registry := getXARegistry()
	registry.setState(c.txCtx.XID, xaStateEnded)

	if c.end(ctx, xa.TMSuccess) != nil {
		return c.commitErrorHandle(ctx)
	}

	if c.checkTimeout(ctx, now) != nil {
		return c.commitErrorHandle(ctx)
	}

	if c.xaResource.XAPrepare(ctx, c.xaBranchXid.String()) != nil {
		return c.commitErrorHandle(ctx)
	}

	// Update registry state to PREPARED and unregister
	registry.setState(c.txCtx.XID, xaStatePrepared)
	registry.unregister(c.txCtx.XID)

	return nil
}

func (c *XAConn) commitErrorHandle(ctx context.Context) error {
	var err error
	if err = c.XaRollback(ctx, c.xaBranchXid); err != nil {
		err = fmt.Errorf("failed to report XA branch commit-failure xid:%s, err:%w", c.txCtx.XID, err)
	}
	c.cleanXABranchContext()
	return err
}

func (c *XAConn) ShouldBeHeld() bool {
	return c.res.IsShouldBeHeld() || (c.res.GetDbType().String() != "" && c.res.GetDbType() != types.DBTypeUnknown)
}

func (c *XAConn) checkTimeout(ctx context.Context, now time.Time) error {
	if now.Sub(c.branchRegisterTime) > xaConnTimeout {
		c.XaRollback(ctx, c.xaBranchXid)
		return fmt.Errorf("XA branch timeout error xid:%s", c.txCtx.XID)
	}
	return nil
}

func (c *XAConn) Close() error {
	// Get XID safely - txCtx might be nil in some cases
	xid := ""
	if c.txCtx != nil {
		xid = c.txCtx.XID
	}
	log.Infof("[XA-DEBUG] Close called, txCtx.XID: %s, xaActive: %v, isConnKept: %v, rollBacked: %v",
		xid, c.xaActive, c.isConnKept, c.rollBacked)
	c.rollBacked = false
	if c.isConnKept && c.ShouldBeHeld() {
		log.Infof("[XA-DEBUG] Close skipped: connection is kept")
		return nil
	}
	c.cleanXABranchContext()
	// Check if Conn is nil before calling Close
	if c.Conn == nil {
		log.Infof("[XA-DEBUG] Conn is nil, skipping Conn.Close()")
		return nil
	}
	if err := c.Conn.Close(); err != nil {
		log.Errorf("[XA-DEBUG] Conn.Close failed: %v", err)
		return err
	}
	log.Infof("[XA-DEBUG] Close succeeded")
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
	c.releaseIfNecessary()
	return err
}

func (c *XAConn) XaRollbackByBranchId(ctx context.Context, xaXid XAXid) error {
	return c.XaRollback(ctx, xaXid)
}

func (c *XAConn) XaRollback(ctx context.Context, xaXid XAXid) error {
	err := c.xaResource.Rollback(ctx, xaXid.String())
	c.releaseIfNecessary()
	return err
}

// isXAER_RMFAILAlreadyEnded checks if the XAER_RMFAIL error indicates the XA branch is already ended
// expected error: Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the IDLE state
func isXAER_RMFAILAlreadyEnded(err error) bool {
	if err == nil {
		return false
	}
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if mysqlErr.Number == types.ErrCodeXAER_RMFAIL_IDLE {
			return strings.Contains(mysqlErr.Message, "IDLE state") || strings.Contains(mysqlErr.Message, "already ended")
		}
	}
	// TODO: handle other DB errors

	return false
}
