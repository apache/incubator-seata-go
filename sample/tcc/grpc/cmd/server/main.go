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
	"fmt"
	"net"

	"github.com/seata/seata-go/pkg/rm/tcc"

	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	server2 "github.com/seata/seata-go/pkg/integration/grpc/server"
	"github.com/seata/seata-go/sample/tcc/grpc/pb"
	"github.com/seata/seata-go/sample/tcc/grpc/service"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("server register")
	s := grpc.NewServer(grpc.UnaryInterceptor(server2.ServerTransactionInterceptor))

	tccProxy1, err := tcc.NewTCCServiceProxy(service.BusinessServer11)
	if err != nil {
		log.Errorf("get userProviderProxy tcc service proxy error, %v", err.Error())
		return
	}
	err = tccProxy1.RegisterResource()
	if err != nil {
		log.Errorf("userProviderProxy register resource error, %v", err.Error())
		return
	}

	tccProxy2, err := tcc.NewTCCServiceProxy(service.BusinessServer22)
	if err != nil {
		log.Errorf("get userProviderProxy tcc service proxy error, %v", err.Error())
		return
	}
	err = tccProxy2.RegisterResource()
	if err != nil {
		log.Errorf("userProviderProxy register resource error, %v", err.Error())
		return
	}

	pb.RegisterTCCServiceBusiness1Server(s, service.BusinessServer11)
	pb.RegisterTCCServiceBusiness2Server(s, service.BusinessServer22)

	log.Infof("business listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
