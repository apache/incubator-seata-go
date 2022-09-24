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

func GetSqlDataType(dataType string) int32 {
	return SqlDataTypes[strings.ToUpper(dataType)]
}
