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

type BranchUndoLog struct {
	Xid string
	BranchId int64
	SqlUndoLogs []*SqlUndoLog
}

