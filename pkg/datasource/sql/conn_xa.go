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
	"seata.apache.org/seata-go/pkg/util/log"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/pkg/tm"
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
		return c.Conn.BeginTx(ctx, opts)
	}

	c.autoCommit = false

	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts
	c.txCtx.ResourceID = c.res.resourceID
	c.txCtx.XID = tm.GetXID(ctx)
	c.txCtx.TransactionMode = types.XAMode

	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	c.tx = tx

	if !c.autoCommit {
		if c.xaActive {
			return nil, errors.New("should NEVER happen: setAutoCommit from true to false while xa branch is active")
		}

		baseTx, ok := tx.(*Tx)
		if !ok {
			return nil, fmt.Errorf("start xa %s transaction failure: tx type invalid", c.txCtx.XID)
		}

		if err := baseTx.register(c.txCtx); err != nil {
			c.cleanXABranchContext()
			return nil, fmt.Errorf("failed to register xa branch %s: %w", c.txCtx.XID, err)
		}

		c.xaBranchXid = NewXABranchXid(
			WithXid(c.txCtx.XID),
			WithBranchId(uint64(c.txCtx.BranchID)),
			WithDatabaseType(c.txCtx.DBType),
		)

		c.keepIfNecessary()

		if err = c.start(ctx); err != nil {
			c.cleanXABranchContext()
			return nil, err
		}
		c.xaActive = true
		c.branchRegisterTime = time.Now()
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
		if err != nil || recoverErr != nil {
			log.Errorf("conn at rollback  error:%v or recoverErr:%v", err, recoverErr)
			if c.tx != nil {
				rollbackErr := c.tx.Rollback()
				if rollbackErr != nil {
					log.Errorf("conn at rollback error:%v", rollbackErr)
				}
			}
		}
	}()

	currentAutoCommit := c.autoCommit

	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		tx, err = c.BeginTx(ctx, driver.TxOptions{
			Isolation: driver.IsolationLevel(gosql.LevelDefault),
		})
		if err != nil {
			return nil, err
		}
	}

	// execute SQL
	ret, err := f()
	if err != nil {
		// XA End & Rollback
		if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
			log.Errorf("failed to rollback xa branch of :%s, err:%v", c.txCtx.XID, rollbackErr)
		}
		return nil, err
	}

	if tx != nil && currentAutoCommit {
		if err = c.Commit(ctx); err != nil {
			log.Errorf("xa connection proxy commit failure xid:%s, err:%v", c.txCtx.XID, err)
			// XA End & Rollback
			if err := c.Rollback(ctx); err != nil {
				log.Errorf("xa connection proxy rollback failure xid:%s, err:%v", c.txCtx.XID, err)
			}
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
	if c.ShouldBeHeld() && c.xaBranchXid.String() != "" {
		if c.isConnKept {
			c.res.Release(c.xaBranchXid.String())
			c.isConnKept = false
		}
	}
}

func (c *XAConn) start(ctx context.Context) error {
	// Extract the underlying transaction from the wrapped Tx
	var targetTx driver.Tx
	if baseTx, ok := c.tx.(*Tx); ok && baseTx != nil {
		targetTx = baseTx.GetTarget()
	} else {
		targetTx = c.tx
	}

	xaResource, err := xa.CreateXAResource(c.Conn.targetConn, c.dbType, targetTx)
	if err != nil {
		return fmt.Errorf("create xa xid:%s resoruce err:%w", c.txCtx.XID, err)
	}
	c.xaResource = xaResource

	if err := c.xaResource.Start(ctx, c.xaBranchXid.String(), xa.TMNoFlags); err != nil {
		return fmt.Errorf("xa xid %s resource connection start err:%w", c.txCtx.XID, err)
	}

	if err := c.termination(c.xaBranchXid.String()); err != nil {
		c.xaResource.End(ctx, c.xaBranchXid.String(), xa.TMFail)
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

	if c.end(ctx, xa.TMSuccess) != nil {
		return c.commitErrorHandle(ctx)
	}

	// Check timeout BEFORE prepare, not after
	if c.checkTimeout(ctx, time.Now()) != nil {
		return c.commitErrorHandle(ctx)
	}

	if c.xaResource.XAPrepare(ctx, c.xaBranchXid.String()) != nil {
		return c.commitErrorHandle(ctx)
	}

	// Record prepare time for phase 2 timeout checking
	c.prepareTime = time.Now()

	// Reset connection state after XA PREPARE to allow connection reuse
	// The XA branch is now in PREPARED state and will be committed/rolled back
	// in phase 2 by the global coordinator
	c.autoCommit = true
	c.xaActive = false

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
	if xaConnTimeout > 0 && now.Sub(c.branchRegisterTime) > xaConnTimeout {
		log.Warnf("XA branch timeout detected for xid: %s, attempting rollback", c.txCtx.XID)
		if c.xaResource != nil && c.xaBranchXid != nil {
			if err := c.XaRollback(ctx, c.xaBranchXid); err != nil {
				log.Errorf("failed to rollback timed out XA branch xid: %s, err: %v", c.txCtx.XID, err)
			}
		}
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
	if c.xaResource == nil {
		log.Errorf("xaResource is nil, cannot commit xid: %s", xaXid.String())
		return fmt.Errorf("xaResource is nil for xid: %s", xaXid.String())
	}
	err := c.xaResource.Commit(ctx, xaXid.String(), false)
	c.releaseIfNecessary()
	return err
}

func (c *XAConn) XaRollbackByBranchId(ctx context.Context, xaXid XAXid) error {
	return c.XaRollback(ctx, xaXid)
}

func (c *XAConn) XaRollback(ctx context.Context, xaXid XAXid) error {
	if c.xaResource == nil {
		log.Errorf("xaResource is nil, cannot rollback xid: %s", xaXid.String())
		return fmt.Errorf("xaResource is nil for xid: %s", xaXid.String())
	}
	err := c.xaResource.Rollback(ctx, xaXid.String())
	c.releaseIfNecessary()
	return err
}

func isXAER_RMFAILAlreadyEnded(err error) bool {
	if err == nil {
		return false
	}

	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if mysqlErr.Number == types.ErrCodeXAER_RMFAIL_IDLE {
			return strings.Contains(mysqlErr.Message, "IDLE state") || strings.Contains(mysqlErr.Message, "already ended")
		}
		// For MySQL errors, only trust the specific error code
		return false
	}

	if pgErr, ok := err.(*pq.Error); ok {
		return isPostgreSQLXAAlreadyEnded(pgErr)
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "idle") ||
		strings.Contains(errMsg, "already ended") ||
		strings.Contains(errMsg, "no active") ||
		strings.Contains(errMsg, "invalid transaction state")
}

func isPostgreSQLXAAlreadyEnded(pgErr *pq.Error) bool {
	if pgErr == nil {
		return false
	}

	switch pgErr.Code {
	case types.PostgreSQLErrCodeNoActiveSQLTx:
		return true
	case types.PostgreSQLErrCodeIdleInTx:
		return true
	case types.PostgreSQLErrCodeFailedSQLTx:
		return strings.Contains(strings.ToLower(pgErr.Message), "already") ||
			strings.Contains(strings.ToLower(pgErr.Message), "ended")
	}

	if pgErr.Code.Class() == types.PostgreSQLErrClassInvalidTxState {
		message := strings.ToLower(pgErr.Message)
		return strings.Contains(message, "idle") ||
			strings.Contains(message, "already ended") ||
			strings.Contains(message, "no active")
	}

	return false
}
