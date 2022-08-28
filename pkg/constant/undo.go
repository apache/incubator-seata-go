package constant

const (
	DeleteFrom                     = "DELETE FROM "
	DefaultTransactionUndoLogTable = " undo_log "
	// UndoLogTableName Todo get from config
	UndoLogTableName = DefaultTransactionUndoLogTable
	DeleteUndoLogSql = DeleteFrom + UndoLogTableName + " WHERE " + UndoLogBranchXid + " = ? AND " + UndoLogXid + " = ?"
)
