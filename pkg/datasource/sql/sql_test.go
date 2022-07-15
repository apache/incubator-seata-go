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
	"fmt"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func Test_SQLOpen(t *testing.T) {

	db, err := Open("mysql", "root:polaris@tcp(127.0.0.1:3306)/polaris_server?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	sqlStmt := `
	create table if not exists foo (id integer not null primary key, name text);
	delete from foo;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		t.Fatal(err)
	}

	wait := sync.WaitGroup{}

	txInvoker := func(prefix string, offset, total int) {
		defer wait.Done()

		tx, err := db.BeginTx(context.Background(), &gosql.TxOptions{})
		if err != nil {
			t.Fatal(err)
		}

		stmt, err := tx.Prepare("insert into foo(id, name) values(?, ?)")
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()
		for i := 0; i < total; i++ {
			_, err = stmt.Exec(i+offset, fmt.Sprintf("%s-%03d", prefix, i))
			if err != nil {
				t.Fatal(err)
			}
		}
		err = tx.Commit()
		if err != nil {
			t.Fatal(err)
		}
	}

	wait.Add(1)
	go txInvoker("seata-go-at-1", 0, 10)

	wait.Add(1)
	go txInvoker("seata-go-at-2", 20, 10)

	wait.Wait()
}
