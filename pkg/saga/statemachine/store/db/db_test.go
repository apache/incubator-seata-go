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

package db

import (
	"database/sql"
	"os"
	"sync"
)

var (
	oncePrepareDB sync.Once
	db            *sql.DB
)

func prepareDB() {
	oncePrepareDB.Do(func() {
		var err error
		db, err = sql.Open("sqlite3", ":memory:")
		query_, err := os.ReadFile("testdata/sql/saga/sqlite_init.sql")
		initScript := string(query_)
		if err != nil {
			panic(err)
		}
		if _, err := db.Exec(initScript); err != nil {
			panic(err)
		}
	})

}
