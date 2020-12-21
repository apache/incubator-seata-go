package lock

import (
	"testing"
)

import (
	"github.com/go-xorm/xorm"
)

var dsn = "root:123456@tcp(127.0.0.1:3306)/seata2?timeout=1s&readTimeout=1s&writeTimeout=1s&parseTime=true&loc=Local&charset=utf8mb4,utf8"

func TestLockStoreDataBaseDao_UnLockByXidAndBranchIds(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		panic(err)
	}
	lockStore := &LockStoreDataBaseDao{engine: engine}

	lockStore.UnLockByXidAndBranchIds(":0:2000042936", []int64{2000042938})
}
