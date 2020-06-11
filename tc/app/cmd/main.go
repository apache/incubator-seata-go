package main

import (
	_ "net/http/pprof"
)

import (
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/holder"
	"github.com/dk-lockdown/seata-golang/tc/lock"
	_ "github.com/dk-lockdown/seata-golang/tc/metrics"
	"github.com/dk-lockdown/seata-golang/tc/server"
)

const (
	APP_CONF_FILE     = "/Users/scottlewis/dksl/git/1/seata-golang/tc/app/profiles/dev/config.yml"
)

func main() {
	config.InitConf(APP_CONF_FILE)
	uuid.Init(1)
	lock.Init()
	holder.Init()
	srv := server.NewServer()
	conf := config.GetServerConfig()
	srv.Start(conf.Host+":"+conf.Port)
}
