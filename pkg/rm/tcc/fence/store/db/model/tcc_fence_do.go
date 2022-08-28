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

package model

import (
	"time"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"
)

type TCCFenceDO struct {
	/**
	 * the global transaction id
	 */
	Xid string

	/**
	 * the branch transaction id
	 */
	BranchId int64

	/**
	 * the action name
	 */
	ActionName string

	/**
	 * the tcc fence status
	 * tried: 1; committed: 2; rollbacked: 3; suspended: 4
	 */
	Status constant.FenceStatus

	/**
	 * create time
	 */
	GmtCreate time.Time

	/**
	 * update time
	 */
	GmtModified time.Time
}
