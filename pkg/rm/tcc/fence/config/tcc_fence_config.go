package config

import (
	"database/sql/driver"
	"math"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/dao"

	"go.uber.org/atomic"
)

var (
	schedulerDone sync.WaitGroup
)

const (
	MaxPeriod = time.Duration(math.MaxInt32)
)

type TccFenceConfig struct {
	Initialized        atomic.Bool      `default:"false"`
	LogTableName       string           `default:"tcc_fence_log"`
	Datasource         driver.Connector `default:"nil"`
	TransactionManager interface{}      `default:"nil"`
}

func SetTccFenceConfig(datasource driver.Connector, transactionManager interface{}) {

}

func init() {
	// todo
}

func (this *TccFenceConfig) Init() {
	// set log table name
	if this.LogTableName != "" {
		dao.GetTccFenceStoreDatabaseMapperSingleton().SetLogTableName(this.LogTableName)
	}

	if this.Datasource != nil {
		//handler.SetDatasource(this.Datasource)
	} else {
		panic("datasource need injected")
	}

	if this.TransactionManager != nil {
		// set transaction template
		//handler.SetTransactionManager(this.TransactionManager)
	} else {
		panic("transaction manager  need injected")
	}
}

func TccFenceCleanScheduler() {

}

func InitCleanTask() {
	schedulerDone.Add(1)
}

func Destory() {
	schedulerDone.Done()
}
