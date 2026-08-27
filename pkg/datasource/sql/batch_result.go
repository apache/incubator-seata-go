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
	gosql "database/sql"
	"errors"
	"fmt"
)

// NoFailedBatchItem indicates that a batch failure is not associated with an item.
const NoFailedBatchItem = -1

// BatchPhase identifies the phase that caused a batch to fail.
type BatchPhase uint8

const (
	BatchPhaseNone BatchPhase = iota
	BatchPhaseValidate
	BatchPhaseBegin
	BatchPhaseExecute
	BatchPhaseCommit
)

// BatchTransactionState describes the transaction outcome known to the batch API.
type BatchTransactionState uint8

const (
	BatchTransactionNotStarted BatchTransactionState = iota
	BatchTransactionPending
	BatchTransactionCommitted
	BatchTransactionRolledBack
	BatchTransactionRollbackFailed
	BatchTransactionCommitUnknown
)

// BatchItemState describes the execution state of one ordered batch item.
type BatchItemState uint8

const (
	BatchItemNotExecuted BatchItemState = iota
	BatchItemExecuted
	BatchItemFailed
)

// BatchOutcome describes where a batch failed and its transaction outcome.
type BatchOutcome struct {
	FailedIndex      int
	FailurePhase     BatchPhase
	TransactionState BatchTransactionState
}

// BatchResult contains one result for each input argument group, in input order.
type BatchResult struct {
	Items   []ItemResult
	Outcome BatchOutcome
}

// ItemResult contains the execution result of one batch item.
// BatchItemExecuted means execution succeeded; durability is described by BatchResult.Outcome.
type ItemResult struct {
	Index int
	State BatchItemState

	execErr         error
	lastInsertID    int64
	lastInsertIDErr error
	rowsAffected    int64
	rowsAffectedErr error
}

var errBatchItemResultUnavailable = errors.New("batch item result is unavailable")

// Err returns the item's execution error, if any.
func (r ItemResult) Err() error {
	return r.execErr
}

// LastInsertId returns the driver's snapshotted LastInsertId result.
func (r ItemResult) LastInsertId() (int64, error) {
	if r.State != BatchItemExecuted {
		return 0, errBatchItemResultUnavailable
	}
	return r.lastInsertID, r.lastInsertIDErr
}

// RowsAffected returns the driver's snapshotted RowsAffected result.
func (r ItemResult) RowsAffected() (int64, error) {
	if r.State != BatchItemExecuted {
		return 0, errBatchItemResultUnavailable
	}
	return r.rowsAffected, r.rowsAffectedErr
}

func (r *ItemResult) recordResult(result gosql.Result) {
	r.State = BatchItemExecuted
	r.lastInsertID, r.lastInsertIDErr = result.LastInsertId()
	r.rowsAffected, r.rowsAffectedErr = result.RowsAffected()
}

// BatchError describes a batch failure while preserving its underlying errors.
type BatchError struct {
	Outcome     BatchOutcome
	Cause       error
	RollbackErr error
}

func (e *BatchError) Error() string {
	message := "batch failed"
	if e.Cause != nil {
		message = e.Cause.Error()
	}
	if e.RollbackErr != nil {
		return fmt.Sprintf("%s; %v", message, e.RollbackErr)
	}
	return message
}

// Unwrap exposes both the primary failure and a rollback failure to errors.Is and errors.As.
func (e *BatchError) Unwrap() []error {
	errs := make([]error, 0, 2)
	if e.Cause != nil {
		errs = append(errs, e.Cause)
	}
	if e.RollbackErr != nil {
		errs = append(errs, e.RollbackErr)
	}
	return errs
}

func newBatchResult(itemCount int, state BatchTransactionState) BatchResult {
	result := BatchResult{
		Items: make([]ItemResult, itemCount),
		Outcome: BatchOutcome{
			FailedIndex:      NoFailedBatchItem,
			TransactionState: state,
		},
	}
	for i := range result.Items {
		result.Items[i].Index = i
	}
	return result
}

func newBatchError(result BatchResult, cause, rollbackErr error) *BatchError {
	return &BatchError{Outcome: result.Outcome, Cause: cause, RollbackErr: rollbackErr}
}
