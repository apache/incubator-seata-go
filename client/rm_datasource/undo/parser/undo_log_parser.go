package parser

import "github.com/dk-lockdown/seata-golang/client/rm_datasource/undo"

type IUndoLogParser interface {
	GetName() string

	// return the default content if undo log is empty
	GetDefaultContent() []byte

	Encode(branchUndoLog *undo.BranchUndoLog) []byte

	Decode(data []byte) *undo.BranchUndoLog
}

func GetUndoLogParser() IUndoLogParser {
	return ProtoBufUndoLogParser{}
}