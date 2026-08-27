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
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type panickingBatchValuer struct{}

func (panickingBatchValuer) Value() (driver.Value, error) {
	panic("batch valuer panic")
}

func TestExecBatchContextRejectsInconsistentArgumentCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	query := "UPDATE user SET name = ? WHERE id = ?"

	result, err := ExecBatchContext(ctx, db, query, [][]any{{"user1", 1}, {"user2"}, {"user3", 3}})

	require.ErrorIs(t, err, errInconsistentBatchArgs)
	require.Contains(t, err.Error(), "batch item 1")
	require.Contains(t, err.Error(), "has 1 arguments, expected 2")
	require.Equal(t, BatchPhaseValidate, result.Outcome.FailurePhase)
	require.Equal(t, BatchTransactionNotStarted, result.Outcome.TransactionState)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchContextCommitOnSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	query := "UPDATE user SET name = ? WHERE id = ?"
	metadataErr := errors.New("result metadata unavailable")

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs("user1", 1).
		WillReturnResult(sqlmock.NewResult(10, 1))

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs("user2", 2).
		WillReturnResult(sqlmock.NewResult(20, 2))

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs("user3", 3).
		WillReturnResult(sqlmock.NewErrorResult(metadataErr))

	mock.ExpectCommit()

	result, err := ExecBatchContext(ctx, db, query, [][]any{{"user1", 1}, {"user2", 2}, {"user3", 3}})
	require.NoError(t, err)
	require.Equal(t, BatchTransactionCommitted, result.Outcome.TransactionState)
	require.Equal(t, NoFailedBatchItem, result.Outcome.FailedIndex)
	require.Len(t, result.Items, 3)

	for i := range result.Items {
		require.Equal(t, i, result.Items[i].Index)
		require.Equal(t, BatchItemExecuted, result.Items[i].State)
	}

	lastInsertID, err := result.Items[0].LastInsertId()
	require.NoError(t, err)
	require.EqualValues(t, 10, lastInsertID)
	rowsAffected, err := result.Items[1].RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, 2, rowsAffected)
	_, err = result.Items[2].LastInsertId()
	require.ErrorIs(t, err, metadataErr)
	_, err = result.Items[2].RowsAffected()
	require.ErrorIs(t, err, metadataErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchContextRollbackOnItemFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	query := "UPDATE user SET name = ? WHERE id = ?"
	execErr := errors.New("execute failed")

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs("user1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs("user2", 2).
		WillReturnError(execErr)

	mock.ExpectRollback()

	result, err := ExecBatchContext(ctx, db, query, [][]any{{"user1", 1}, {"user2", 2}, {"user3", 3}})

	require.Error(t, err)
	require.ErrorIs(t, err, execErr)
	require.Contains(t, err.Error(), "batch item 1")
	require.Equal(t, BatchOutcome{
		FailedIndex:      1,
		FailurePhase:     BatchPhaseExecute,
		TransactionState: BatchTransactionRolledBack,
	}, result.Outcome)
	require.Equal(t, BatchItemExecuted, result.Items[0].State)
	require.Equal(t, BatchItemFailed, result.Items[1].State)
	require.Equal(t, BatchItemNotExecuted, result.Items[2].State)
	require.ErrorIs(t, result.Items[1].Err(), execErr)
	rowsAffected, resultErr := result.Items[0].RowsAffected()
	require.NoError(t, resultErr)
	require.EqualValues(t, 1, rowsAffected)

	var batchErr *BatchError
	require.ErrorAs(t, err, &batchErr)
	require.Equal(t, result.Outcome, batchErr.Outcome)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchContextReportsRollbackFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	execErr := errors.New("execute failed")
	rollbackErr := errors.New("rollback failed")
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnError(execErr)
	mock.ExpectRollback().WillReturnError(rollbackErr)

	result, err := ExecBatchContext(context.Background(), db, "UPDATE", [][]any{{}})

	require.ErrorIs(t, err, execErr)
	require.ErrorIs(t, err, rollbackErr)
	require.Equal(t, BatchTransactionRollbackFailed, result.Outcome.TransactionState)
	require.Equal(t, 0, result.Outcome.FailedIndex)

	var batchErr *BatchError
	require.ErrorAs(t, err, &batchErr)
	require.ErrorIs(t, batchErr.RollbackErr, rollbackErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchContextPreservesResultsWhenCommitOutcomeUnknown(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	commitErr := errors.New("commit failed")
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(10, 2))
	mock.ExpectCommit().WillReturnError(commitErr)

	result, err := ExecBatchContext(context.Background(), db, "UPDATE", [][]any{{}})

	require.ErrorIs(t, err, commitErr)
	require.Equal(t, BatchPhaseCommit, result.Outcome.FailurePhase)
	require.Equal(t, BatchTransactionCommitUnknown, result.Outcome.TransactionState)
	require.Equal(t, NoFailedBatchItem, result.Outcome.FailedIndex)
	require.Equal(t, BatchItemExecuted, result.Items[0].State)
	rowsAffected, resultErr := result.Items[0].RowsAffected()
	require.NoError(t, resultErr)
	require.EqualValues(t, 2, rowsAffected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchContextRollsBackWhenExecutionPanics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	require.PanicsWithValue(t, "batch valuer panic", func() {
		_, _ = ExecBatchContext(context.Background(), db, "UPDATE", [][]any{{panickingBatchValuer{}}})
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchInTxContextKeepsCallerTransactionOpen(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	batchQuery := "UPDATE user SET name = ? WHERE id=?"
	singleQuery := "UPDATE account SET balance = ? WHERE id = ?"

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(batchQuery)).
		WithArgs("user1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(batchQuery)).
		WithArgs("user2", 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := ExecBatchInTxContext(ctx, tx, batchQuery, [][]any{{"user1", 1}, {"user2", 2}})
	require.NoError(t, err)
	require.Equal(t, BatchTransactionPending, result.Outcome.TransactionState)
	require.Equal(t, BatchItemExecuted, result.Items[0].State)
	require.Equal(t, BatchItemExecuted, result.Items[1].State)

	// If the batch API committed the transaction internally,this statement would fail with sql.ErrTxDone
	mock.ExpectExec(regexp.QuoteMeta(singleQuery)).
		WithArgs(100, 10).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err = tx.ExecContext(ctx, singleQuery, 100, 10)
	require.NoError(t, err)

	mock.ExpectCommit()
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchInTxContextDoesNotRollbackCallerTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	query := "UPDATE user SET name = ? WHERE id = ?"
	execErr := errors.New("execute failed")

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs("user1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs("user2", 2).
		WillReturnError(execErr)

	result, err := ExecBatchInTxContext(ctx, tx, query, [][]any{{"user1", 1}, {"user2", 2}})
	require.ErrorIs(t, err, execErr)
	require.Equal(t, BatchTransactionPending, result.Outcome.TransactionState)
	require.Equal(t, 1, result.Outcome.FailedIndex)
	require.Equal(t, BatchItemExecuted, result.Items[0].State)
	require.Equal(t, BatchItemFailed, result.Items[1].State)

	// The caller still owns the transaction
	mock.ExpectRollback()

	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchContextEmptyBatchIsNoop(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	result, err := ExecBatchContext(context.Background(), db, "UPDATE user SET name = ? WHERE id = ?", nil)
	require.NoError(t, err)
	require.Empty(t, result.Items)
	require.Equal(t, BatchTransactionNotStarted, result.Outcome.TransactionState)
	// No Begin/Exec/Commit should happen.
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecBatchContextDoesNotLeakStateAfterFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	query := "UPDATE user SET name = ? WHERE id = ?"

	firstBatchErr := errors.New("first batch failed")

	// Batch A.
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs("usera1", 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs("usera2", 2).WillReturnError(firstBatchErr)
	mock.ExpectRollback()

	_, err = ExecBatchContext(ctx, db, query, [][]any{{"usera1", 1}, {"usera2", 2}})
	require.ErrorIs(t, err, firstBatchErr)

	// Batch B.
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs("userb1", 3).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs("userb2", 4).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	_, err = ExecBatchContext(ctx, db, query, [][]any{{"userb1", 3}, {"userb2", 4}})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
