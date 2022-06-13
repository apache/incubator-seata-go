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

package transaction

type GlobalTransactionRole int8

const (
	LAUNCHER    GlobalTransactionRole = 0
	PARTICIPANT GlobalTransactionRole = 1
)

type TransactionManager interface {
	// GlobalStatusBegin a new global transaction.
	Begin(applicationId, transactionServiceGroup, name string, timeout int64) (string, error)

	// Global commit.
	Commit(xid string) (GlobalStatus, error)

	//Global rollback.
	Rollback(xid string) (GlobalStatus, error)

	// Get current status of the give transaction.
	GetStatus(xid string) (GlobalStatus, error)

	// Global report.
	GlobalReport(xid string, globalStatus GlobalStatus) (GlobalStatus, error)
}
