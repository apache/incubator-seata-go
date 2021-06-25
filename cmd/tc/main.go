package main

import (
	"fmt"
	"net"
	"os"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/config"
	_ "github.com/opentrx/seata-golang/v2/pkg/tc/metrics"
	"github.com/opentrx/seata-golang/v2/pkg/tc/server"
	_ "github.com/opentrx/seata-golang/v2/pkg/tc/storage/driver/mysql"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
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
					config, err := resolveConfiguration(c.Args().Slice())
					log.Init(config.Log.LogPath, config.Log.LogLevel)

					address := fmt.Sprintf(":%v", config.Server.Port)
					lis, err := net.Listen("tcp", address)
					if err != nil {
						log.Fatalf("failed to listen: %v", err)
					}

					s := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(config.GetEnforcementPolicy()),
						grpc.KeepaliveParams(config.GetServerParameters()))

					tc := server.NewTransactionCoordinator(config)
					apis.RegisterTransactionManagerServiceServer(s, tc)
					apis.RegisterResourceManagerServiceServer(s, tc)

					if err := s.Serve(lis); err != nil {
						log.Fatalf("failed to serve: %v", err)
					}
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

func resolveConfiguration(args []string) (*config.Configuration, error) {
	var configurationPath string

	if len(args) > 0 {
		configurationPath = args[0]
	} else if os.Getenv("SEATA_CONFIGURATION_PATH") != "" {
		configurationPath = os.Getenv("SEATA_CONFIGURATION_PATH")
	}

	if configurationPath == "" {
		return nil, fmt.Errorf("configuration path unspecified")
	}

	fp, err := os.Open(configurationPath)
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	config, err := config.Parse(fp)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", configurationPath, err)
	}

	return config, nil
}
