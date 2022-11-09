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

package mysql

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"gotest.tools/assert"
)

// TestGetTableMeta
func TestGetTableMeta(t *testing.T) {
	// local test can annotation t.SkipNow()
	t.SkipNow()

	testTableMeta := func() {
		metaInstance := GetTableMetaInstance()

		db, err := sql.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/seata?multiStatements=true")
		if err != nil {
			t.Fatal(err)
		}

		defer db.Close()

		ctx := context.Background()

		tableMeta, err := metaInstance.GetTableMeta(ctx, "seata_client", "undo_log", nil)
		assert.NilError(t, err)

		t.Logf("%+v", tableMeta)
	}

	t.Run("testTableMeta", func(t *testing.T) {
		testTableMeta()
	})
}
