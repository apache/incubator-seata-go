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
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/integration/grpc/client"
	"github.com/seata/seata-go/pkg/rm/tcc/remoting/gprc/example"
	"github.com/seata/seata-go/pkg/tm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(client.ClientTransactionInterceptor))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := example.NewTCCServiceBusinessClient(conn)
	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	// GlobalTransactional is starting
	log.Infof("global transaction begin")
	ctx = tm.Begin(ctx, "TestTCCServiceBusiness")
	defer func() {
		resp := tm.CommitOrRollback(ctx, err)
		log.Infof("tx result %v", resp)
	}()
	defer cancel()

	// BranchTransactional remoting running
	log.Infof("branch transaction begin")
	r, err := c.Remoting(ctx, &example.Params{A: "1", B: "2"})
	if err != nil {
		log.Fatalf("could not do TestTCCServiceBusiness: %v", err)
	}
	log.Infof("TestTCCServiceBusiness#Prepare res: %s", r.String())
}
