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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/seata/seata-go/pkg/client"
	grpc2 "github.com/seata/seata-go/pkg/integration/grpc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	"github.com/seata/seata-go/sample/tcc/grpc/pb"
)

func main() {
	flag.Parse()
	// to set up grpc env
	// set up a connection to the server.
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpc2.ClientTransactionInterceptor))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c1, c2 := pb.NewTCCServiceBusiness1Client(conn), pb.NewTCCServiceBusiness2Client(conn)

	client.InitPath("./sample/conf/seatago.yml")
	tm.WithGlobalTx(
		context.Background(),
		&tm.GtxConfig{
			Name: "TccSampleLocalGlobalTx",
		},
		func(ctx context.Context) (re error) {
			r1, re := c1.Remoting(ctx, &pb.Params{A: "1", B: "2"})
			if re != nil {
				log.Fatalf("could not do TestTCCServiceBusiness 1: %v", re)
				return
			}
			log.Infof("TestTCCServiceBusiness#Prepare res: %s", r1)

			r2, re := c2.Remoting(ctx, &pb.Params{A: "3", B: "4"})
			if re != nil {
				log.Fatalf("could not do TestTCCServiceBusiness 2: %v", re)
				return
			}
			log.Infof("TestTCCServiceBusiness#Prepare res: %v", r2)

			return
		})
	<-make(chan struct{})
}
