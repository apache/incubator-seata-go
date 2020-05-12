package main

import (
	"github.com/dk-lockdown/seata-golang/tc/holder"
	"github.com/dk-lockdown/seata-golang/tc/lock"
	"os"
)

import (
	_ "net/http/pprof"
)

import (
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/server"
)

const (
	APP_CONF_FILE     = "APP_CONF_FILE"
)

func main() {
	confFile := os.Getenv(APP_CONF_FILE)
	config.InitConf(confFile)
	uuid.Init(1)
	lock.Init()
	holder.Init()
	srv := server.NewServer()
	conf := config.GetServerConfig()
	srv.Start(conf.Host+":"+conf.Port)
}
