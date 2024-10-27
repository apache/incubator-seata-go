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
	"database/sql/driver"
	"fmt"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/util/log"
)

// CreateXAResource create a connection for xa with the different db type.
// Such as mysql, oracle, MARIADB, POSTGRESQL
func CreateXAResource(conn driver.Conn, dbType types.DBType) (XAResource, error) {
	var err error
	var xaConnection XAResource
	switch dbType {
	case types.DBTypeMySQL:
		xaConnection = NewMysqlXaConn(conn)
	case types.DBTypeOracle:
	case types.DBTypePostgreSQL:
	default:
		err = fmt.Errorf("not support db type for :%s", dbType.String())
	}

	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}

	return xaConnection, nil
}
