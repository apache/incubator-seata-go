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
	"fmt"
	"strconv"

	"seata.apache.org/seata-go/pkg/protocol/grpc"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/util/log"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func Encode(msg message.RpcMessage) (*pb.GrpcMessageProto, error) {
	rpcMessage := &pb.GrpcMessageProto{
		Id:          msg.ID,
		MessageType: int32(msg.Type),
		HeadMap:     msg.HeadMap,
	}
	if rpcMessage.HeadMap == nil {
		rpcMessage.HeadMap = make(map[string]string)
	}
	rpcMessage.HeadMap[string(grpc.CodecType)] = strconv.Itoa(int(msg.Codec))
	rpcMessage.HeadMap[string(grpc.CompressType)] = strconv.Itoa(int(msg.Compressor))
	anyMsg, err := anypb.New(msg.Body.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("could not new any msg: %v", err)
	}
	marshal, err := proto.Marshal(anyMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal any msg: %v", err)
	}
	rpcMessage.Body = marshal
	return rpcMessage, nil
}

func Decode(msg *pb.GrpcMessageProto) (message.RpcMessage, error) {
	rpcMessage := message.RpcMessage{
		ID:      msg.Id,
		Type:    message.RequestType(msg.MessageType),
		HeadMap: msg.HeadMap,
	}
	if val, ok := msg.HeadMap[string(grpc.CodecType)]; ok {
		i, err := strconv.ParseInt(val, 10, 8)
		if err != nil {
			return rpcMessage, fmt.Errorf("codec type: faild to convert string:%v to byte. err: %v", val, err)
		}
		rpcMessage.Codec = byte(i)
	}
	if val, ok := msg.HeadMap[string(grpc.CompressType)]; ok {
		i, err := strconv.ParseInt(val, 10, 8)
		if err != nil {
			return rpcMessage, fmt.Errorf("compress type: faild to convert string:%v to byte. err: %v", val, err)
		}
		rpcMessage.Compressor = byte(i)
	}
	if msg.Body == nil {
		if msg.MessageType == int32(message.RequestTypeHeartbeatResponse) {
			rpcMessage.Body = &pb.HeartbeatMessageProto{Ping: false}
		} else if msg.MessageType == int32(message.RequestTypeHeartbeatRequest) {
			rpcMessage.Body = &pb.HeartbeatMessageProto{Ping: true}
		} else {
			return rpcMessage, fmt.Errorf("rpc message recv empty body")
		}
		return rpcMessage, nil
	}
	var anyMsg anypb.Any
	if err := proto.Unmarshal(msg.Body, &anyMsg); err != nil {
		log.Errorf("failed to unmarshal to any: %v", err)
	}
	body, err := anyMsg.UnmarshalNew()
	if err != nil {
		log.Errorf("failed to unmarshal to msg: %v", err)
	}
	rpcMessage.Body = body
	return rpcMessage, nil
}
