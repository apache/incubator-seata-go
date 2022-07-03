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

//go:generate stringer -type=SQLType
type SQLType int32

const (
	_ SQLType = iota
	SQLType_Unknown
	SQLType_SELECT
	SQLType_INSERT
	SQLType_UPDATE
	SQLType_DELETE
	SQLType_SELECT_FOR_UPDATE
	SQLType_REPLACE
	SQLType_TRUNCATE
	SQLType_CREATE
	SQLType_DROP
	SQLType_LOAD
	SQLType_MERGE
	SQLType_SHOW
	SQLType_ALTER
	SQLType_RENAME
	SQLType_DUMP
	SQLType_DEBUG
	SQLType_EXPLAIN
	SQLType_PROCEDURE
	SQLType_DESC
	SQLType_SET
	SQLType_RELOAD
	SQLType_SELECT_UNION
	SQLType_CREATE_TABLE
	SQLType_DROP_TABLE
	SQLType_ALTER_TABLE
	SQLType_SELECT_FROM_UPDATE
	SQLType_MULTI_DELETE
	SQLType_MULTI_UPDATE
	SQLType_CREATE_INDEX
	SQLType_DROP_INDEX
)
