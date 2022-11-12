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

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/tm"
)

type Foo struct {
	id   int
	name string
}

func main() {
	client.Init()
	initService()
	tm.WithGlobalTx(context.Background(), &tm.TransactionInfo{
		Name: "ATSampleLocalGlobalTx",
	}, updateData)
	<-make(chan struct{})
}

func selectData() {
	var foo Foo
	row := db.QueryRow("select * from  foo where id = ? ", 1)
	err := row.Scan(&foo.id, &foo.name)
	if err != nil {
		panic(err)
	}
	fmt.Println(foo)
}

func updateData(ctx context.Context) error {
	sql := "update foo set name=? where id=?"
	ret, err := db.ExecContext(ctx, sql, fmt.Sprintf("Zhangsan-%d", time.Now().UnixMilli()), 1)
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return nil
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return nil
	}
	fmt.Printf("update success： %d.\n", rows)
	return nil
}

func insertData(ctx context.Context) error {
	sqlStr := "insert into foo(name) values (?)"
	ret, err := db.ExecContext(ctx, sqlStr, fmt.Sprintf("Zhangsan-%d", time.Now().UnixMilli()))
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		return err
	}
	theID, err := ret.LastInsertId()
	if err != nil {
		fmt.Printf("get lastinsert ID failed, err:%v\n", err)
		return err
	}
	fmt.Printf("insert success, the id is %d.\n", theID)
	return nil
}
