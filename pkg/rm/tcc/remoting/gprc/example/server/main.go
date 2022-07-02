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

// Package main implements a business for Greeter service.
package main

import (
	"context"
	"fmt"
	"net"

	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	server2 "github.com/seata/seata-go/pkg/integration/grpc/server"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/rm/tcc/remoting/gprc/example"
	"github.com/seata/seata-go/pkg/tm"
	"google.golang.org/protobuf/types/known/emptypb"

	"google.golang.org/grpc"
)

type business struct {
	example.UnimplementedTCCServiceBusinessServer
}

func (b *business) Prepare(ctx context.Context, params interface{}) error {
	log.Infof("TestTCCServiceBusiness Prepare, param %v", params)
	return nil
}

func (b *business) Remoting(ctx context.Context, params *example.Params) (*emptypb.Empty, error) {
	log.Infof("Remoting be called")
	return new(emptypb.Empty), tcc.NewTCCServiceProxy(b).Prepare(ctx, params)
}

func (b *business) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Commit, param %v", businessActionContext)
	return nil
}

func (b *business) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Rollback, param %v", businessActionContext)
	return nil
}

func (b *business) GetActionName() string {
	return "TCCServiceBusiness"
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("server register")
	s := grpc.NewServer(grpc.UnaryInterceptor(server2.ServerTransactionInterceptor))
	example.RegisterTCCServiceBusinessServer(s, &business{})
	log.Infof("business listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
