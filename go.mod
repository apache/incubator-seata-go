module github.com/transaction-wg/seata-golang

go 1.14

require (
	github.com/apache/dubbo-getty v1.4.3
	github.com/creasty/defaults v1.5.1
	github.com/denisenkom/go-mssqldb v0.0.0-20191124224453-732737034ffd // indirect
	github.com/dubbogo/gost v1.11.11
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-xorm/xorm v0.7.9
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.5
	github.com/imdario/mergo v0.3.12
	github.com/lib/pq v1.1.1 // indirect
	github.com/mattn/go-sqlite3 v2.0.1+incompatible // indirect
	github.com/nacos-group/nacos-sdk-go v1.0.8
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pingcap/check v0.0.0-20200212061837-5e12011dc712 // indirect
	github.com/pingcap/errors v0.11.5-0.20190809092503-95897b64e011 // indirect
	github.com/pingcap/log v0.0.0-20200117041106-d28c14d3b1cd // indirect
	github.com/pingcap/parser v0.0.0-20200424075042-8222d8b724a4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/shima-park/agollo v1.2.6
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli/v2 v2.3.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.17.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	google.golang.org/grpc v1.38.0
	gopkg.in/yaml.v2 v2.4.0
	vimagination.zapto.org/byteio v0.0.0-20200222190125-d27cba0f0b10
	vimagination.zapto.org/memio v0.0.0-20200222190306-588ebc67b97d // indirect
	xorm.io/builder v0.3.9
)

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.4

replace go.etcd.io/bbolt => github.com/coreos/bbolt v1.3.4

replace github.com/apache/dubbo-getty => github.com/dubbogo/getty v1.4.3
