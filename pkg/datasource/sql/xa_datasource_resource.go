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
	"fmt"
	"time"

	"github.com/bluele/gcache"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/xa"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/util/log"
)

// XADatasourceResource DataSource proxy for XA mode.
type XADatasourceResource struct {
	db           *DBResource
	shouldBeHeld bool
	keeper       map[string]xa.Holdable
	Cache        map[string]branch.BranchStatus
}

var BranchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()

func (b *XADatasourceResource) init() error {
	return nil
}

func (b *XADatasourceResource) GetDB() *DBResource {
	return b.db
}

func (b *XADatasourceResource) SetDB(db *DBResource) {
	b.db = db
}

func (b *XADatasourceResource) IsShouldBeHeld() bool {
	return b.shouldBeHeld
}

func (b *XADatasourceResource) SetShouldBeHeld(shouldBeHeld bool) {
	b.shouldBeHeld = shouldBeHeld
}

func (b *XADatasourceResource) GetKeeper() map[string]xa.Holdable {
	return b.keeper
}

func (b *XADatasourceResource) SetKeeper(keeper map[string]xa.Holdable) {
	b.keeper = keeper
}

func (b *XADatasourceResource) GetCache() map[string]branch.BranchStatus {
	return b.Cache
}

func (b *XADatasourceResource) SetCache(cache map[string]branch.BranchStatus) {
	b.Cache = cache
}

func (b *XADatasourceResource) GetResourceId() string {
	return b.db.GetResourceId()
}

func (b *XADatasourceResource) Hold(key string, value xa.Holdable) (interface{}, error) {
	if value.IsHeld() {
		var x = b.keeper[key]
		if x != value {
			return nil, fmt.Errorf("something wrong with keeper, keeping[%v] but[%v] is also kept with the same key[%v]", x, value, key)
		}
		return value, nil
	}
	var x = b.keeper[key]
	b.keeper[key] = value
	value.SetHeld(true)
	return x, nil
}

func (b *XADatasourceResource) Release(key string, value xa.Holdable) (xa.Holdable, error) {
	if value.IsHeld() {
		var x = b.keeper[key]
		if x != value {
			return nil, fmt.Errorf("something wrong with keeper, keeping[%v] but[%v] is also kept with the same key[%v]", x, value, key)
		}
		return value, nil
	}
	var x = b.keeper[key]
	b.keeper[key] = value
	value.SetHeld(true)
	return x, nil
}

func (b *XADatasourceResource) Lookup(key string) (*xa.ConnectionProxyXA, bool) {
	holdable, ok := b.keeper[key]
	connectionProxyXA, ok := holdable.(*xa.ConnectionProxyXA)
	if !ok {
		return nil, false
	}
	return connectionProxyXA, ok
}

func (b *XADatasourceResource) GetBranchStatus(xaBranchXid string) (interface{}, error) {
	branchStatus, err := BranchStatusCache.GetIFPresent(xaBranchXid)
	return branchStatus, err
}

func (b *XADatasourceResource) GetDbType() string {
	return b.db.GetDbType().String()
}

func (b *XADatasourceResource) SetDbType(dbType types.DBType) {
	b.db.SetDbType(dbType)
}

// ConnectionForXA get the connection for xa
func (b *XADatasourceResource) ConnectionForXA(ctx context.Context, xaXid xa.XAXid) (*xa.ConnectionProxyXA, error) {
	xaBranchXid := xaXid.String()
	connectionXAProxy, ok := b.Lookup(xaBranchXid)
	if !ok && connectionXAProxy != nil {
		//todo check connection is closed
		return connectionXAProxy, nil
	}

	conn, err := b.db.Conn(ctx)
	if err != nil {
		log.Errorf("get connection for xa id:%s, error:%v", xaXid.String(), err)
		return nil, err
	}

	xaConnection, err := xa.CreateXAConnection(ctx, conn, b.GetDbType())
	if err != nil {
		return nil, err
	}

	connectionProxyXA, err := xa.NewConnectionProxyXA(conn, xaConnection, b, xaXid.String())
	if err != nil {
		log.Errorf("create connection proxy xa id:%s err:%v", xaXid.String(), err)
		return nil, err
	}
	return connectionProxyXA, nil
}
