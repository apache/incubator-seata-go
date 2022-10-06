package parser

import "github.com/seata/seata-go/pkg/datasource/sql/undo"

type MysqlUndoLogParser struct {
	baseParser BaseParser
}

func (u *MysqlUndoLogParser) GetName() string {
	return ""
}

func (u *MysqlUndoLogParser) GetDefaultContent() []byte {

	return nil
}

func (u *MysqlUndoLogParser) Encode(l undo.BranchUndoLog) []byte {
	return nil
}

func (u *MysqlUndoLogParser) Decode(bytes []byte) undo.BranchUndoLog {
	return undo.BranchUndoLog{}
}
