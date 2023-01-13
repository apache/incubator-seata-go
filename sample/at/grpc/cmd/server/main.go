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
	"github.com/seata/seata-go/pkg/client"
	"net"

	"google.golang.org/grpc"

	grpc2 "github.com/seata/seata-go/pkg/integration/grpc"
	"github.com/seata/seata-go/pkg/util/log"
	"github.com/seata/seata-go/sample/at/grpc/pb"
	"github.com/seata/seata-go/sample/at/grpc/service"
)

func main() {
	client.Init()
	service.InitService()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("server register")
	s := grpc.NewServer(grpc.UnaryInterceptor(grpc2.ServerTransactionInterceptor))

	pb.RegisterATServiceBusinessServer(s, &service.GrpcBusinessService{})
	log.Infof("business listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
