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

package constant

const (
	DeleteFrom                     = "DELETE FROM "
	InsertFrom                     = "INSERT INTO "
	DefaultTransactionUndoLogTable = " undo_log "
	// UndoLogTableName Todo get from config
	UndoLogTableName = DefaultTransactionUndoLogTable
	DeleteUndoLogSql = DeleteFrom + UndoLogTableName + " WHERE " + UndoLogBranchXid + " = ? AND " + UndoLogXid + " = ?"
	/**
	 * branch_id, xid, context, rollback_info, log_status, log_created, log_modified
	 */
	InsertUndoLogSql = "INSERT INTO " + DefaultTransactionUndoLogTable +
		" (" + UndoLogBranchXid + ", " + UndoLogXid + ", " + UndoLogContext + ", " +
		UndoLogRollBackInfo + ", " + UndoLogLogStatus + ", " + UndoLogLogCreated +
		", " + UndoLogLogModified + ")" + " VALUES (?, ?, ?, ?, ?, now(6), now(6))"
)

const ErrCodeTableNotExist = "1146"
