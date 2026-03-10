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
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type XATx struct {
	tx *Tx
}

// Commit do commit action
// case 1. no open global-transaction, just do local transaction commit
// case 2. not need flush undolog, is XA mode, do local transaction commit
// case 3. need run AT transaction
func (tx *XATx) Commit() error {
	if err := tx.tx.beforeCommit(); err != nil {
		return err
	}
	return tx.commitOnXA()
}

// Rollback executes XA END(TMFAIL), XA ROLLBACK and reports to TC
func (tx *XATx) Rollback() error {
	originTx := tx.tx

	if !originTx.tranCtx.OpenGlobalTransaction() {
		return nil
	}

	xid := originTx.tranCtx.XID
	branchID := originTx.tranCtx.BranchID

	log.Infof("xa branch [%d/%s] executing XA rollback", branchID, xid)

	if originTx.xaConn != nil {
		if err := originTx.xaConn.Rollback(context.Background()); err != nil {
			log.Errorf("xa branch [%d/%s] XA END(TMFAIL) + XA ROLLBACK failed: %v", branchID, xid, err)
			if originTx.tranCtx.IsBranchRegistered() {
				if reportErr := originTx.report(false); reportErr != nil {
					log.Errorf("xa branch [%d/%s] failed to report rollback failure to TC: %v", branchID, xid, reportErr)
				}
			}
			return err
		}
		log.Infof("xa branch [%d/%s] XA END(TMFAIL) + XA ROLLBACK succeeded", branchID, xid)
	}

	if originTx.tranCtx.IsBranchRegistered() {
		if err := originTx.report(false); err != nil {
			log.Errorf("xa branch [%d/%s] failed to report rollback to TC: %v", branchID, xid, err)
			return err
		}
		log.Infof("xa branch [%d/%s] reported rollback to TC", branchID, xid)
	}

	return nil
}

// commitOnXA executes XA END, XA PREPARE and reports to TC
func (tx *XATx) commitOnXA() error {
	originTx := tx.tx

	if !originTx.tranCtx.OpenGlobalTransaction() {
		return nil
	}

	if originTx.xaConn == nil {
		return fmt.Errorf("xa transaction requires xaConn")
	}

	xid := originTx.tranCtx.XID
	branchID := originTx.tranCtx.BranchID

	log.Infof("xa branch [%d/%s] executing XA END + XA PREPARE", branchID, xid)

	if err := originTx.xaConn.Commit(context.Background()); err != nil {
		log.Errorf("xa branch [%d/%s] XA END + XA PREPARE failed: %v", branchID, xid, err)
		if originTx.tranCtx.IsBranchRegistered() {
			if reportErr := originTx.report(false); reportErr != nil {
				log.Errorf("xa branch [%d/%s] failed to report phase-1 failure to TC: %v", branchID, xid, reportErr)
				return fmt.Errorf("XA PREPARE failed: %w, and report failed: %v", err, reportErr)
			}
		}
		return err
	}

	log.Infof("xa branch [%d/%s] XA END + XA PREPARE succeeded", branchID, xid)

	if originTx.tranCtx.IsBranchRegistered() {
		if err := originTx.report(true); err != nil {
			log.Errorf("xa branch [%d/%s] failed to report phase-1 success to TC: %v", branchID, xid, err)
			return fmt.Errorf("XA PREPARE succeeded but report to TC failed: %w", err)
		}
		log.Infof("xa branch [%d/%s] reported phase-1 success to TC", branchID, xid)
	}

	return nil
}
