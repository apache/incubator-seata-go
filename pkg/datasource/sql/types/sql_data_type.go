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

package types

import "strings"

var SqlDataTypes = map[string]int32{
	"BIT":                     -7,
	"TINYINT":                 -6,
	"SMALLINT":                5,
	"INTEGER":                 4,
	"BIGINT":                  -5,
	"FLOAT":                   6,
	"REAL":                    7,
	"DOUBLE":                  8,
	"NUMERIC":                 2,
	"DECIMAL":                 3,
	"CHAR":                    1,
	"VARCHAR":                 12,
	"LONGVARCHAR":             -1,
	"DATE":                    91,
	"TIME":                    92,
	"TIMESTAMP":               93,
	"BINARY":                  -2,
	"VARBINARY":               -3,
	"LONGVARBINARY":           -4,
	"NULL":                    0,
	"OTHER":                   1111,
	"JAVA_OBJECT":             2000,
	"DISTINCT":                2001,
	"STRUCT":                  2002,
	"ARRAY":                   2003,
	"BLOB":                    2004,
	"CLOB":                    2005,
	"REF":                     2006,
	"DATALINK":                70,
	"BOOLEAN":                 16,
	"ROWID":                   -8,
	"NCHAR":                   -15,
	"NVARCHAR":                -9,
	"LONGNVARCHAR":            -16,
	"NCLOB":                   2011,
	"SQLXML":                  2009,
	"REF_CURSOR":              2012,
	"TIME_WITH_TIMEZONE":      2013,
	"TIMESTAMP_WITH_TIMEZONE": 2014,
}

// JdbcTypeToMySQLType maps JDBC type codes to MySQL type bytes
var JdbcTypeToMySQLType = map[int32]byte{
	-7:   1,   // BIT -> mysql.TypeTiny
	-6:   1,   // TINYINT -> mysql.TypeTiny
	5:    2,   // SMALLINT -> mysql.TypeShort
	4:    3,   // INTEGER -> mysql.TypeLong
	-5:   8,   // BIGINT -> mysql.TypeLonglong
	6:    4,   // FLOAT -> mysql.TypeFloat
	7:    4,   // REAL -> mysql.TypeFloat
	8:    5,   // DOUBLE -> mysql.TypeDouble
	2:    246, // NUMERIC -> mysql.TypeNewDecimal
	3:    246, // DECIMAL -> mysql.TypeNewDecimal
	1:    254, // CHAR -> mysql.TypeString
	12:   15,  // VARCHAR -> mysql.TypeVarchar
	-1:   252, // LONGVARCHAR -> mysql.TypeBlob
	91:   10,  // DATE -> mysql.TypeDate
	92:   11,  // TIME -> mysql.TypeDuration
	93:   7,   // TIMESTAMP -> mysql.TypeTimestamp
	-2:   254, // BINARY -> mysql.TypeString
	-3:   15,  // VARBINARY -> mysql.TypeVarchar
	-4:   252, // LONGVARBINARY -> mysql.TypeBlob
	2004: 252, // BLOB -> mysql.TypeBlob
	2005: 252, // CLOB -> mysql.TypeBlob
	16:   1,   // BOOLEAN -> mysql.TypeTiny
	-15:  254, // NCHAR -> mysql.TypeString
	-9:   15,  // NVARCHAR -> mysql.TypeVarchar
	-16:  252, // LONGNVARCHAR -> mysql.TypeBlob
	2011: 252, // NCLOB -> mysql.TypeBlob
	2009: 245, // SQLXML -> mysql.TypeJSON
}

func GetSqlDataType(dataType string) int32 {
	return SqlDataTypes[strings.ToUpper(dataType)]
}

// ConvertJdbcTypeToMySQLType converts JDBC type code to MySQL type byte
func ConvertJdbcTypeToMySQLType(jdbcType int32) byte {
	if mysqlType, ok := JdbcTypeToMySQLType[jdbcType]; ok {
		return mysqlType
	}
	// Default to VARCHAR for unknown types
	return 15 // mysql.TypeVarchar
}
