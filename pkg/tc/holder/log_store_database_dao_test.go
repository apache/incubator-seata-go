package holder

import (
	"testing"
)

import (
	"github.com/go-playground/assert/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/tc/model"
)

var dsn = "root:123456@tcp(127.0.0.1:3306)/seata2?timeout=1s&readTimeout=1s&writeTimeout=1s&parseTime=true&loc=Local&charset=utf8mb4,utf8"

func TestLogStoreDataBaseDAO_InsertGlobalTransactionDO(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		panic(err)
	}
	logStore := &LogStoreDataBaseDAO{engine: engine}

	globalTransactionDO := model.GlobalTransactionDO{
		XID:                     ":0:2000042921",
		TransactionID:           2000042921,
		Status:                  1,
		ApplicationID:           "order_aggregation_service",
		TransactionServiceGroup: "order_aggregation_service_group",
		TransactionName:         "createSo(boolean)",
		Timeout:                 60000,
		BeginTime:               1589192346991,
		ApplicationData:         nil,
	}
	logStore.InsertGlobalTransactionDO(globalTransactionDO)
}

func TestLogStoreDataBaseDAO_QueryGlobalTransactionDOByTransactionID(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		panic(err)
	}
	logStore := &LogStoreDataBaseDAO{engine: engine}

	globalTransactionDO := logStore.QueryGlobalTransactionDOByTransactionID(2000042921)
	assert.Equal(t, globalTransactionDO.TransactionID, int64(2000042921))
}

func TestLogStoreDataBaseDAO_QueryGlobalTransactionDOByStatuses(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		panic(err)
	}
	logStore := &LogStoreDataBaseDAO{engine: engine}

	globalTransactionDOs := logStore.QueryGlobalTransactionDOByStatuses([]int{1}, 100)

	assert.Equal(t, globalTransactionDOs[0].TransactionID, int64(2000042921))
}
