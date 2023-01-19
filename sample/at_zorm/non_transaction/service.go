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
)

var db *zorm.DBDao

func initService() {
	dbConfig := zorm.DataSourceConfig{
		//连接数据库DSN
		// db, err = sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(demo.wuxian.pro:3306)/seata_client?multiStatements=true&interpolateParams=true")
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
		log.Fatalf("数据库连接异常 %v", err)
		panic(err)
	}
	log.Println("数据库连接成功")
}
