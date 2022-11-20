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
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/client"
)

type OrderTbl struct {
	id            int
	userID        string
	commodityCode string
	count         int64
	money         int64
	descs         string
}

var (
	count         = time.Now().UnixMilli()
	userID        = fmt.Sprintf("NO-%d", count)
	commodityCode = fmt.Sprintf("C%d", count)
	descs         = fmt.Sprintf("desc %d", count)
)

func main() {
	client.Init()
	initService()

	insertId := insertData()

	//insertDuplicateData(insertId)

	selectData(insertId)

	updateData(insertId)

	selectData(insertId)

	deleteData(insertId)

	selectData(insertId)

	userIds := batchInsertData()

	batchDeleteData(userIds)

	<-make(chan struct{})
}

func insertData() int64 {
	ret, err := db.Exec("insert into order_tbl (`user_id`, `commodity_code`, `count`, `money`, `descs`) values (?, ?, ?, ?, ?)",
		userID, commodityCode, 100, 100, descs)
	if err != nil {
		panic(err)
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		panic(err)
	}

	insertId, err := ret.LastInsertId()
	if err != nil {
		fmt.Printf("get insert id failed, err:%v\n", err)
		panic(err)
	}

	fmt.Printf("insert success： %d.\n", rows)
	return insertId
}

func batchInsertData() []string {
	var userIds []string
	sql := "insert into order_tbl (`user_id`, `commodity_code`, `count`, `money`, `descs`) values "
	for i := 0; i < 5; i++ {
		tmpCount := time.Now().UnixMilli()
		tmpUserID := fmt.Sprintf("NO-%d", tmpCount)
		userIds = append(userIds, tmpUserID)
		tmpCommodityCode := fmt.Sprintf("C%d", tmpCount)
		tmpDescs := fmt.Sprintf("desc %d", tmpCount)
		sql += fmt.Sprintf("('%s','%s',1000,100,'%s'),", tmpUserID, tmpCommodityCode, tmpDescs)
	}
	sql = sql[:len(sql)-1]

	ret, err := db.Exec(sql)
	if err != nil {
		panic(err)
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		panic(err)
	}
	fmt.Printf("insert success： %d.\n", rows)
	return userIds
}

func insertDuplicateData(id int64) int64 {
	ret, err := db.Exec("insert into order_tbl (`id`, `user_id`, `commodity_code`, `count`, `money`, `descs`) values (?,?, ?, ?, ?, ?)",
		id, userID, commodityCode, 100, 100, descs)
	if err != nil {
		panic(err)
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		panic(err)
	}

	insertId, err := ret.LastInsertId()
	if err != nil {
		fmt.Printf("get insert id failed, err:%v\n", err)
		panic(err)
	}

	fmt.Printf("insert success： %d.\n", rows)
	return insertId
}

func selectData(id int64) {
	var orderTbl OrderTbl
	row := db.QueryRow("select id,user_id,commodity_code,count,money,descs from  order_tbl where id = ? ", id)
	err := row.Scan(&orderTbl.id, &orderTbl.userID, &orderTbl.commodityCode, &orderTbl.count, &orderTbl.money, &orderTbl.descs)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("select return null")
			return
		}
		panic(err)
	}
	fmt.Printf("select --> : %v\n", orderTbl)
}

func updateData(insertID int64) error {
	sql := "update order_tbl set descs=? where id=?"
	ret, err := db.Exec(sql, fmt.Sprintf("NewDescs-%d", time.Now().UnixMilli()), insertID)
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

func deleteData(insertID int64) error {
	sql := "delete from order_tbl where id=?"
	ret, err := db.Exec(sql, insertID)
	if err != nil {
		fmt.Printf("delete failed, err:%v\n", err)
		return nil
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("delete failed, err:%v\n", err)
		return nil
	}
	fmt.Printf("delete success： %d.\n", rows)
	return nil
}

func batchDeleteData(userIds []string) error {
	var sql string
	for _, v := range userIds {
		sql += fmt.Sprintf("delete from order_tbl where user_id = '%s';", v)
	}
	ret, err := db.Exec(sql)
	if err != nil {
		fmt.Printf("batch delete failed, err:%v\n", err)
		return nil
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("batch delete failed, err:%v\n", err)
		return nil
	}
	fmt.Printf("batch delete success： %d.\n", rows)
	return nil
}
