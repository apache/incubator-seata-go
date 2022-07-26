/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/seata/seata-go/pkg/integration/grpc/interceptor/server"

	"github.com/seata/seata-go/pkg/integration/grpc/interceptor/client"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/seata/seata-go/pkg/tm"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/integration/grpc/pb"
	"google.golang.org/grpc"
)

type ContextRpcTestServer struct {
	pb.UnimplementedContextRpcServer
}

func (c *ContextRpcTestServer) ContextRpc(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	log.Infof("receive the %s", req.Name)
	return &pb.Response{Greet: fmt.Sprintf("receive the name %s, xid is %s, return greet!", req.Name, tm.GetXID(ctx))}, nil
}

func TestClientHeaderDeliveredToServer(t *testing.T) {
	StartServer(t)
	StartClientWithCall(t)
}

// StartServer to start grpc server
func StartServer(t *testing.T) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		t.FailNow()
	}
	//inject server interceptor
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(server.ServerTransactionInterceptor))
	pb.RegisterContextRpcServer(grpcServer, &ContextRpcTestServer{})

	go func() {
		grpcServer.Serve(lis)
	}()
}

// StartClientWithCall to start grpc client and call server
func StartClientWithCall(t *testing.T) {

	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(client.ClientTransactionInterceptor)) //inject client interceptor
	if err != nil {
		log.Fatalf("dial to server: %v", err)
		t.FailNow()
	}
	defer conn.Close()
	contextRpcClient := pb.NewContextRpcClient(conn)
	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "111111")
	response, err := contextRpcClient.ContextRpc(ctx, &pb.Request{Name: "zhangsan"})

	if err != nil {
		log.Fatalf("call rpc : %v", err)
		t.FailNow()
	}
	// if success, the response msg  will contain the xid.
	log.Info(response.Greet)
	assert.Equal(t, "receive the name zhangsan, xid is 111111, return greet!", response.Greet)

}
