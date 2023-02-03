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
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
)

type ConnectionProxyXA struct {
	xaBranchXid             *XABranchXid
	currentAutoCommitStatus bool  `default:"true"`
	xaActive                bool  `default:"false"`
	kept                    bool  `default:"false"`
	rollBacked              bool  `default:"false"`
	branchRegisterTime      int64 `default:"0"`
	prepareTime             int64 `default:"0"`
	timeout                 int   `default:"0"`
	proxyShouldBeHeld       bool  `default:"false"`
	originalConnection      sql.Conn
	xaConnection            XAConnection
	xaResource              exec.XAResource
	resource                sql.BaseDataSourceResource
	xid                     string
}

const timeout int = 60000

func NewConnectionProxyXA(originalConnection sql.Conn, xaConnection XAConnection, resource sql.BaseDataSourceResource, xid string) (*ConnectionProxyXA, error) {
	connectionProxyXA := &ConnectionProxyXA{}

	connectionProxyXA.originalConnection = originalConnection
	connectionProxyXA.xaConnection = xaConnection
	connectionProxyXA.resource = resource
	connectionProxyXA.xid = xid

	connectionProxyXA.proxyShouldBeHeld = connectionProxyXA.resource.IsShouldBeHeld()

	xaResource, err := xaConnection.getXAResource()
	if err != nil {
		return nil, fmt.Errorf("get xa resource failed")
	} else {
		connectionProxyXA.xaResource = xaResource
	}
	var rootContext sql.RootContext
	transactionTimeout, ok := rootContext.GetTimeout()
	if !ok {
		transactionTimeout = timeout
	}
	if transactionTimeout < timeout {
		transactionTimeout = timeout
	}
	connectionProxyXA.timeout = transactionTimeout
	connectionProxyXA.currentAutoCommitStatus = connectionProxyXA.originalConnection.GetAutoCommit()
	if !connectionProxyXA.currentAutoCommitStatus {
		return nil, fmt.Errorf("connection[autocommit=false] as default is NOT supported")
	}

	return connectionProxyXA, nil
}

func (c *ConnectionProxyXA) keepIfNecessary() {
	if c.ShouldBeHeld() {
		c.resource.Hold(c.xaBranchXid.String(), c)
	}
}

func (c *ConnectionProxyXA) releaseIfNecessary() {
	if c.ShouldBeHeld() {
		if c.xaBranchXid == nil {
			if c.IsHeld() {
				c.resource.Release(c.xaBranchXid.String(), c)
			}
		}
	}
}

func (c *ConnectionProxyXA) XaCommit(xid string, branchId int64) error {
	xaXid := Build(xid, branchId)
	err := c.xaResource.Commit(xaXid.String(), false)
	c.releaseIfNecessary()
	return err
}

func (c *ConnectionProxyXA) XaRollbackByBranchId(xid string, branchId int64) {
	xaXid := Build(xid, branchId)
	c.XaRollback(xaXid)
}

func (c *ConnectionProxyXA) XaRollback(xaXid XAXid) error {
	err := c.xaResource.Rollback(xaXid.GetGlobalXid())
	c.releaseIfNecessary()
	return err
}

func (c *ConnectionProxyXA) SetAutoCommit(autoCommit bool) error {
	if c.currentAutoCommitStatus == autoCommit {
		return nil
	}
	if autoCommit {
		if c.xaActive {
			_ = c.Commit()
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
		c.xaBranchXid = Build(c.xid, branchId)
		c.keepIfNecessary()
		err = c.start()
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

func (c *ConnectionProxyXA) Commit() error {
	if c.currentAutoCommitStatus {
		return nil
	}
	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT commit on an inactive session")
	}
	now := time.Now().UnixMilli()
	if c.end(exec.TMSUCCESS) != nil {
		return c.commitErrorHandle()
	}
	if c.checkTimeout(now) != nil {
		return c.commitErrorHandle()
	}
	if c.xaResource.XAPrepare(c.xaBranchXid.String()) != nil {
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
		return fmt.Errorf("Failed to report XA branch commit-failure on [%v] - [%v]", c.xid, c.xaBranchXid.GetBranchId())
	}
	c.cleanXABranchContext()
	return fmt.Errorf("Failed to end(TMSUCCESS)/prepare xa branch on [%v] - [%v]", c.xid, c.xaBranchXid.GetBranchId())
}

func (c *ConnectionProxyXA) Rollback() error {
	if c.currentAutoCommitStatus {
		return nil
	}
	if !c.xaActive || c.xaBranchXid == nil {
		return fmt.Errorf("should NOT rollback on an inactive session")
	}
	if !c.rollBacked {
		if c.xaResource.End(c.xaBranchXid.String(), exec.TMFAIL) != nil {
			return c.rollbackErrorHandle()
		}
		if c.XaRollback(c.xaBranchXid) != nil {
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

func (c *ConnectionProxyXA) start() error {
	err := c.xaResource.Start(c.xaBranchXid.String(), exec.TMNOFLAGS)
	if err := c.termination(c.xaBranchXid.String()); err != nil {
		c.xaResource.End(c.xaBranchXid.String(), exec.TMFAIL)
		c.XaRollback(c.xaBranchXid)
		return err
	}
	return err
}

func (c *ConnectionProxyXA) end(flags int) error {
	err := c.termination(c.xaBranchXid.String())
	if err != nil {
		return err
	}
	err = c.xaResource.End(c.xaBranchXid.String(), flags)
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
	if !c.IsHeld() {
		c.xaBranchXid = nil
	}
}

func (c *ConnectionProxyXA) checkTimeout(now int64) error {
	if now-c.branchRegisterTime > int64(c.timeout) {
		c.XaRollback(c.xaBranchXid)
		return fmt.Errorf("XA branch timeout error")
	}
	return nil
}

func (c *ConnectionProxyXA) Close() error {
	c.rollBacked = false
	if c.IsHeld() && c.ShouldBeHeld() {
		return nil
	}
	c.cleanXABranchContext()
	if err := c.originalConnection.Close(); err != nil {
		return err
	}
	return nil
}

func (c *ConnectionProxyXA) CloseForce() error {
	physicalConn := c.originalConnection
	if err := physicalConn.Close(); err != nil {
		return err
	}
	c.rollBacked = false
	c.cleanXABranchContext()
	if err := c.originalConnection.Close(); err != nil {
		return err
	}
	c.releaseIfNecessary()
	return nil
}

func (c *ConnectionProxyXA) SetHeld(kept bool) {
	c.kept = kept
}

func (c *ConnectionProxyXA) IsHeld() bool {
	return c.kept
}

func (c *ConnectionProxyXA) ShouldBeHeld() bool {
	return c.proxyShouldBeHeld || c.resource.GetDB() != nil
}

func (c *ConnectionProxyXA) GetPrepareTime() int64 {
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
