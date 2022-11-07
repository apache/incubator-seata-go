package parser

import (
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	filter2 "github.com/seata/seata-go/pkg/datasource/sql/undo/parser/filter"
)

type FastJsonUndoLogParser struct {
	baseParser BaseParser
}

func (u *FastJsonUndoLogParser) GetName() string {
	return ""
}

func (u *FastJsonUndoLogParser) GetDefaultContent() []byte {

	return nil
}

func (u *FastJsonUndoLogParser) Encode(l undo.BranchUndoLog) []byte {
	filter := filter2.SelectMarshal("tableMeta", l)
	json := filter.MustJSON()
	return []byte(json)
}

func (u *FastJsonUndoLogParser) Decode(bytes []byte) undo.BranchUndoLog {

	return undo.BranchUndoLog{}
}
