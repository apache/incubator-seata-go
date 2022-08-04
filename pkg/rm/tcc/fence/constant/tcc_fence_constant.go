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

package constant

const (
	// StatusTried
	// PHASE 1: The Commit tried.
	StatusTried int32 = 1

	// StatusCommitted
	// PHASE 2: The Committed.
	StatusCommitted int32 = 2

	// StatusRollbacked
	// PHASE 2: The Rollbacked.
	StatusRollbacked int32 = 3

	// StatusSuspended
	// Suspended status.
	StatusSuspended int32 = 4

	// FencePhaseNotExist
	// fence phase not exist
	FencePhaseNotExist int32 = 0

	// FencePhasePrepare
	// prepare fence phase
	FencePhasePrepare int32 = 1

	// FencePhaseCommit
	// commit fence phase
	FencePhaseCommit int32 = 2

	// FencePhaseRollback
	// rollback fence phase
	FencePhaseRollback int32 = 3
)
