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

package enum

// FenceStatus Used to mark the state of a branch transaction
type FenceStatus byte

const (
	// StatusTried  phase 1: the commit tried.
	StatusTried = FenceStatus(1)

	// StatusCommitted phase 2: the committed.
	StatusCommitted = FenceStatus(2)

	// StatusRollbacked phase 2: the rollbacked.
	StatusRollbacked = FenceStatus(3)

	// StatusSuspended suspended status.
	StatusSuspended = FenceStatus(4)
)
