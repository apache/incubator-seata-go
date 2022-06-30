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

package server

import (
	"context"

	"github.com/seata/seata-go/pkg/common/log"
	grpc2 "github.com/seata/seata-go/pkg/integration/grpc"
	"github.com/seata/seata-go/pkg/tm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ServerTransactionInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Errorf("missing grpc metadata")
	}
	xid := md.Get(grpc2.HEADER_KEY)[0]
	if xid == "" {
		xid = md.Get(grpc2.HEADER_KEY_LOWERCASE)[0]
		if xid != "" {
			ctx = tm.InitSeataContext(ctx)
			tm.SetXID(ctx, xid)
		}
	}

	m, err := handler(ctx, req)
	if err != nil {
		log.Errorf("RPC failed with error %v", err)
	}
	return m, err
}
