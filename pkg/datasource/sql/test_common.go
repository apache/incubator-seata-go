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
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type dbTestConfig struct {
	name             string
	dbType           types.DBType
	atDriverName     string
	xaDriverName     string
	mockXaDriverName string
	dsn              string
	versionQuery     string
	versionResult    [][]interface{}
}

func getAllDBTestConfigs() []dbTestConfig {
	return []dbTestConfig{
		{
			name:             "MySQL",
			dbType:           types.DBTypeMySQL,
			atDriverName:     SeataATMySQLDriver,
			xaDriverName:     SeataXAMySQLDriver,
			mockXaDriverName: "",
			dsn:              "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true",
			versionQuery:     "SELECT VERSION()",
			versionResult:    [][]interface{}{{"8.0.29"}},
		},
		{
			name:             "PostgreSQL",
			dbType:           types.DBTypePostgreSQL,
			atDriverName:     SeataATPostgreSQLDriver,
			xaDriverName:     SeataXAPostgreSQLDriver,
			mockXaDriverName: "mock-seata-xa-postgres",
			dsn:              "postgres://postgres:seata_go@127.0.0.1:5432/seata_go_test?sslmode=disable",
			versionQuery:     "SELECT version()",
			versionResult:    [][]interface{}{{"PostgresSQL 14.5"}},
		},
	}
}
