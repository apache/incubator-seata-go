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

package metrics

// Shared metric names, tag keys, and tag values.
const (
	MeterNameTransaction = "seata.transaction"

	TagKeyApplicationID = "applicationId"
	TagKeyGroup         = "group"
	TagKeyName          = "name"
	TagKeyRole          = "role"
	TagKeyMeter         = "meter"
	TagKeyStatistic     = "statistic"
	TagKeyStatus        = "status"
	TagKeyRetry         = "retry"

	RoleValueTC = "tc"
	RoleValueTM = "tm"
	RoleValueRM = "rm"

	MeterValueGauge   = "gauge"
	MeterValueCounter = "counter"
	MeterValueTimer   = "timer"

	StatisticValueCount   = "count"
	StatisticValueTotal   = "total"
	StatisticValueMax     = "max"
	StatisticValueAverage = "average"

	StatusValueActive          = "active"
	StatusValueCommitted       = "committed"
	StatusValueRollbacked      = "rollbacked"
	StatusValueFailed          = "failed"
	StatusValueTwoPhaseTimeout = "2phaseTimeout"

	StatusValueAfterCommitted  = "AfterCommitted"
	StatusValueAfterRollbacked = "AfterRollbacked"
)
