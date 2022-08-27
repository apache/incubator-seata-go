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
