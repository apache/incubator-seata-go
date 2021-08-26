package client

import (
	"log"

	"google.golang.org/grpc"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/client/config"
	"github.com/opentrx/seata-golang/v2/pkg/client/rm"
	"github.com/opentrx/seata-golang/v2/pkg/client/tcc"
	"github.com/opentrx/seata-golang/v2/pkg/client/tm"
)

// Init init resource manager，init transaction manager, expose a port to listen tc
// call back request.
func Init(config *config.Configuration) {
	conn, err := grpc.Dial(config.ServerAddressing,
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(config.GetClientParameters()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	resourceManagerClient := apis.NewResourceManagerServiceClient(conn)
	transactionManagerClient := apis.NewTransactionManagerServiceClient(conn)

	rm.InitResourceManager(config.Addressing, resourceManagerClient)
	tm.InitTransactionManager(config.Addressing, transactionManagerClient)
	rm.RegisterTransactionServiceServer(tcc.GetTCCResourceManager())
}
