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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

// ClientTransactionInterceptor is client interceptor of grpc,
// it's function is obtain xid in SeataContext,
// and put it in the http header.
func ClientTransactionInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// set the XID when intercepting a client request and release it directly when intercepting a response
	if tm.IsSeataContext(ctx) {
		xid := tm.GetXID(ctx)
		header := make(map[string]string)
		header[constant.XidKey] = xid
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(header))
	}

	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	end := time.Now()
	log.Infof("RPC: %s, start time: %s, end time: %s, err: %v", method,
		start.Format("Basic"), end.Format(time.RFC3339), err)
	return err
}

// ServerTransactionInterceptor is server interceptor of grpc
// it's function is get xid from grpc http header ,and put it
// into the context.
func ServerTransactionInterceptor(ctx context.Context, req interface{},
	_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Errorf("missing grpc metadata")
	}
	var xid string
	if slice := md.Get(constant.XidKey); slice != nil && len(slice) > 0 {
		xid = slice[0]
	}
	if xid == "" {
		if slice := md.Get(constant.XidKeyLowercase); slice != nil && len(slice) > 0 {
			xid = slice[0]
		}
	}
	if xid != "" {
		ctx = tm.InitSeataContext(ctx)
		tm.SetXID(ctx, xid)
		log.Infof("global transaction xid is :%s", xid)
	} else {
		log.Info("global transaction xid is empty")
	}

	m, err := handler(ctx, req)
	if err != nil {
		log.Errorf("RPC failed with error %v", err)
	}
	return m, err
}
