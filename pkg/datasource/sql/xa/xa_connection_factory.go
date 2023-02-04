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

package xa

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/seata/seata-go/pkg/util/log"
)

const (
	DbTypeOracle  = "oracle"
	DbTypeMysql   = "mysql"
	DbTypeDB2     = "db2"
	DbTypeMariadb = "mariadb"
	DbTypePG      = "postgresql"
)

// CreateXAConnection create a connection for xa with the different db type.
// Such as mysql, oracle, MARIADB, POSTGRESQL
func CreateXAConnection(ctx context.Context, conn *sql.Conn, dbType string) (XAConnection, error) {
	var err error
	var xaConnection XAConnection
	switch dbType {
	case DbTypeMysql:
		var mysqlXaConn MysqlXAConn
		xaConnection, err = mysqlXaConn.GetXAResource()
	case DbTypeOracle:
	case DbTypeMariadb:
	case DbTypePG:
	case DbTypeDB2:
	default:
		err = fmt.Errorf("not support db type for :%s", dbType)
	}

	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}

	return xaConnection, nil
}
