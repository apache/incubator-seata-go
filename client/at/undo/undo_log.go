package undo

import (
	"github.com/dk-lockdown/seata-golang/client/at/sql/schema"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
)

type SqlUndoLog struct {
	SqlType     sqlparser.SQLType
	TableName   string
	BeforeImage *schema.TableRecords
	AfterImage  *schema.TableRecords
}

func (undoLog *SqlUndoLog) SetTableMeta(tableMeta schema.TableMeta) {
	if undoLog.BeforeImage != nil {
		undoLog.BeforeImage.TableMeta = tableMeta
	}
	if undoLog.AfterImage != nil {
		undoLog.AfterImage.TableMeta = tableMeta
	}
}

func (undoLog *SqlUndoLog) GetUndoRows() *schema.TableRecords {
	if undoLog.SqlType == sqlparser.SQLType_UPDATE ||
		undoLog.SqlType == sqlparser.SQLType_DELETE {
		return undoLog.BeforeImage
	} else if undoLog.SqlType == sqlparser.SQLType_INSERT {
		return undoLog.AfterImage
	}
	return nil
}

type BranchUndoLog struct {
	Xid         string
	BranchId    int64
	SqlUndoLogs []*SqlUndoLog
}
