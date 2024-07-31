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

package fence

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"seata.apache.org/seata-go/pkg/tm"
)

type FenceTx struct {
	Ctx           context.Context
	TargetTx      driver.Tx
	TargetFenceTx *sql.Tx
}

func (tx *FenceTx) Commit() error {
	if err := tx.TargetTx.Commit(); err != nil {
		return err
	}

	tx.clearFenceTx()
	return tx.TargetFenceTx.Commit()
}

func (tx *FenceTx) Rollback() error {
	if err := tx.TargetTx.Rollback(); err != nil {
		return err
	}

	tx.clearFenceTx()
	return tx.TargetFenceTx.Rollback()
}

func (tx *FenceTx) clearFenceTx() {
	tm.SetFenceTxBeginedFlag(tx.Ctx, false)
}
