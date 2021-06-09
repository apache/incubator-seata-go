package extension

import "sync"
import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/undo/manager"
)

var (
	undoLogManagersMu sync.RWMutex
	undoLogManagers   = make(map[string]func() manager.UndoLogManager)
)

//利用函数传递，可以延迟调用，用到的时候在进行实例化
func SetUndoLogManager(name string, v func() manager.UndoLogManager) {
	undoLogManagersMu.Lock()
	defer undoLogManagersMu.Unlock()
	if _, dup := undoLogManagers[name]; dup {
		panic("called twice for UndoLogManager " + name)
	}
	undoLogManagers[name] = v
}

func GetUndoLogManager(name string) manager.UndoLogManager {
	undoLogManagersMu.RLock()
	undoLogManager := undoLogManagers[name]
	undoLogManagersMu.RUnlock()
	return undoLogManager()
}
