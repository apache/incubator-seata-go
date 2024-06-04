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

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
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

	currentAutoCommit := c.autoCommit
	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		tx, err = c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			return nil, err
		}
	}
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

	// execute SQL
	ret, err := f()
	if err != nil {
		// XA End & Rollback
		if rollbackErr := c.Rollback(ctx); rollbackErr != nil {
			log.Errorf("failed to rollback xa branch of :%s, err:%w", c.txCtx.XID, rollbackErr)
		}
		return nil, err
	}

	if tx != nil && currentAutoCommit {
		if err := c.Commit(ctx); err != nil {
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
	xaResource, err := xa.CreateXAResource(c.Conn.targetConn, c.dbType)
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
		if c.xaResource.End(ctx, c.xaBranchXid.String(), xa.TMFail) != nil {
			return c.rollbackErrorHandle()
		}
		if c.XaRollback(ctx, c.xaBranchXid) != nil {
			c.cleanXABranchContext()
			return c.rollbackErrorHandle()
		}
		if err := c.tx.Rollback(); err != nil {
			c.cleanXABranchContext()
			return fmt.Errorf("failed to report XA branch commit-failure on xid:%s err:%w", c.txCtx.XID, err)
		}
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
	if c.end(ctx, xa.TMSuccess) != nil {
		return c.commitErrorHandle(ctx)
	}

	if c.checkTimeout(ctx, now) != nil {
		return c.commitErrorHandle(ctx)
	}

	if c.xaResource.XAPrepare(ctx, c.xaBranchXid.String()) != nil {
		return c.commitErrorHandle(ctx)
	}
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
