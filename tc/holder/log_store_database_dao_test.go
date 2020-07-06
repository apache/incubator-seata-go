package holder

import (
	"github.com/Davmuz/gqt"
	"github.com/dk-lockdown/seata-golang/tc/model"
	"github.com/go-playground/assert/v2"
	"testing"
)

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

var dsn = "root:123456@tcp(127.0.0.1:3306)/seata2?timeout=1s&readTimeout=1s&writeTimeout=1s&parseTime=true&loc=Local&charset=utf8mb4,utf8"
var gqtPath = "/Users/scottlewis/dksl/git/1/seata-golang/tc/app/profiles/sql"

func TestLogStoreDataBaseDAO_InsertGlobalTransactionDO(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		panic(err)
	}
	gqt.Add(gqtPath, "*.sql")
	logStore := &LogStoreDataBaseDAO{engine: engine}

	globalTransactionDO := model.GlobalTransactionDO{
		Xid:                     ":0:2000042921",
		TransactionId:           2000042921,
		Status:                  1,
		ApplicationId:           "order_aggregation_service",
		TransactionServiceGroup: "order_aggregation_service_group",
		TransactionName:         "createSo(boolean)",
		Timeout:                 60000,
		BeginTime:               1589192346991,
		ApplicationData:         nil,
	}
	logStore.InsertGlobalTransactionDO(globalTransactionDO)
}

func TestLogStoreDataBaseDAO_QueryGlobalTransactionDOByTransactionId(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		panic(err)
	}
	gqt.Add(gqtPath, "*.sql")
	logStore := &LogStoreDataBaseDAO{engine: engine}

	globalTransactionDO := logStore.QueryGlobalTransactionDOByTransactionId(2000042921)
	assert.Equal(t, globalTransactionDO.TransactionId, int64(2000042921))
}

func TestLogStoreDataBaseDAO_QueryGlobalTransactionDOByStatuses(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		panic(err)
	}
	gqt.Add(gqtPath, "*.sql")
	logStore := &LogStoreDataBaseDAO{engine: engine}

	globalTransactionDOs := logStore.QueryGlobalTransactionDOByStatuses([]int{1}, 100)

	assert.Equal(t, globalTransactionDOs[0].TransactionId, int64(2000042921))
}
