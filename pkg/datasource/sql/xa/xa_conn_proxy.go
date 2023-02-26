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
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/xa/xaresource"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
)

type ConnectionProxyXA struct {
	xaConn     *sql.XAConn
	resource   *sql.DBResource
	xid        string
	xaResource xaresource.XAResource

	xaBranchXid        *XABranchXid
	xaActive           bool
	rollBacked         bool
	branchRegisterTime int64
	prepareTime        time.Time
	timeout            int

	currentAutoCommitStatus bool
	isConnKept              bool
}

func NewConnectionProxyXA(originalConnection driver.Conn, resource *sql.DBResource, xid string) (*ConnectionProxyXA, error) {
	xaConn, isXAConn := originalConnection.(*sql.XAConn)
	if !isXAConn {
		return nil, fmt.Errorf("new xa connection proxy failure because originalConnection is not sql.XAConn, xid:%s", xid)
	}

	connectionProxyXA := &ConnectionProxyXA{
		xaConn:   xaConn,
		resource: resource,
		xid:      xid,
	}

	connectionProxyXA.currentAutoCommitStatus = connectionProxyXA.xaConn.GetAutoCommit()
	if !connectionProxyXA.currentAutoCommitStatus {
		return nil, fmt.Errorf("connection[autocommit=false] as default is NOT supported")
	}

	return connectionProxyXA, nil
}

func (c *ConnectionProxyXA) SetXAResource(resource xaresource.XAResource) {
	c.xaResource = resource
}

func (c *ConnectionProxyXA) keepIfNecessary() {
	if c.ShouldBeHeld() {
		if err := c.resource.Hold(c.xaBranchXid.String(), c); err == nil {
			c.isConnKept = true
		}
	}
}

func (c *ConnectionProxyXA) releaseIfNecessary() {
	if c.ShouldBeHeld() && c.xaBranchXid.String() != "" {
		if c.isConnKept {
			c.resource.Release(c.xaBranchXid.String())
			c.isConnKept = false
		}
	}
}

func (c *ConnectionProxyXA) XaCommit(ctx context.Context, xid string, branchId int64) error {
	xaXid := XaIdBuild(xid, branchId)
	err := c.xaResource.Commit(ctx, xaXid.String(), false)
	c.releaseIfNecessary()
	return err
}

func (c *ConnectionProxyXA) XaRollbackByBranchId(ctx context.Context, xid string, branchId int64) error {
	xaXid := XaIdBuild(xid, branchId)
	return c.XaRollback(ctx, xaXid)
}

func (c *ConnectionProxyXA) XaRollback(ctx context.Context, xaXid XAXid) error {
	err := c.xaResource.Rollback(ctx, xaXid.GetGlobalXid())
	c.releaseIfNecessary()
	return err
}

func (c *ConnectionProxyXA) SetAutoCommit(ctx context.Context, autoCommit bool) error {
	if c.currentAutoCommitStatus == autoCommit {
		return nil
	}
	if autoCommit {
		if c.xaActive {
			_ = c.Commit(ctx)
		}
	} else {
		if c.xaActive {
			return fmt.Errorf("should NEVER happen: setAutoCommit from true to false while xa branch is active")
		}

		c.branchRegisterTime = time.Now().UnixMilli()
		var branchRegisterParam rm.BranchRegisterParam
		branchRegisterParam.BranchType = branch.BranchTypeXA
		branchRegisterParam.ResourceId = c.resource.GetResourceId()
		branchRegisterParam.Xid = c.xid
		branchId, err := datasource.GetDataSourceManager(branch.BranchTypeXA).BranchRegister(context.TODO(), branchRegisterParam)
		if err != nil {
			c.cleanXABranchContext()
			return fmt.Errorf("failed to register xa branch [%v]", c.xid)
		}
		c.xaBranchXid = XaIdBuild(c.xid, branchId)
		c.keepIfNecessary()
		err = c.start(ctx)
		if err != nil {
			c.cleanXABranchContext()
			return fmt.Errorf("failed to start xa branch [%v]", c.xid)
		}
		c.xaActive = true
	}
	c.currentAutoCommitStatus = autoCommit
	return nil
}

func (c *ConnectionProxyXA) GetAutoCommit() bool {
	return c.currentAutoCommitStatus
}

func (c *ConnectionProxyXA) Commit(ctx context.Context) error {
	if c.currentAutoCommitStatus {
		return nil
	}
	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT commit on an inactive session")
	}
	now := time.Now().UnixMilli()
	if c.end(ctx, xaresource.TMSuccess) != nil {
		return c.commitErrorHandle()
	}
	if c.checkTimeout(ctx, now) != nil {
		return c.commitErrorHandle()
	}
	if c.xaResource.XAPrepare(ctx, c.xaBranchXid.String()) != nil {
		return c.commitErrorHandle()
	}
	return nil
}

func (c *ConnectionProxyXA) commitErrorHandle() error {
	req := message.BranchReportRequest{
		BranchType:      branch.BranchTypeXA,
		Xid:             c.xid,
		BranchId:        c.xaBranchXid.GetBranchId(),
		Status:          branch.BranchStatusPhaseoneFailed,
		ApplicationData: nil,
		ResourceId:      c.resource.GetResourceId(),
	}
	if datasource.NewBasicSourceManager().BranchReport(context.TODO(), req) != nil {
		c.cleanXABranchContext()
		return fmt.Errorf("failed to report XA branch commit-failure on [%v] - [%v]", c.xid, c.xaBranchXid.GetBranchId())
	}
	c.cleanXABranchContext()
	return fmt.Errorf("failed to end(TMSUCCESS)/prepare xa branch on [%v] - [%v]", c.xid, c.xaBranchXid.GetBranchId())
}

func (c *ConnectionProxyXA) Rollback(ctx context.Context) error {
	if c.currentAutoCommitStatus {
		return nil
	}
	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT rollback on an inactive session")
	}
	if !c.rollBacked {
		if c.xaResource.End(ctx, c.xaBranchXid.String(), xaresource.TMFail) != nil {
			return c.rollbackErrorHandle()
		}
		if c.XaRollback(ctx, c.xaBranchXid) != nil {
			c.cleanXABranchContext()
			return c.rollbackErrorHandle()
		}
		req := message.BranchReportRequest{
			BranchType:      branch.BranchTypeXA,
			Xid:             c.xid,
			BranchId:        c.xaBranchXid.GetBranchId(),
			Status:          branch.BranchStatusPhaseoneFailed,
			ApplicationData: nil,
			ResourceId:      c.resource.GetResourceId(),
		}
		if datasource.NewBasicSourceManager().BranchReport(context.TODO(), req) != nil {
			c.cleanXABranchContext()
			return fmt.Errorf("failed to report XA branch commit-failure on [%v] - [%v]", c.xid, c.xaBranchXid.GetBranchId())
		}
	}
	c.cleanXABranchContext()
	return nil
}

func (c *ConnectionProxyXA) rollbackErrorHandle() error {
	return fmt.Errorf("failed to end(TMFAIL) xa branch on [%v] - [%v]", c.xid, c.xaBranchXid.GetBranchId())
}

func (c *ConnectionProxyXA) start(ctx context.Context) error {
	err := c.xaResource.Start(ctx, c.xaBranchXid.String(), xaresource.TMNoFlags)
	if err := c.termination(c.xaBranchXid.String()); err != nil {
		c.xaResource.End(ctx, c.xaBranchXid.String(), xaresource.TMFail)
		c.XaRollback(ctx, c.xaBranchXid)
		return err
	}
	return err
}

func (c *ConnectionProxyXA) end(ctx context.Context, flags int) error {
	err := c.termination(c.xaBranchXid.String())
	if err != nil {
		return err
	}
	err = c.xaResource.End(ctx, c.xaBranchXid.String(), flags)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConnectionProxyXA) cleanXABranchContext() {
	c.branchRegisterTime = 0
	c.prepareTime = 0
	c.timeout = 0
	c.xaActive = false
	if !c.isConnKept {
		c.xaBranchXid = nil
	}
}

func (c *ConnectionProxyXA) checkTimeout(ctx context.Context, now int64) error {
	if now-c.branchRegisterTime > int64(c.timeout) {
		c.XaRollback(ctx, c.xaBranchXid)
		return fmt.Errorf("XA branch timeout error")
	}
	return nil
}

func (c *ConnectionProxyXA) Close() error {
	c.rollBacked = false
	if c.isConnKept && c.ShouldBeHeld() {
		return nil
	}
	c.cleanXABranchContext()
	if err := c.xaConn.Close(); err != nil {
		return err
	}
	return nil
}

func (c *ConnectionProxyXA) CloseForce() error {
	physicalConn := c.xaConn
	if err := physicalConn.Close(); err != nil {
		return err
	}
	c.rollBacked = false
	c.cleanXABranchContext()
	if err := c.xaConn.Close(); err != nil {
		return err
	}
	c.releaseIfNecessary()
	return nil
}

func (c *ConnectionProxyXA) ShouldBeHeld() bool {
	return c.resource.IsShouldBeHeld() || (c.resource.GetDbType().String() != "" && c.resource.GetDbType() != types.DBTypeUnknown)
}

func (c *ConnectionProxyXA) GetPrepareTime() time.Time {
	return c.prepareTime
}

func (c *ConnectionProxyXA) setPrepareTime(prepareTime int64) {
	c.prepareTime = prepareTime
}

func (c *ConnectionProxyXA) termination(xaBranchXid string) error {
	branchStatus, err := c.resource.GetBranchStatus(xaBranchXid)
	if err != nil {
		c.releaseIfNecessary()
		return fmt.Errorf("failed xa branch [%v] the global transaction has finish, branch status: [%v]", c.xid, branchStatus)
	}
	return nil
}
