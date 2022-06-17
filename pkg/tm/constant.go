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

import "github.com/seata/seata-go/pkg/protocol/message"

const (
	LAUNCHER    GlobalTransactionRole = 0
	PARTICIPANT GlobalTransactionRole = 1
)

type (
	Propagation           int8
	GlobalTransactionRole int8
)

type TransactionManager interface {
	// GlobalStatusBegin a new global transaction.
	Begin(applicationId, transactionServiceGroup, name string, timeout int64) (string, error)

	// Global commit.
	Commit(xid string) (message.GlobalStatus, error)

	//Global rollback.
	Rollback(xid string) (message.GlobalStatus, error)

	// Get current status of the give transaction.
	GetStatus(xid string) (message.GlobalStatus, error)

	// Global report.
	GlobalReport(xid string, globalStatus message.GlobalStatus) (message.GlobalStatus, error)
}

const (

	/**
	 * The REQUIRED.
	 * The default propagation.
	 *
	 * <p>
	 * If transaction is existing, execute with current transaction,
	 * else execute with new transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx == null) {
	 *         try {
	 *             tx = beginNewTransaction(); // begin new transaction, is not existing
	 *             Object rs = business.execute(); // execute with new transaction
	 *             commitTransaction(tx);
	 *             return rs;
	 *         } catch (Exception ex) {
	 *             rollbackTransaction(tx);
	 *             throw ex;
	 *         }
	 *     } else {
	 *         return business.execute(); // execute with current transaction
	 *     }
	 * </pre></code>
	 * </p>
	 */
	REQUIRED = Propagation(0)

	/**
	 * The REQUIRES_NEW.
	 *
	 * <p>
	 * If transaction is existing, suspend it, and then execute business with new transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     try {
	 *         if (tx != null) {
	 *             suspendedResource = suspendTransaction(tx); // suspend current transaction
	 *         }
	 *         try {
	 *             tx = beginNewTransaction(); // begin new transaction
	 *             Object rs = business.execute(); // execute with new transaction
	 *             commitTransaction(tx);
	 *             return rs;
	 *         } catch (Exception ex) {
	 *             rollbackTransaction(tx);
	 *             throw ex;
	 *         }
	 *     } finally {
	 *         if (suspendedResource != null) {
	 *             resumeTransaction(suspendedResource); // resume transaction
	 *         }
	 *     }
	 * </pre></code>
	 * </p>
	 */
	REQUIRES_NEW = Propagation(1)

	/**
	 * The NOT_SUPPORTED.
	 *
	 * <p>
	 * If transaction is existing, suspend it, and then execute business without transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     try {
	 *         if (tx != null) {
	 *             suspendedResource = suspendTransaction(tx); // suspend current transaction
	 *         }
	 *         return business.execute(); // execute without transaction
	 *     } finally {
	 *         if (suspendedResource != null) {
	 *             resumeTransaction(suspendedResource); // resume transaction
	 *         }
	 *     }
	 * </pre></code>
	 * </p>
	 */
	NOT_SUPPORTED = Propagation(2)

	/**
	 * The SUPPORTS.
	 *
	 * <p>
	 * If transaction is not existing, execute without global transaction,
	 * else execute business with current transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx != null) {
	 *         return business.execute(); // execute with current transaction
	 *     } else {
	 *         return business.execute(); // execute without transaction
	 *     }
	 * </pre></code>
	 * </p>
	 */
	SUPPORTS = Propagation(3)

	/**
	 * The NEVER.
	 *
	 * <p>
	 * If transaction is existing, throw exception,
	 * else execute business without transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx != null) {
	 *         throw new TransactionException("existing transaction");
	 *     }
	 *     return business.execute(); // execute without transaction
	 * </pre></code>
	 * </p>
	 */
	NEVER = Propagation(4)

	/**
	 * The MANDATORY.
	 *
	 * <p>
	 * If transaction is not existing, throw exception,
	 * else execute business with current transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx == null) {
	 *         throw new TransactionException("not existing transaction");
	 *     }
	 *     return business.execute(); // execute with current transaction
	 * </pre></code>
	 * </p>
	 */
	MANDATORY = Propagation(5)
)
