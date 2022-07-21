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

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"time"

	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/sample/tcc/grpc/service"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/integration/grpc/client"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(client.ClientTransactionInterceptor))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c1 := pb.NewTCCServiceBusiness1Client(conn)
	c2 := pb.NewTCCServiceBusiness2Client(conn)
	tcc.NewTCCServiceProxy()
	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	// GlobalTransactional is starting
	log.Infof("global transaction begin")
	ctx = tm.Begin(ctx, "TestTCCServiceBusiness")
	defer func() {
		resp := tm.CommitOrRollback(ctx, &err)
		log.Infof("tx result %v", resp)
		<-make(chan bool)
	}()
	defer cancel()

	tccProxy1, err := tcc.NewTCCServiceProxy(service.BusinessServer1)
	if err != nil {
		log.Errorf("get userProviderProxy tcc service proxy error, %v", err.Error())
		return
	}

	tccProxy2, err := tcc.NewTCCServiceProxy(service.BusinessServer2)
	if err != nil {
		log.Errorf("get userProviderProxy tcc service proxy error, %v", err.Error())
		return
	}

	// BranchTransactional remoting running
	go func() {
		log.Infof("branch transaction begin 1")
		var r interface{}
		r, err = tccProxy1.Prepare(ctx, &pb.Params{
			A: "A1",
			B: "B1",
			C: "C1",
		})
		if err != nil {
			log.Fatalf("could not do TestTCCServiceBusiness1: %v", err)
		}
		log.Infof("TestTCCServiceBusiness1#Prepare2 res: %s", r.(*wrapperspb.BoolValue))
	}()

	go func() {
		log.Infof("branch transaction begin 2")
		var r interface{}
		r, err = tccProxy2.Prepare(ctx, &pb.Params{
			A: "A2",
			B: "B2",
			C: "C2",
		})

		if err != nil {
			log.Fatalf("could not do TestTCCServiceBusiness1: %v", err)
		}

		log.Infof("TestTCCServiceBusiness1#Prepare2 res: %s", r.(*wrapperspb.BoolValue))
	}()
}
