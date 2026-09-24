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
	"context"
	gosql "database/sql"
	"errors"
	"fmt"
	"strings"
)

// The batch helpers in this file are driver-agnostic. They provide ordered
// execution within one database/sql transaction only. When used with a Seata
// driver, AT-specific behavior is provided by that driver and its executors.

var (
	errNilBatchDB            = errors.New("batch db is nil")
	errNilBatchTx            = errors.New("batch tx is nil")
	errEmptyBatchQuery       = errors.New("batch query is empty")
	errInconsistentBatchArgs = errors.New("inconsistent batch argument count")
)

// batchExecContext describes one semantic batch.
// A batch contains exactly one SQL template and an ordered set of argument groups.
// Its lifetime is limited to one batch invocation.
type batchExecContext struct {
	query     string
	batchArgs [][]any
}

func newBatchExecContext(ctx context.Context, query string, batchArgs [][]any) (*batchExecContext, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(query) == "" {
		return nil, errEmptyBatchQuery
	}

	if len(batchArgs) > 1 {
		expectedArgCount := len(batchArgs[0])

		for i := 1; i < len(batchArgs); i++ {
			if len(batchArgs[i]) != expectedArgCount {
				return nil, fmt.Errorf(
					"%w: batch item %d has %d arguments, expected %d",
					errInconsistentBatchArgs, i, len(batchArgs[i]), expectedArgCount,
				)
			}
		}
	}

	return &batchExecContext{query: query, batchArgs: batchArgs}, nil
}

// ExecBatchContext executes one SQL template with multiple argument groups.
//
// The transaction is owned by this function.
// All batch items are executed sequentially in one local transaction.
// The first execution error stops the batch and causes the whole transaction to be rolled back.
// The returned result remains valid on error and contains one ordered item for each argument group.
//
// When used with a Seata AT driver in a global transaction, all batch items
// participate in the same local transaction and therefore share the same AT
// branch lifecycle. AT-specific image and undo-log handling remains the
// responsibility of the Seata driver and its executors.
//
// This is the batch counterpart of database/sql.DB.ExecContext:
// when callers need to combine the batch with other statements in the same transaction,
// they should begin a transaction explicitly and use ExecBatchInTxContext.
func ExecBatchContext(ctx context.Context, db *gosql.DB, query string, batchArgs [][]any) (BatchResult, error) {
	result := newBatchResult(len(batchArgs), BatchTransactionNotStarted)
	if db == nil {
		result.Outcome.FailurePhase = BatchPhaseValidate
		return result, newBatchError(result, errNilBatchDB, nil)
	}

	batchCtx, err := newBatchExecContext(ctx, query, batchArgs)
	if err != nil {
		result.Outcome.FailurePhase = BatchPhaseValidate
		return result, newBatchError(result, err, nil)
	}

	if len(batchCtx.batchArgs) == 0 {
		return result, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		result.Outcome.FailurePhase = BatchPhaseBegin
		cause := fmt.Errorf("begin batch transaction: %w", err)
		return result, newBatchError(result, cause, nil)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// database/sql may already have rolled back the transaction after context cancellation.
	// Do not let ErrTxDone hide the original execution error.
	if failedIndex, err := executeBatch(ctx, tx, batchCtx, &result); err != nil {
		result.Outcome.FailedIndex = failedIndex
		result.Outcome.FailurePhase = BatchPhaseExecute
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, gosql.ErrTxDone) {
			result.Outcome.TransactionState = BatchTransactionRollbackFailed
			rollbackErr = fmt.Errorf("rollback batch transaction: %w", rollbackErr)
			return result, newBatchError(result, err, rollbackErr)
		}
		result.Outcome.TransactionState = BatchTransactionRolledBack
		return result, newBatchError(result, err, nil)
	}

	if err := tx.Commit(); err != nil {
		result.Outcome.FailurePhase = BatchPhaseCommit
		result.Outcome.TransactionState = BatchTransactionCommitUnknown
		var rollbackErr error
		var commitErr *atCommitError
		if errors.As(err, &commitErr) {
			switch commitErr.outcome {
			case atCommitOutcomeRolledBack:
				result.Outcome.TransactionState = BatchTransactionRolledBack
			case atCommitOutcomeRollbackFailed:
				result.Outcome.TransactionState = BatchTransactionRollbackFailed
				rollbackErr = commitErr.rollbackErr
			case atCommitOutcomeCommitted:
				result.Outcome.TransactionState = BatchTransactionCommitted
			}
		}
		cause := fmt.Errorf("commit batch transaction: %w", err)
		return result, newBatchError(result, cause, rollbackErr)
	}
	result.Outcome.TransactionState = BatchTransactionCommitted
	return result, nil
}

// ExecBatchInTxContext executes one SQL template with multiple argument groups
// inside a caller-owned transaction.
//
// When used with a Seata AT driver, the batch joins the caller's existing transaction
// and doesn't create or finish an AT branch on its own.
//
// The function never commits or rolls back tx.
// If an item fails, execution stops immediately and the error is returned to the caller,
// which remains responsible for the transaction lifecycle.
// The returned transaction state is pending because the caller owns its final outcome.
func ExecBatchInTxContext(ctx context.Context, tx *gosql.Tx, query string, batchArgs [][]any) (BatchResult, error) {
	result := newBatchResult(len(batchArgs), BatchTransactionNotStarted)
	if tx == nil {
		result.Outcome.FailurePhase = BatchPhaseValidate
		return result, newBatchError(result, errNilBatchTx, nil)
	}
	result.Outcome.TransactionState = BatchTransactionPending

	batchCtx, err := newBatchExecContext(ctx, query, batchArgs)
	if err != nil {
		result.Outcome.FailurePhase = BatchPhaseValidate
		return result, newBatchError(result, err, nil)
	}

	if len(batchCtx.batchArgs) == 0 {
		return result, nil
	}

	failedIndex, err := executeBatch(ctx, tx, batchCtx, &result)
	if err != nil {
		result.Outcome.FailedIndex = failedIndex
		result.Outcome.FailurePhase = BatchPhaseExecute
		return result, newBatchError(result, err, nil)
	}
	return result, nil
}

// executeBatch is the semantic batch execution core.
//
// Regardless of whether the transaction was created by ExecBatchContext or
// supplied by the caller, all items are executed on the same *sql.Tx and
// therefore the same underlying database transaction.
func executeBatch(ctx context.Context, tx *gosql.Tx, batchCtx *batchExecContext, result *BatchResult) (int, error) {
	for i, arg := range batchCtx.batchArgs {
		sqlResult, err := tx.ExecContext(ctx, batchCtx.query, arg...)
		if err != nil {
			result.Items[i].State = BatchItemFailed
			result.Items[i].execErr = err
			return i, fmt.Errorf("execute batch item %d: %w", i, err)
		}
		result.Items[i].recordResult(sqlResult)
	}
	return NoFailedBatchItem, nil
}
