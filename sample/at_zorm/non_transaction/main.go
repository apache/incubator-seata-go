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
	"errors"
	"fmt"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/seata/seata-go/pkg/client"
)

type OrderTbl struct {
	zorm.EntityStruct

	Id            int64  `column:"id"`
	UserId        string `column:"user_id"`
	CommodityCode string `column:"commodity_code"`
	Count         int64  `column:"count"`
	Money         int64  `column:"money"`
	Descs         string `column:"descs"`
}

func (entity *OrderTbl) GetTableName() string {
	return "order_tbl"
}

func (entity *OrderTbl) GetPKColumnName() string {
	return "id"
}

var (
	count         = time.Now().UnixMilli()
	userID        = fmt.Sprintf("NO-%d", count)
	commodityCode = fmt.Sprintf("C%d", count)
	descs         = fmt.Sprintf("desc %d", count)
)

func main() {
	client.InitPath("./sample/conf/seatago.yml")
	initService()

	insertId := insertData()
	fmt.Printf("insertId = %d\n", insertId)

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
	var (
		insertId int64
		ctx      = context.Background()
	)

	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// insert
		var (
			userid = userID
			sql    = "insert into order_tbl (`user_id`, `commodity_code`, `count`, `money`, `descs`) values (?, ?, ?, ?, ?)"
		)
		finder := zorm.NewFinder()
		finder = finder.Append(sql, userid, commodityCode, 100, 100, descs)
		affected, err := zorm.UpdateFinder(ctx, finder)
		if err != nil {
			fmt.Printf("insert failed, err:%v\n", err)
			return nil, err
		}
		fmt.Printf("insert success： %d.\n", affected)

		// select
		qfinder := zorm.NewFinder()
		qfinder.Append("select id from order_tbl where user_id =?", userid)
		has, err := zorm.QueryRow(ctx, qfinder, &insertId)
		if err != nil {
			fmt.Printf("get insert id failed, err:%v\n", err)
			return nil, err
		}
		if !has {
			fmt.Printf("get insert id empty.\n")
			return nil, err
		}
		return nil, err
	})
	if err != nil {
		fmt.Printf("insert id failed, err:%v\n", err)
		panic(err)
	}
	return insertId
}

func selectData(id int64) {
	var orderTbl OrderTbl

	_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {

		finder := zorm.NewFinder()
		finder.Append("select id, user_id, commodity_code, count, money, descs from order_tbl where id = ? ", id)
		has, err := zorm.QueryRow(ctx, finder, &orderTbl)
		if err != nil {
			fmt.Printf("select return err: %v\n", err)
			return nil, err
		}
		if !has {
			fmt.Println("select return null")
			return nil, errors.New("select return null")
		}

		return nil, err
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("select --> : %v\n", orderTbl)
}

func batchInsertData() []string {
	var (
		rows    int
		userIds []string
		ctx     = context.Background()
	)

	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		orderSlice := make([]zorm.IEntityStruct, 0)
		for i := 0; i < 5; i++ {
			tmpCount := time.Now().UnixMilli()

			var tmp OrderTbl
			tmp.Id = int64(i)
			tmp.UserId = fmt.Sprintf("NO-%d", tmpCount)
			tmp.CommodityCode = fmt.Sprintf("%d", tmpCount)
			tmp.Count = int64(i)
			tmp.Money = 100
			tmp.Descs = fmt.Sprintf("desc %d", tmpCount)

			userIds = append(userIds, tmp.UserId)
			orderSlice = append(orderSlice, &tmp)
		}

		var err error
		rows, err = zorm.InsertSlice(ctx, orderSlice)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return nil, err
	})
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		panic(err)
	}

	fmt.Printf("insert success： %d.\n", rows)
	return userIds
}

func insertDuplicateData(id int64) int64 {
	var (
		insertId int64
		ctx      = context.Background()
	)

	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		// insert
		var (
			userid = userID
			sql    = "insert into order_tbl (`id`, `user_id`, `commodity_code`, `count`, `money`, `descs`) values (?,?, ?, ?, ?, ?)"
		)
		finder := zorm.NewFinder()
		finder = finder.Append(sql, id, userid, commodityCode, 100, 100, descs)
		affected, err := zorm.UpdateFinder(ctx, finder)
		if err != nil {
			fmt.Printf("insert failed, err:%v\n", err)
			return nil, err
		}
		fmt.Printf("insert success： %d.\n", affected)

		// select
		qfinder := zorm.NewFinder()
		qfinder.Append("select id from order_tbl where user_id =?", userid)
		has, err := zorm.QueryRow(ctx, qfinder, &insertId)
		if err != nil {
			fmt.Printf("get insert id failed, err:%v\n", err)
			return nil, err
		}
		if !has {
			fmt.Printf("get insert id empty.\n")
			return nil, errors.New("get insert id empty.")
		}
		return nil, err
	})
	if err != nil {
		fmt.Printf("insert id failed, err:%v\n", err)
		panic(err)
	}

	return insertId
}

func updateData(insertID int64) error {

	_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
		sql := "update order_tbl set descs=? where id=?"

		finder := zorm.NewFinder()
		finder = finder.Append(sql, fmt.Sprintf("NewDescs-%d", time.Now().UnixMilli()), insertID)
		affected, err := zorm.UpdateFinder(ctx, finder)
		if err != nil {
			fmt.Printf("update failed, err:%v\n", err)
			return nil, err
		}
		fmt.Printf("update success： %d.\n", affected)

		return nil, err
	})
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		panic(err)
	}
	return nil
}

func deleteData(insertID int64) error {

	_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {

		finder := zorm.NewFinder()
		finder = finder.Append("delete from order_tbl where id=?", insertID)
		affected, err := zorm.UpdateFinder(ctx, finder)
		if err != nil {
			fmt.Printf("delete failed, err:%v\n", err)
			return nil, err
		}

		fmt.Printf("deletesuccess: %d.\n", affected)
		return nil, err
	})
	if err != nil {
		fmt.Printf("delete failed, err:%v\n", err)
		panic(err)
	}

	return nil
}

func batchDeleteData(userIds []string) error {
	var rows int

	_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {

		finder := zorm.NewFinder()
		finder.Append("delete from order_tbl where user_id in (?)", userIds)

		var err error
		rows, err = zorm.UpdateFinder(ctx, finder)
		return nil, err
	})
	if err != nil {
		fmt.Printf("batch delete failed, err:%v\n", err)
		return err
	}

	fmt.Printf("batch delete success: %d.\n", rows)
	return nil
}
