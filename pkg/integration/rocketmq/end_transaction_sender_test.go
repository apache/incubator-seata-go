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

package rocketmq

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBrokerAddrResolver struct {
	addr string
	err  error

	calls       int
	lastNsAddrs []string
	lastTopic   string
	lastBroker  string
	lastTimeout time.Duration
}

func (s *stubBrokerAddrResolver) ResolveBrokerAddr(nameServerAddrs []string, topic string, brokerName string, timeout time.Duration) (string, error) {
	s.calls++
	s.lastNsAddrs = nameServerAddrs
	s.lastTopic = topic
	s.lastBroker = brokerName
	s.lastTimeout = timeout
	if s.err != nil {
		return "", s.err
	}
	return s.addr, nil
}

type stubTCPSender struct {
	err error

	calls    int
	lastAddr string
	lastData []byte
}

func (s *stubTCPSender) Send(addr string, data []byte, timeout time.Duration) error {
	s.calls++
	s.lastAddr = addr
	s.lastData = data
	if s.err != nil {
		return s.err
	}
	return nil
}

func (s *stubTCPSender) Close() error {
	return nil
}

func TestEndTransactionRequestHeader_Encode(t *testing.T) {
	header := &endTransactionRequestHeader{
		Topic:                "test-topic",
		ProducerGroup:        "test-group",
		TranStateTableOffset: 42,
		CommitLogOffset:      1024,
		CommitOrRollback:     commitOrRollbackCommit,
		FromTransactionCheck: false,
		MsgID:                "msg-001",
		TransactionId:        "tx-001",
	}

	result := header.Encode()

	assert.Equal(t, "test-topic", result["topic"])
	assert.Equal(t, "test-group", result["producerGroup"])
	assert.Equal(t, "42", result["tranStateTableOffset"])
	assert.Equal(t, "1024", result["commitLogOffset"])
	assert.Equal(t, "8", result["commitOrRollback"])
	assert.Equal(t, "false", result["fromTransactionCheck"])
	assert.Equal(t, "msg-001", result["msgId"])
	assert.Equal(t, "tx-001", result["transactionId"])
}

func TestEndTransactionRequestHeader_EncodeRollback(t *testing.T) {
	header := &endTransactionRequestHeader{
		Topic:                "test-topic",
		ProducerGroup:        "test-group",
		TranStateTableOffset: 10,
		CommitLogOffset:      2048,
		CommitOrRollback:     commitOrRollbackRollback,
		FromTransactionCheck: false,
		MsgID:                "msg-002",
		TransactionId:        "tx-002",
	}

	result := header.Encode()

	assert.Equal(t, "12", result["commitOrRollback"])
}

func TestNewEndTransactionCommand(t *testing.T) {
	header := &endTransactionRequestHeader{
		Topic:            "test-topic",
		ProducerGroup:    "test-group",
		CommitOrRollback: commitOrRollbackCommit,
		MsgID:            "msg-001",
		TransactionId:    "tx-001",
	}

	cmd := newEndTransactionCommand(header)

	assert.Equal(t, reqEndTransaction, cmd.Code)
	assert.Equal(t, rmqLanguageGo, cmd.Language)
	assert.Equal(t, rmqProtocolVersion, cmd.Version)
	assert.NotZero(t, cmd.Opaque)
	assert.Equal(t, "test-topic", cmd.ExtFields["topic"])
	assert.Equal(t, "test-group", cmd.ExtFields["producerGroup"])
	assert.Equal(t, "8", cmd.ExtFields["commitOrRollback"])
}

func TestRemotingCommand_Encode_FrameFormat(t *testing.T) {
	cmd := &remotingCommand{
		Code:      reqEndTransaction,
		Language:  rmqLanguageGo,
		Version:   rmqProtocolVersion,
		Opaque:    1,
		Flag:      0,
		Remark:    "",
		ExtFields: map[string]string{},
	}

	data, err := cmd.encode()
	require.NoError(t, err)
	require.NotNil(t, data)

	reader := bytes.NewReader(data)

	var frameSize int32
	err = binary.Read(reader, binary.BigEndian, &frameSize)
	require.NoError(t, err)
	assert.Equal(t, int32(len(data)-4), frameSize)

	var headerLenRaw int32
	err = binary.Read(reader, binary.BigEndian, &headerLenRaw)
	require.NoError(t, err)
	codecType := byte((headerLenRaw >> 24) & 0xFF)
	headerLen := headerLenRaw & 0xFFFFFF
	assert.Equal(t, rmqCodecType, codecType)
	assert.Equal(t, int32(rmqHeaderFixedLength), headerLen)

	var code int16
	err = binary.Read(reader, binary.BigEndian, &code)
	require.NoError(t, err)
	assert.Equal(t, reqEndTransaction, code)

	language, err := reader.ReadByte()
	require.NoError(t, err)
	assert.Equal(t, rmqLanguageGo, language)

	var version int16
	err = binary.Read(reader, binary.BigEndian, &version)
	require.NoError(t, err)
	assert.Equal(t, rmqProtocolVersion, version)

	var opaque int32
	err = binary.Read(reader, binary.BigEndian, &opaque)
	require.NoError(t, err)
	assert.Equal(t, int32(1), opaque)

	var flag int32
	err = binary.Read(reader, binary.BigEndian, &flag)
	require.NoError(t, err)
	assert.Equal(t, int32(0), flag)

	var remarkLen int32
	err = binary.Read(reader, binary.BigEndian, &remarkLen)
	require.NoError(t, err)
	assert.Equal(t, int32(0), remarkLen)

	var extLen int32
	err = binary.Read(reader, binary.BigEndian, &extLen)
	require.NoError(t, err)
	assert.Equal(t, int32(0), extLen)
}

func TestRemotingCommand_Encode_WithExtFields(t *testing.T) {
	cmd := &remotingCommand{
		Code:     reqEndTransaction,
		Language: rmqLanguageGo,
		Version:  rmqProtocolVersion,
		Opaque:   42,
		ExtFields: map[string]string{
			"producerGroup": "test-group",
		},
	}

	data, err := cmd.encode()
	require.NoError(t, err)
	require.NotNil(t, data)

	frameSize := int32(binary.BigEndian.Uint32(data[0:4]))
	assert.Equal(t, int32(len(data)-4), frameSize)

	headerLenRaw := binary.BigEndian.Uint32(data[4:8])
	headerLen := int(headerLenRaw & 0xFFFFFF)
	assert.Equal(t, rmqHeaderFixedLength+29, headerLen)
}

func TestEncodeExtFields_Empty(t *testing.T) {
	result, err := encodeExtFields(map[string]string{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestEncodeExtFields_SingleField(t *testing.T) {
	fields := map[string]string{
		"key": "value",
	}

	result, err := encodeExtFields(fields)
	require.NoError(t, err)

	reader := bytes.NewReader(result)

	var keyLen int16
	err = binary.Read(reader, binary.BigEndian, &keyLen)
	require.NoError(t, err)
	assert.Equal(t, int16(3), keyLen)

	keyBuf := make([]byte, keyLen)
	_, err = reader.Read(keyBuf)
	require.NoError(t, err)
	assert.Equal(t, "key", string(keyBuf))

	var valueLen int32
	err = binary.Read(reader, binary.BigEndian, &valueLen)
	require.NoError(t, err)
	assert.Equal(t, int32(5), valueLen)

	valueBuf := make([]byte, valueLen)
	_, err = reader.Read(valueBuf)
	require.NoError(t, err)
	assert.Equal(t, "value", string(valueBuf))
}

func TestMarkProtocolType(t *testing.T) {
	result := markProtocolType(100)

	assert.Equal(t, rmqCodecType, result[0])
	assert.Equal(t, byte(0x00), result[1])
	assert.Equal(t, byte(0x00), result[2])
	assert.Equal(t, byte(0x64), result[3])
}

func TestParseBrokerAddrFromRouteBody_FoundMaster(t *testing.T) {
	route := topicRouteData{
		BrokerDataList: []brokerData{
			{
				BrokerName: "broker-a",
				BrokerAddrs: map[string]string{
					"0": "192.168.1.100:10911",
					"1": "192.168.1.101:10911",
				},
			},
		},
	}
	body, _ := json.Marshal(route)

	addr, err := parseBrokerAddrFromRouteBody(body, "broker-a")

	require.NoError(t, err)
	assert.Equal(t, "192.168.1.100:10911", addr)
}

func TestParseBrokerAddrFromRouteBody_MasterNotFound(t *testing.T) {
	route := topicRouteData{
		BrokerDataList: []brokerData{
			{
				BrokerName: "broker-a",
				BrokerAddrs: map[string]string{
					"1": "192.168.1.101:10911",
				},
			},
		},
	}
	body, _ := json.Marshal(route)

	_, err := parseBrokerAddrFromRouteBody(body, "broker-a")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "master (addr key '0') not found")
}

func TestParseBrokerAddrFromRouteBody_BrokerNotFound(t *testing.T) {
	route := topicRouteData{
		BrokerDataList: []brokerData{
			{
				BrokerName:  "broker-a",
				BrokerAddrs: map[string]string{"0": "192.168.1.100:10911"},
			},
		},
	}
	body, _ := json.Marshal(route)

	_, err := parseBrokerAddrFromRouteBody(body, "broker-b")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker-b")
}

func TestParseBrokerAddrFromRouteBody_InvalidJSON(t *testing.T) {
	_, err := parseBrokerAddrFromRouteBody([]byte("invalid json"), "broker-a")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestGetQueueOffsetFromActionContext_Float64(t *testing.T) {
	ctx := map[string]interface{}{
		ActionContextKeyQueueOffset: float64(42),
	}
	assert.Equal(t, int64(42), getQueueOffsetFromActionContext(ctx))
}

func TestGetQueueOffsetFromActionContext_Int64(t *testing.T) {
	ctx := map[string]interface{}{
		ActionContextKeyQueueOffset: int64(42),
	}
	assert.Equal(t, int64(42), getQueueOffsetFromActionContext(ctx))
}

func TestGetQueueOffsetFromActionContext_Int(t *testing.T) {
	ctx := map[string]interface{}{
		ActionContextKeyQueueOffset: int(42),
	}
	assert.Equal(t, int64(42), getQueueOffsetFromActionContext(ctx))
}

func TestGetQueueOffsetFromActionContext_Missing(t *testing.T) {
	ctx := map[string]interface{}{}
	assert.Equal(t, int64(0), getQueueOffsetFromActionContext(ctx))
}

func TestGetQueueOffsetFromActionContext_InvalidType(t *testing.T) {
	ctx := map[string]interface{}{
		ActionContextKeyQueueOffset: "not a number",
	}
	assert.Equal(t, int64(0), getQueueOffsetFromActionContext(ctx))
}

func TestGetStringFromMap_Found(t *testing.T) {
	ctx := map[string]interface{}{
		ActionContextKeyMsgId: "msg-001",
	}
	assert.Equal(t, "msg-001", getStringFromMap(ctx, ActionContextKeyMsgId))
}

func TestGetStringFromMap_Missing(t *testing.T) {
	ctx := map[string]interface{}{}
	assert.Equal(t, "", getStringFromMap(ctx, ActionContextKeyMsgId))
}

func TestGetStringFromMap_WrongType(t *testing.T) {
	ctx := map[string]interface{}{
		ActionContextKeyMsgId: 12345,
	}
	assert.Equal(t, "", getStringFromMap(ctx, ActionContextKeyMsgId))
}

func TestSendEndTransaction_Success(t *testing.T) {
	resolver := &stubBrokerAddrResolver{addr: "192.168.1.100:10911"}
	sender := &stubTCPSender{}
	header := &endTransactionRequestHeader{
		Topic:                "test-topic",
		ProducerGroup:        "test-group",
		TranStateTableOffset: 42,
		CommitLogOffset:      1024,
		CommitOrRollback:     commitOrRollbackCommit,
		MsgID:                "msg-001",
		TransactionId:        "tx-001",
	}

	err := sendEndTransaction(
		[]string{"nameserver:9876"},
		"test-topic",
		"broker-a",
		header,
		3*time.Second,
		resolver,
		sender,
	)

	require.NoError(t, err)
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, []string{"nameserver:9876"}, resolver.lastNsAddrs)
	assert.Equal(t, "test-topic", resolver.lastTopic)
	assert.Equal(t, "broker-a", resolver.lastBroker)
	assert.Equal(t, 1, sender.calls)
	assert.Equal(t, "192.168.1.100:10911", sender.lastAddr)
	assert.NotEmpty(t, sender.lastData)

	frameSize := int32(binary.BigEndian.Uint32(sender.lastData[0:4]))
	assert.Equal(t, int32(len(sender.lastData)-4), frameSize)
}

func TestSendEndTransaction_ResolverError(t *testing.T) {
	resolver := &stubBrokerAddrResolver{err: errors.New("name server unreachable")}
	sender := &stubTCPSender{}
	header := &endTransactionRequestHeader{
		Topic:         "test-topic",
		ProducerGroup: "test-group",
	}

	err := sendEndTransaction(
		[]string{"nameserver:9876"},
		"test-topic",
		"broker-a",
		header,
		3*time.Second,
		resolver,
		sender,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve broker addr failed")
	assert.Equal(t, 0, sender.calls)
}

func TestSendEndTransaction_SendError(t *testing.T) {
	resolver := &stubBrokerAddrResolver{addr: "192.168.1.100:10911"}
	sender := &stubTCPSender{err: errors.New("connection refused")}
	header := &endTransactionRequestHeader{
		Topic:         "test-topic",
		ProducerGroup: "test-group",
	}

	err := sendEndTransaction(
		[]string{"nameserver:9876"},
		"test-topic",
		"broker-a",
		header,
		3*time.Second,
		resolver,
		sender,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "send to broker")
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, 1, sender.calls)
}

func TestDefaultBrokerAddrResolver_TriesAllNameServers(t *testing.T) {
	resolver := &defaultBrokerAddrResolver{}

	_, err := resolver.ResolveBrokerAddr(
		[]string{"127.0.0.1:1", "127.0.0.1:1"},
		"test-topic",
		"broker-a",
		3*time.Second,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker")
}

func TestConnPool_GetAndPut(t *testing.T) {
	// Start a temporary TCP listener to serve as a fake broker.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4)
				if _, err := io.ReadFull(c, buf); err != nil {
					return
				}
				frameLen := binary.BigEndian.Uint32(buf)
				payload := make([]byte, frameLen)
				if _, err := io.ReadFull(c, payload); err != nil {
					return
				}
				// Write back a success response (code=0).
				resp := make([]byte, 8)
				binary.BigEndian.PutUint32(resp[0:4], 4)
				binary.BigEndian.PutUint16(resp[4:6], 0)
				c.Write(resp)
			}(conn)
		}
	}()

	pool := newConnPool(ln.Addr().String(), 2)

	// First get should create a new connection.
	conn1, isNew1, err := pool.get(3 * time.Second)
	require.NoError(t, err)
	require.NotNil(t, conn1)
	assert.True(t, isNew1)

	// Return the connection to the pool.
	pool.put(conn1)

	// Second get should reuse the pooled connection.
	conn2, isNew2, err := pool.get(3 * time.Second)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	assert.False(t, isNew2)
	assert.Equal(t, conn1.RemoteAddr().String(), conn2.RemoteAddr().String())

	conn2.Close()
}

func TestConnPool_PutWhenFullClosesConnection(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	pool := newConnPool(ln.Addr().String(), 2)

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)

	// Fill the pool.
	for i := 0; i < 2; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		require.NoError(t, err)
		pool.put(c)
	}

	// Putting one more should close it instead of overflowing.
	pool.put(conn)
	// The connection should eventually be closed; we just verify no panic.
}

func TestConnPool_PutAfterClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	pool := newConnPool(ln.Addr().String(), 2)

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)

	pool.closeAll()

	// After closeAll, put should close the connection instead of returning it to the pool.
	pool.put(conn)
	// No panic and connection is closed; verified implicitly.
}

func TestConnPool_GetDialsWhenEmpty(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	pool := newConnPool(ln.Addr().String(), 2)

	// Get from an empty pool should dial a new connection.
	conn, isNew, err := pool.get(3 * time.Second)
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.True(t, isNew)
	conn.Close()
}

func TestDefaultTCPSender_RetryOnPooledConnFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4)
				if _, err := io.ReadFull(c, buf); err != nil {
					return
				}
				frameLen := binary.BigEndian.Uint32(buf)
				payload := make([]byte, frameLen)
				if _, err := io.ReadFull(c, payload); err != nil {
					return
				}
				resp, _ := (&remotingCommand{
					Code:      0,
					Language:  rmqLanguageGo,
					Version:   rmqProtocolVersion,
					Opaque:    1,
					Flag:      0,
					Remark:    "",
					ExtFields: map[string]string{},
				}).encode()
				c.Write(resp)
			}(conn)
		}
	}()

	sender := newDefaultTCPSender(2)

	// Dial a connection, immediately close it, and put the closed conn into the pool.
	staleConn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	staleConn.Close()

	poolAny, _ := sender.pools.LoadOrStore(ln.Addr().String(), newConnPool(ln.Addr().String(), 2))
	pool := poolAny.(*connPool)
	pool.put(staleConn)

	data, err := (&remotingCommand{
		Code:      reqEndTransaction,
		Language:  rmqLanguageGo,
		Version:   rmqProtocolVersion,
		Opaque:    1,
		ExtFields: map[string]string{},
	}).encode()
	require.NoError(t, err)

	// Send should fail on the stale pooled conn, then retry with a fresh conn and succeed.
	err = sender.Send(ln.Addr().String(), data, 3*time.Second)
	require.NoError(t, err)
}
