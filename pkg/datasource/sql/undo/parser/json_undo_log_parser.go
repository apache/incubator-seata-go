package parser

import (
	"encoding/json"

	"github.com/seata/seata-go/pkg/datasource/sql/undo"
)

type JsonUndoLogParser struct {
}

func (j JsonUndoLogParser) GetName() string {
	return "fastjson"
}

func (j JsonUndoLogParser) GetDefaultContent() []byte {
	return []byte("{}")
}

func (j JsonUndoLogParser) Encode(l undo.BranchUndoLog) []byte {
	b, _ := json.Marshal(l)
	return b
}

func (j JsonUndoLogParser) Decode(b []byte) undo.BranchUndoLog {
	var u undo.BranchUndoLog
	json.Unmarshal(b, &u)
	return u
}
