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

package tm

import "seata.apache.org/seata-go/pkg/protocol/message"

type TransactionManager interface {
	// Begin a new global transaction.
	Begin(applicationId, transactionServiceGroup, name string, timeout int64) (string, error)

	// Commit Global commit.
	Commit(xid string) (message.GlobalStatus, error)

	// Rollback Global rollback.
	Rollback(xid string) (message.GlobalStatus, error)

	// GetStatus Get current status of the give transaction.
	GetStatus(xid string) (message.GlobalStatus, error)

	// GlobalReport Global report.
	GlobalReport(xid string, globalStatus message.GlobalStatus) (message.GlobalStatus, error)
}

// GlobalTransactionRole Identifies whether a global transaction is beginNewGtx or participates in something else
type GlobalTransactionRole int8

const (
	UnKnow      = GlobalTransactionRole(0)
	Launcher    = GlobalTransactionRole(1)
	Participant = GlobalTransactionRole(2)
)

func (role GlobalTransactionRole) String() string {
	switch role {
	case UnKnow:
		return "UnKnow"
	case Launcher:
		return "Launcher"
	case Participant:
		return "Participant"
	}
	return "UnKnow"
}

// Propagation Used to identify the spread of the global transaction enumerated types
type Propagation int8

const (
	// Required
	// The default propagation.
	// If transaction is existing, execute with current transaction,
	// else execute with beginNewGtx transaction.
	Required = Propagation(0)

	// RequiresNew
	// If transaction is existing, suspend it, and then execute business with beginNewGtx transaction.
	RequiresNew = Propagation(1)

	// NotSupported
	// If transaction is existing, suspend it, and then execute business without transaction.
	NotSupported = Propagation(2)

	// Supports
	// If transaction is not existing, execute without global transaction,
	// else execute business with current transaction.
	Supports = Propagation(3)

	// Never
	// If transaction is existing, throw exception,
	// else execute business without transaction.
	Never = Propagation(4)

	// Mandatory
	// If transaction is not existing, throw exception,
	// else execute business with current transaction.
	Mandatory = Propagation(5)
)

func (p Propagation) String() string {
	switch p {
	case Required:
		return "Required"
	case RequiresNew:
		return "RequiresNew"
	case NotSupported:
		return "NotSupported"
	case Supports:
		return "Supports"
	case Never:
		return "Never"
	case Mandatory:
		return "Mandatory"
	}
	return "UnKnow"
}
