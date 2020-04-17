package main

import (
	"os"
)

import (
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/server"
)

const (
	APP_CONF_FILE     = "APP_CONF_FILE"
)

func main() {
	confFile := os.Getenv(APP_CONF_FILE)
	config.InitConf(confFile)
	server.SetServerGrpool()
	srv := server.NewServer()
	conf := config.GetServerConfig()
	srv.Start(conf.Host+":"+conf.Port)
}
