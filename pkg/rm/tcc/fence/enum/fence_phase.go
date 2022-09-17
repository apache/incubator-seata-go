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

// FencePhase used to mark the stage of a tcc transaction
type FencePhase byte

const (
	// FencePhaseNotExist fence phase not exist
	FencePhaseNotExist = FencePhase(0)

	// FencePhasePrepare prepare fence phase
	FencePhasePrepare = FencePhase(1)

	// FencePhaseCommit commit fence phase
	FencePhaseCommit = FencePhase(2)

	// FencePhaseRollback rollback fence phase
	FencePhaseRollback = FencePhase(3)
)
