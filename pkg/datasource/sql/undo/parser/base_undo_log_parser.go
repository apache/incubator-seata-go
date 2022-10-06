package parser

import (
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"sync"
)

var parser = &BaseParser{}

type BaseParser struct {
}

var (
	mu        sync.RWMutex
	factories = make(map[string]BaseParser)
)

func (u *BaseParser) GetName() string {
	return ""
}

func (u *BaseParser) GetDefaultContent() []byte {

	return nil
}

func (u *BaseParser) Encode(l undo.BranchUndoLog) []byte {
	return nil
}

func (u *BaseParser) Decode(bytes []byte) undo.BranchUndoLog {
	return undo.BranchUndoLog{}
}
