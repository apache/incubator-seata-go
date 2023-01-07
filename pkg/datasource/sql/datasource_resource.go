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
	"fmt"
	"time"

	"github.com/bluele/gcache"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/protocol/branch"
)

type Holdable interface {
	SetHeld(held bool)
	IsHeld() bool
	ShouldBeHeld() bool
}

type BaseDataSourceResource struct {
	db           *DBResource
	shouldBeHeld bool
	keeper       map[string]interface{}
	Cache        map[string]branch.BranchStatus
}

var BranchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()

func (b *BaseDataSourceResource) init() error {
	return nil
}

func (b *BaseDataSourceResource) GetDB() *DBResource {
	return b.db
}

func (b *BaseDataSourceResource) SetDB(db *DBResource) {
	b.db = db
}

func (b *BaseDataSourceResource) IsShouldBeHeld() bool {
	return b.shouldBeHeld
}

func (b *BaseDataSourceResource) SetShouldBeHeld(shouldBeHeld bool) {
	b.shouldBeHeld = shouldBeHeld
}

func (b *BaseDataSourceResource) GetKeeper() map[string]interface{} {
	return b.keeper
}

func (b *BaseDataSourceResource) SetKeeper(keeper map[string]interface{}) {
	b.keeper = keeper
}

func (b *BaseDataSourceResource) GetCache() map[string]branch.BranchStatus {
	return b.Cache
}

func (b *BaseDataSourceResource) SetCache(cache map[string]branch.BranchStatus) {
	b.Cache = cache
}

func (b *BaseDataSourceResource) GetResourceId() string {
	return b.db.GetResourceId()
}

func (b *BaseDataSourceResource) Hold(key string, value Holdable) (interface{}, error) {
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

func (b *BaseDataSourceResource) Release(key string, value Holdable) (interface{}, error) {
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

func (b *BaseDataSourceResource) GetBranchStatus(xaBranchXid string) (interface{}, error) {
	branchStatus, err := BranchStatusCache.GetIFPresent(xaBranchXid)
	return branchStatus, err
}

func (b *BaseDataSourceResource) GetDbType() string {
	return b.db.dbType.String()
}

func (b *BaseDataSourceResource) SetDbType(dbType types.DBType) {
	b.db.dbType = dbType
}
