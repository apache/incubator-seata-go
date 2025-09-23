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

type XATx struct {
	tx *Tx
}

// Commit do commit action
// case 1. no open global-transaction, just do local transaction commit
// case 2. not need flush undolog, is XA mode, do local transaction commit
// case 3. need run AT transaction
func (tx *XATx) Commit() error {
	tx.tx.beforeCommit()
	return tx.commitOnXA()
}

func (tx *XATx) Rollback() error {
	originTx := tx.tx
	if originTx.tranCtx.OpenGlobalTransaction() && originTx.tranCtx.IsBranchRegistered() {
		return originTx.report(false)
	}
	return nil
}

// commitOnXA commit xa and register branch transaction
func (tx *XATx) commitOnXA() error {
	return nil
}
