package main

import (
	"net/http"
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
	srv := server.NewServer()
	conf := config.GetServerConfig()
	go http.ListenAndServe("0.0.0.0:6060", nil)
	srv.Start(conf.Host+":"+conf.Port)
}
