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

import (
	"database/sql"
	"reflect"
)

// https://dev.mysql.com/doc/internals/en/com-query-response.html#packet-Protocol::ColumnType
type FieldType byte

const (
	FieldTypeDecimal FieldType = iota
	FieldTypeTiny
	FieldTypeShort
	FieldTypeLong
	FieldTypeFloat
	FieldTypeDouble
	FieldTypeNULL
	FieldTypeTimestamp
	FieldTypeLongLong
	FieldTypeInt24
	FieldTypeDate
	FieldTypeTime
	FieldTypeDateTime
	FieldTypeYear
	FieldTypeNewDate
	FieldTypeVarChar
	FieldTypeBit
)

const (
	FieldTypeJSON FieldType = iota + 0xf5
	FieldTypeNewDecimal
	FieldTypeEnum
	FieldTypeSet
	FieldTypeTinyBLOB
	FieldTypeMediumBLOB
	FieldTypeLongBLOB
	FieldTypeBLOB
	FieldTypeVarString
	FieldTypeString
	FieldTypeGeometry
)

type nullTime = sql.NullTime

var (
	ScanTypeFloat32   = reflect.TypeOf(float32(0))
	ScanTypeFloat64   = reflect.TypeOf(float64(0))
	ScanTypeInt8      = reflect.TypeOf(int8(0))
	ScanTypeInt16     = reflect.TypeOf(int16(0))
	ScanTypeInt32     = reflect.TypeOf(int32(0))
	ScanTypeInt64     = reflect.TypeOf(int64(0))
	ScanTypeNullFloat = reflect.TypeOf(sql.NullFloat64{})
	ScanTypeNullInt   = reflect.TypeOf(sql.NullInt64{})
	ScanTypeNullTime  = reflect.TypeOf(nullTime{})
	ScanTypeUint8     = reflect.TypeOf(uint8(0))
	ScanTypeUint16    = reflect.TypeOf(uint16(0))
	ScanTypeUint32    = reflect.TypeOf(uint32(0))
	ScanTypeUint64    = reflect.TypeOf(uint64(0))
	ScanTypeRawBytes  = reflect.TypeOf(sql.RawBytes{})
	ScanTypeUnknown   = reflect.TypeOf(new(interface{}))
)

// JDBCType's source is seata java: java.sql.Types.java
// it used in  undo_log.rollback_info.sqlUndoLogs.afterImage.rows.fields.type field
type JDBCType int16

const (
	JDBCTypeBit                   JDBCType = -7
	JDBCTypeTinyInt               JDBCType = -6
	JDBCTypeSmallInt              JDBCType = 5
	JDBCTypeInteger               JDBCType = 4
	JDBCTypeBigInt                JDBCType = -5
	JDBCTypeFloat                 JDBCType = 6
	JDBCTypeReal                  JDBCType = 7
	JDBCTypeDouble                JDBCType = 8
	JDBCTypeNumberic              JDBCType = 2
	JDBCTypeDecimal               JDBCType = 3
	JDBCTypeChar                  JDBCType = 1
	JDBCTypeVarchar               JDBCType = 12
	JDBCTypeLongVarchar           JDBCType = -1
	JDBCTypeDate                  JDBCType = 91
	JDBCTypeTime                  JDBCType = 92
	JDBCTypeTimestamp             JDBCType = 93
	JDBCTypeBinary                JDBCType = -2
	JDBCTypeVarBinary             JDBCType = -3
	JDBCTypeLongVarBinary         JDBCType = -4
	JDBCTypeNull                  JDBCType = 0
	JDBCTypeOther                 JDBCType = 1111
	JDBCTypeJavaObject            JDBCType = 2000
	JDBCTypeDistinct              JDBCType = 2001
	JDBCTypeStruct                JDBCType = 2002
	JDBCTypeArray                 JDBCType = 2003
	JDBCTypeBlob                  JDBCType = 2004
	JDBCTypeClob                  JDBCType = 2005
	JDBCTypeRef                   JDBCType = 2006
	JDBCTypeDateLink              JDBCType = 70
	JDBCTypeBoolean               JDBCType = 16
	JDBCTypeRowID                 JDBCType = -8
	JDBCTypeNchar                 JDBCType = -15
	JDBCTypeNvarchar              JDBCType = -9
	JDBCTypeLongNvVarchar         JDBCType = -16
	JDBCTypeNclob                 JDBCType = 2011
	JDBCTypeSqlXML                JDBCType = 2009
	JDBCTypeRefCursor             JDBCType = 2012
	JDBCTypeTimeWithTimeZone      JDBCType = 2013
	JDBCTypeTimestampWithTimezone JDBCType = 2014
)

// todo perfect all type name
func GetJDBCTypeByTypeName(typeName string) JDBCType {
	switch typeName {
	case "BIT":
		return JDBCTypeBit
	case "TEXT":
		return JDBCTypeLongVarchar
	case "BLOB":
		return JDBCTypeBlob
	case "DATE":
		return JDBCTypeDate
	case "DATETIME":
		return JDBCTypeTimestamp
	case "DECIMAL":
		return JDBCTypeDecimal
	case "DOUBLE":
		return JDBCTypeDouble
	case "ENUM":
		return JDBCTypeTinyInt
		// todo 待完善
	//case fieldTypeEnum:
	//	return "ENUM"
	//case fieldTypeFloat:
	//	return "FLOAT"
	//case fieldTypeGeometry:
	//	return "GEOMETRY"
	//case fieldTypeInt24:
	//	return "MEDIUMINT"
	//case fieldTypeJSON:
	//	return "JSON"
	//case fieldTypeLong:
	//	return "INT"
	//case fieldTypeLongBLOB:
	//	if mf.charSet != collations[binaryCollation] {
	//		return "LONGTEXT"
	//	}
	//	return "LONGBLOB"
	//case fieldTypeLongLong:
	//	return "BIGINT"
	//case fieldTypeMediumBLOB:
	//	if mf.charSet != collations[binaryCollation] {
	//		return "MEDIUMTEXT"
	//	}
	//	return "MEDIUMBLOB"
	//case fieldTypeNewDate:
	//	return "DATE"
	//case fieldTypeNewDecimal:
	//	return "DECIMAL"
	//case fieldTypeNULL:
	//	return "NULL"
	//case fieldTypeSet:
	//	return "SET"
	//case fieldTypeShort:
	//	return "SMALLINT"
	//case fieldTypeString:
	//	if mf.charSet == collations[binaryCollation] {
	//		return "BINARY"
	//	}
	//	return "CHAR"
	//case fieldTypeTime:
	//	return "TIME"
	//case fieldTypeTimestamp:
	//	return "TIMESTAMP"
	//case fieldTypeTiny:
	//	return "TINYINT"
	//case fieldTypeTinyBLOB:
	//	if mf.charSet != collations[binaryCollation] {
	//		return "TINYTEXT"
	//	}
	//	return "TINYBLOB"
	//case fieldTypeVarChar:
	//	if mf.charSet == collations[binaryCollation] {
	//		return "VARBINARY"
	//	}
	//	return "VARCHAR"
	//case fieldTypeVarString:
	//	if mf.charSet == collations[binaryCollation] {
	//		return "VARBINARY"
	//	}
	//	return "VARCHAR"
	//case fieldTypeYear:
	//	return "YEAR"
	default:
		return -1
	}
}
