package parser

import (
	"bytes"
	"encoding/json"
	"github.com/dk-lockdown/seata-golang/client/rm_datasource/undo"
	"github.com/pkg/errors"
)

type JsonUndoLogParser struct {

}

func (parser JsonUndoLogParser) GetName() string {
	return "json"
}

// return the default content if undo log is empty
func (parser JsonUndoLogParser) GetDefaultContent() []byte {
	return []byte("{}")
}

func (parser JsonUndoLogParser) Encode(branchUndoLog undo.BranchUndoLog) []byte {
	data,err := json.Marshal(branchUndoLog)
	if err != nil {
		panic(errors.Errorf("BranchUndoLog encoded error:%v",branchUndoLog))
	}
	return data
}

func (parser JsonUndoLogParser) Decode(data []byte) undo.BranchUndoLog {
	var undoLog undo.BranchUndoLog

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&undoLog); err != nil {
		panic(errors.Errorf("BranchUndoLog decoded error:%v",data))
	}

	return undoLog
}
