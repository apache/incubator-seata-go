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

package example

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// TCCServiceBusinessClient is the client API for TCCServiceBusiness service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TCCServiceBusinessClient interface {
	Remoting(ctx context.Context, in *Params, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type tCCServiceBusinessClient struct {
	cc grpc.ClientConnInterface
}

func NewTCCServiceBusinessClient(cc grpc.ClientConnInterface) TCCServiceBusinessClient {
	return &tCCServiceBusinessClient{cc}
}

func (c *tCCServiceBusinessClient) Remoting(ctx context.Context, in *Params, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/TCCServiceBusiness/Remoting", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TCCServiceBusinessServer is the server API for TCCServiceBusiness service.
// All implementations must embed UnimplementedTCCServiceBusinessServer
// for forward compatibility
type TCCServiceBusinessServer interface {
	Remoting(context.Context, *Params) (*emptypb.Empty, error)
	mustEmbedUnimplementedTCCServiceBusinessServer()
}

// UnimplementedTCCServiceBusinessServer must be embedded to have forward compatible implementations.
type UnimplementedTCCServiceBusinessServer struct {
}

func (UnimplementedTCCServiceBusinessServer) Remoting(context.Context, *Params) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Remoting not implemented")
}
func (UnimplementedTCCServiceBusinessServer) mustEmbedUnimplementedTCCServiceBusinessServer() {}

// UnsafeTCCServiceBusinessServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TCCServiceBusinessServer will
// result in compilation errors.
type UnsafeTCCServiceBusinessServer interface {
	mustEmbedUnimplementedTCCServiceBusinessServer()
}

func RegisterTCCServiceBusinessServer(s grpc.ServiceRegistrar, srv TCCServiceBusinessServer) {
	s.RegisterService(&TCCServiceBusiness_ServiceDesc, srv)
}

func _TCCServiceBusiness_Remoting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Params)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TCCServiceBusinessServer).Remoting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/TCCServiceBusiness/Remoting",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TCCServiceBusinessServer).Remoting(ctx, req.(*Params))
	}
	return interceptor(ctx, in, info, handler)
}

// TCCServiceBusiness_ServiceDesc is the grpc.ServiceDesc for TCCServiceBusiness service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TCCServiceBusiness_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "TCCServiceBusiness",
	HandlerType: (*TCCServiceBusinessServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Remoting",
			Handler:    _TCCServiceBusiness_Remoting_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "example/tcc_grpc_test.proto",
}
