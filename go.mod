module github.com/dk-lockdown/seata-golang

go 1.14

require (
	github.com/Davmuz/gqt v0.0.0-20161229104334-7589d282c7c3
	github.com/bwmarrin/snowflake v0.3.0
	github.com/denisenkom/go-mssqldb v0.0.0-20191124224453-732737034ffd // indirect
	github.com/dubbogo/getty v1.3.7
	github.com/dubbogo/gost v1.9.0
	github.com/gin-gonic/gin v1.6.2
	github.com/go-playground/assert/v2 v2.0.1
	github.com/go-sql-driver/mysql v1.4.1
	github.com/go-xorm/xorm v0.7.9
	github.com/golang/protobuf v1.3.4
	github.com/google/go-cmp v0.2.0
	github.com/juju/testing v0.0.0-20200608005635-e4eedbc6f7aa // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.1.1 // indirect
	github.com/mattn/go-sqlite3 v2.0.1+incompatible // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pingcap/check v0.0.0-20200212061837-5e12011dc712 // indirect
	github.com/pingcap/errors v0.11.5-0.20190809092503-95897b64e011 // indirect
	github.com/pingcap/log v0.0.0-20200117041106-d28c14d3b1cd // indirect
	github.com/pingcap/parser v0.0.0-20200424075042-8222d8b724a4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/stretchr/testify v1.5.1
	github.com/urfave/cli/v2 v2.2.0
	go.uber.org/atomic v1.6.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd // indirect
	golang.org/x/tools v0.0.0-20200325203130-f53864d0dba1 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.8
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
	vimagination.zapto.org/byteio v0.0.0-20200222190125-d27cba0f0b10
	vimagination.zapto.org/memio v0.0.0-20200222190306-588ebc67b97d // indirect
	xorm.io/builder v0.3.6
)

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.4

replace go.etcd.io/bbolt => github.com/coreos/bbolt v1.3.4
