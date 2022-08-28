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
	// UndoLogId The constant undo_log column name xid, this field is not use in mysql
	UndoLogId string = "id"

	// UndoLogXid The constant undo_log column name xid
	UndoLogXid = "xid"

	// UndoLogBranchXid The constant undo_log column name branch_id
	UndoLogBranchXid = "branch_id"

	// UndoLogContext The constant undo_log column name context
	UndoLogContext = "context"

	// UndoLogRollBackInfo The constant undo_log column name rollback_info
	UndoLogRollBackInfo = "rollback_info"

	// UndoLogLogStatus The constant undo_log column name log_status
	UndoLogLogStatus = "log_status"

	// UndoLogLogCreated The constant undo_log column name log_created
	UndoLogLogCreated = "log_created"

	// UndoLogLogModified The constant undo_log column name log_modified
	UndoLogLogModified = "log_modified"
)
