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
	"database/sql"
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/client"
	sql2 "github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/tm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type OrderTblModel struct {
	Id            int64  `gorm:"column:id" json:"id"`
	UserId        string `gorm:"column:user_id" json:"user_id"`
	CommodityCode string `gorm:"commodity_code" json:"commodity_code"`
	Count         int64  `gorm:"count" json:"count"`
	Money         int64  `gorm:"money" json:"money"`
	Descs         string `gorm:"descs" json:"descs"`
}

func main() {
	client.InitPath("./sample/conf/seatago.yml")
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, insertData)
	<-make(chan struct{})
}

func insertData(ctx context.Context) error {

	sqlDB, err := sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		panic("init service error")
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})

	data := OrderTblModel{
		UserId:        "NO-100002",
		CommodityCode: "C100001",
		Count:         101,
		Money:         11,
		Descs:         "insert desc",
	}

	gormDB = gormDB.Table("order_tbl").Create(&data)
	if gormDB.Error != nil {
		fmt.Println(gormDB.Error)
		return err
	}

	return nil
}
