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

type OrderTbl struct {
	id            int
	userID        string
	commodityCode string
	count         int64
	money         int64
	descs         string
}

func main() {
	client.Init()
	initService()
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, updateData)
	<-make(chan struct{})
}

func updateData(ctx context.Context) error {
	sql := "update order_tbl set descs=? where id=?"
	ret, err := db.ExecContext(ctx, sql, fmt.Sprintf("NewDescs-%d", time.Now().UnixMilli()), 1)
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return err
	}
	fmt.Printf("update success： %d.\n", rows)
	return nil
}
