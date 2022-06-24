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

package client

import (
	"context"
	"github.com/seata/seata-go/pkg/common/log"
	grpc2 "github.com/seata/seata-go/pkg/integration/grpc"
	"github.com/seata/seata-go/pkg/tm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"time"
)

func ClientTransactionInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	var xid string
	// 当拦截客户端请求时设置XID，当拦截到响应时直接放行
	if tm.IsSeataContext(ctx) {
		xid = tm.GetXID(ctx)
		header := make(map[string]string)
		header[grpc2.HEADER_KEY] = xid
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(header))
	}

	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	end := time.Now()
	log.Infof("RPC: %s, start time: %s, end time: %s, err: %v", method, start.Format("Basic"), end.Format(time.RFC3339), err)
	return err
}
