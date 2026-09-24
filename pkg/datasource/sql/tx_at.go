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
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

type localCommitStage uint8

const (
	localCommitNotStarted localCommitStage = iota
	localCommitInvoked
	localCommitSucceeded
)

type atCommitOutcome uint8

const (
	atCommitOutcomeRolledBack atCommitOutcome = iota
	atCommitOutcomeRollbackFailed
	atCommitOutcomeCommitUnknown
	atCommitOutcomeCommitted
)

type atCommitError struct {
	cause       error
	rollbackErr error
	reportErr   error
	outcome     atCommitOutcome
}

func (e *atCommitError) Error() string {
	message := fmt.Sprintf("AT commit failed: %v", e.cause)
	if e.rollbackErr != nil {
		message += fmt.Sprintf("; rollback failed: %v", e.rollbackErr)
	}
	if e.reportErr != nil {
		message += fmt.Sprintf("; branch failure report failed: %v", e.reportErr)
	}
	return message
}

func (e *atCommitError) Unwrap() error { return e.cause }

// ATTx
type ATTx struct {
	tx *Tx
}

// Commit do commit action
// case 1. no open global-transaction, just do local transaction commit
// case 2. not need flush undolog, is XA mode, do local transaction commit
// case 3. need run AT transaction
func (tx *ATTx) Commit() error {
	stage, err := tx.doCommit()
	if err == nil {
		return nil
	}
	return tx.finishCommitFailure(stage, err)
}

func (tx *ATTx) Rollback() error {
	err := tx.tx.Rollback()
	if err != nil {

		originTx := tx.tx

		if originTx.tranCtx.OpenGlobalTransaction() && originTx.tranCtx.IsBranchRegistered() {
			originTx.report(false)
		}
	}

	return err
}

func (tx *ATTx) doCommit() (localCommitStage, error) {
	originTx := tx.tx
	stage := localCommitNotStarted

	if err := originTx.beforeCommit(); err != nil {
		return stage, err
	}

	if err := originTx.register(originTx.tranCtx); err != nil {
		return stage, err
	}

	undoLogMgr, err := undo.GetUndoLogManager(originTx.tranCtx.DBType)
	if err != nil {
		return stage, err
	}

	if err = undoLogMgr.FlushUndoLog(originTx.tranCtx, originTx.conn.targetConn); err != nil {
		return stage, err
	}

	stage = localCommitInvoked
	if err := originTx.commitOnLocal(); err != nil {
		return stage, err
	}

	stage = localCommitSucceeded
	originTx.report(true)
	return stage, nil
}

func (tx *ATTx) finishCommitFailure(stage localCommitStage, cause error) error {
	originTx := tx.tx
	commitErr := &atCommitError{cause: cause, outcome: atCommitOutcomeRolledBack}

	switch stage {
	case localCommitNotStarted:
		if err := originTx.Rollback(); err != nil {
			commitErr.rollbackErr = err
			commitErr.outcome = atCommitOutcomeRollbackFailed
			originTx.conn.invalidate()
		}
	case localCommitInvoked:
		commitErr.outcome = atCommitOutcomeCommitUnknown
		originTx.conn.invalidate()
	case localCommitSucceeded:
		commitErr.outcome = atCommitOutcomeCommitted
	}
	if stage != localCommitSucceeded && originTx.tranCtx.IsBranchRegistered() {
		commitErr.reportErr = originTx.report(false)
	}

	return commitErr
}
