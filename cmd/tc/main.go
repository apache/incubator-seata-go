package main

import (
	"fmt"
	"os"
	"strconv"
)

import (
	gxnet "github.com/dubbogo/gost/net"
	"github.com/urfave/cli/v2"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common"
	_ "github.com/transaction-wg/seata-golang/pkg/base/config_center/nacos"
	_ "github.com/transaction-wg/seata-golang/pkg/base/registry/nacos"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/tc/holder"
	"github.com/transaction-wg/seata-golang/pkg/tc/lock"
	_ "github.com/transaction-wg/seata-golang/pkg/tc/metrics"
	"github.com/transaction-wg/seata-golang/pkg/tc/server"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/uuid"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "start seata golang tc server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config, c",
						Usage: "Load configuration from `FILE`",
					},
					&cli.StringFlag{
						Name:  "serverNode, n",
						Value: "1",
						Usage: "server node id, such as 1, 2, 3. default is 1",
					},
				},
				Action: func(c *cli.Context) error {
					configPath := c.String("config")
					serverNode := c.Int64("serverNode")

					conf, err := config.InitConf(configPath)
					if err != nil {
						log.Fatal(err)
					}

					ip, _ := gxnet.GetLocalIP()
					port, err := strconv.Atoi(conf.Port)
					if err != nil {
						log.Fatal(err)
					}

					common.Init(ip, port)
					uuid.Init(serverNode)
					lock.Init()
					holder.Init()

					srv := server.NewServer()
					srv.Start(fmt.Sprintf(":%s", conf.Port))
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err)
	}
}
