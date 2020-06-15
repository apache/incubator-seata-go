package parser

import "github.com/xiaobudongzhang/seata-golang/client/at/undo"

type UndoLogParser interface {
	GetName() string

	// return the default content if undo log is empty
	GetDefaultContent() []byte

	Encode(branchUndoLog *undo.BranchUndoLog) []byte

	Decode(data []byte) *undo.BranchUndoLog
}

func GetUndoLogParser() UndoLogParser {
	return ProtoBufUndoLogParser{}
}
