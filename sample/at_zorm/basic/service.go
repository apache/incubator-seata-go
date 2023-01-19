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
	"log"

	"gitee.com/chunanyong/zorm"
	_ "github.com/go-sql-driver/mysql"
)

var (
	db *zorm.DBDao
)

type orderStruct struct {
	zorm.EntityStruct

	Id            int64  `column:"id"`
	UserId        string `column:"user_id"`
	CommodityCode string `column:"commodity_code"`
	Count         int64  `column:"count"`
	Money         int64  `column:"money"`
	Descs         string `column:"descs"`
}

func (entity *orderStruct) GetTableName() string {
	return "order_tbl"
}

func (entity *orderStruct) GetPKColumnName() string {
	return "id"
}

func initService() {
	dbConfig := zorm.DataSourceConfig{
		// db, err = sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(demo.wuxian.pro:3306)/seata_client?multiStatements=true&interpolateParams=true")

		// SQLDB 使用现有的数据库连接,优先级高于DSN
		// SQLDB: db,
		//连接数据库DSN
		DSN: "root:12345678@tcp(demo.wuxian.pro:3307)/seata_client?charset=utf8&parseTime=true", //MySQL
		//数据库类型
		//sql.Open(DriverName,DSN) DriverName就是驱动的sql.Open第一个字符串参数,根据驱动实际情况获取
		DriverName: "mysql",
		Dialect:    "mysql",
		//SlowSQLMillis 慢sql的时间阈值,单位毫秒.小于0是禁用SQL语句输出;等于0是只输出SQL语句,不计算执行时间;大于0是计算SQL执行时间,并且>=SlowSQLMillis值
		SlowSQLMillis: 0,
		//最大连接数 默认50
		MaxOpenConns: 0,
		//最大空闲数 默认50
		MaxIdleConns: 0,
		//连接存活秒时间. 默认600
		ConnMaxLifetimeSecond: 0,
		//事务隔离级别的默认配置,默认为nil
		DefaultTxOptions: nil,
	}

	var err error
	db, err = zorm.NewDBDao(&dbConfig)
	if err != nil {
		log.Fatalf("init mysql failure: %v", err)
		panic(err)
	}
	log.Println("init mysql success.")
}
