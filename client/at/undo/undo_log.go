package undo

import (
	_struct "github.com/xiaobudongzhang/seata-golang/client/at/sql/struct"
	"github.com/xiaobudongzhang/seata-golang/client/at/sqlparser"
)

type SqlUndoLog struct {
	SqlType     sqlparser.SQLType
	TableName   string
	BeforeImage *_struct.TableRecords
	AfterImage  *_struct.TableRecords
}

func (undoLog *SqlUndoLog) SetTableMeta(tableMeta _struct.TableMeta) {
	if undoLog.BeforeImage != nil {
		undoLog.BeforeImage.TableMeta = tableMeta
	}
	if undoLog.AfterImage != nil {
		undoLog.AfterImage.TableMeta = tableMeta
	}
}

type BranchUndoLog struct {
	Xid         string
	BranchId    int64
	SqlUndoLogs []*SqlUndoLog
}
