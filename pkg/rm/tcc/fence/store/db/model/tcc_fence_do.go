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

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
)

type TCCFenceDO struct {

	// Xid the global transaction id
	Xid string

	// BranchId the branch transaction id
	BranchId int64

	// ActionName the action name
	ActionName string

	// Status the tcc fence status
	// tried: 1; committed: 2; rollbacked: 3; suspended: 4
	Status enum.FenceStatus

	// GmtCreate create time
	GmtCreate time.Time

	// GmtModified update time
	GmtModified time.Time
}
