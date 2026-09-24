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
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	reqEndTransaction int16 = 37

	reqGetRouteInfoByTopic int16 = 105

	rmqProtocolVersion int16 = 317

	rmqLanguageGo byte = 9

	rmqCodecType byte = 1

	rmqHeaderFixedLength = 21

	// rmqResponseHeaderFixedPartLength is the size of the fixed fields in a
	// RocketMQ remoting response header:
	// code(2) + language(1) + version(2) + opaque(4) + flag(4) = 13 bytes.
	rmqResponseHeaderFixedPartLength = 13

	commitOrRollbackCommit = 8

	commitOrRollbackRollback = 12

	maxFrameSize = 1024 * 1024 // 1MB upper bound for remoting frame to prevent OOM

	defaultConnPoolSize = 4

	connPoolCleanInterval = 60 // seconds

	connPoolIdleTimeout = 600 // seconds (10 minutes)
)

var opaqueCounter int32

type endTransactionRequestHeader struct {
	Topic                string
	ProducerGroup        string
	TranStateTableOffset int64
	CommitLogOffset      int64
	CommitOrRollback     int
	FromTransactionCheck bool
	MsgID                string
	TransactionId        string
}

func (h *endTransactionRequestHeader) Encode() map[string]string {
	return map[string]string{
		"topic":                h.Topic,
		"producerGroup":        h.ProducerGroup,
		"tranStateTableOffset": fmt.Sprintf("%d", h.TranStateTableOffset),
		"commitLogOffset":      fmt.Sprintf("%d", h.CommitLogOffset),
		"commitOrRollback":     fmt.Sprintf("%d", h.CommitOrRollback),
		"fromTransactionCheck": fmt.Sprintf("%v", h.FromTransactionCheck),
		"msgId":                h.MsgID,
		"transactionId":        h.TransactionId,
	}
}

type remotingCommand struct {
	Code      int16
	Language  byte
	Version   int16
	Opaque    int32
	Flag      int32
	Remark    string
	ExtFields map[string]string
	Body      []byte
}

func newEndTransactionCommand(header *endTransactionRequestHeader) *remotingCommand {
	return &remotingCommand{
		Code:      reqEndTransaction,
		Language:  rmqLanguageGo,
		Version:   rmqProtocolVersion,
		Opaque:    atomic.AddInt32(&opaqueCounter, 1),
		ExtFields: header.Encode(),
	}
}

func (cmd *remotingCommand) encode() ([]byte, error) {
	headerBytes, err := cmd.encodeHeader()
	if err != nil {
		return nil, fmt.Errorf("encode header failed: %w", err)
	}

	frameSize := 4 + len(headerBytes) + len(cmd.Body)
	buf := bytes.NewBuffer(make([]byte, 0, 4+frameSize))

	if err := binary.Write(buf, binary.BigEndian, int32(frameSize)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, markProtocolType(int32(len(headerBytes)))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(headerBytes); err != nil {
		return nil, err
	}
	if len(cmd.Body) > 0 {
		if _, err := buf.Write(cmd.Body); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (cmd *remotingCommand) encodeHeader() ([]byte, error) {
	extBytes, err := encodeExtFields(cmd.ExtFields)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, rmqHeaderFixedLength+len(cmd.Remark)+len(extBytes)))

	if err := binary.Write(buf, binary.BigEndian, cmd.Code); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(cmd.Language); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, cmd.Version); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, cmd.Opaque); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, cmd.Flag); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, int32(len(cmd.Remark))); err != nil {
		return nil, err
	}
	if len(cmd.Remark) > 0 {
		if _, err := buf.Write([]byte(cmd.Remark)); err != nil {
			return nil, err
		}
	}
	if err := binary.Write(buf, binary.BigEndian, int32(len(extBytes))); err != nil {
		return nil, err
	}
	if len(extBytes) > 0 {
		if _, err := buf.Write(extBytes); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func encodeExtFields(fields map[string]string) ([]byte, error) {
	if len(fields) == 0 {
		return []byte{}, nil
	}

	buf := bytes.NewBuffer(nil)
	for key, value := range fields {
		if err := binary.Write(buf, binary.BigEndian, int16(len(key))); err != nil {
			return nil, err
		}
		if _, err := buf.Write([]byte(key)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, int32(len(value))); err != nil {
			return nil, err
		}
		if _, err := buf.Write([]byte(value)); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func markProtocolType(source int32) []byte {
	result := make([]byte, 4)
	result[0] = rmqCodecType
	result[1] = byte((source >> 16) & 0xFF)
	result[2] = byte((source >> 8) & 0xFF)
	result[3] = byte(source & 0xFF)
	return result
}

type brokerAddrResolver interface {
	ResolveBrokerAddr(nameServerAddrs []string, topic string, brokerName string, timeout time.Duration) (string, error)
}

type tcpSender interface {
	Send(addr string, data []byte, timeout time.Duration) error
	Close() error
}

type defaultBrokerAddrResolver struct{}

func (r *defaultBrokerAddrResolver) ResolveBrokerAddr(nameServerAddrs []string, topic string, brokerName string, timeout time.Duration) (string, error) {
	for _, nsAddr := range nameServerAddrs {
		addr, err := queryBrokerAddrFromNameServer(nsAddr, topic, brokerName, timeout)
		if err == nil && addr != "" {
			return addr, nil
		}
	}
	return "", fmt.Errorf("broker %s addr not found from name servers", brokerName)
}

type connPool struct {
	addr       string
	conns      chan net.Conn
	lastUsedAt int64 // atomic, unix timestamp
	closed     int32 // atomic
}

func newConnPool(addr string, size int) *connPool {
	if size <= 0 {
		size = defaultConnPoolSize
	}
	return &connPool{
		addr:       addr,
		conns:      make(chan net.Conn, size),
		lastUsedAt: time.Now().Unix(),
	}
}

// get returns a connection and a bool indicating whether it was freshly
// dialed (true) or reused from the pool (false).
func (p *connPool) get(timeout time.Duration) (net.Conn, bool, error) {
	atomic.StoreInt64(&p.lastUsedAt, time.Now().Unix())
	select {
	case conn := <-p.conns:
		return conn, false, nil
	default:
	}
	conn, err := net.DialTimeout("tcp", p.addr, timeout)
	return conn, true, err
}

func (p *connPool) put(conn net.Conn) {
	if atomic.LoadInt32(&p.closed) == 1 {
		conn.Close()
		return
	}
	select {
	case p.conns <- conn:
	default:
		conn.Close()
	}
}

func (p *connPool) closeAll() {
	atomic.StoreInt32(&p.closed, 1)
	for {
		select {
		case conn := <-p.conns:
			conn.Close()
		default:
			return
		}
	}
}

type defaultTCPSender struct {
	pools         sync.Map // addr -> *connPool
	poolSize      int
	lastCleanedAt int64 // atomic, unix timestamp
}

func newDefaultTCPSender(poolSize int) *defaultTCPSender {
	if poolSize <= 0 {
		poolSize = defaultConnPoolSize
	}
	return &defaultTCPSender{
		poolSize:      poolSize,
		lastCleanedAt: time.Now().Unix(),
	}
}

func (s *defaultTCPSender) Send(addr string, data []byte, timeout time.Duration) error {
	s.cleanIdlePoolsIfNeeded()

	poolAny, _ := s.pools.LoadOrStore(addr, newConnPool(addr, s.poolSize))
	pool := poolAny.(*connPool)

	conn, isNew, err := pool.get(timeout)
	if err != nil {
		return fmt.Errorf("dial broker %s failed: %w", addr, err)
	}

	sendAndRead := func(c net.Conn) error {
		if err := c.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return fmt.Errorf("set write deadline failed: %w", err)
		}
		if _, err := c.Write(data); err != nil {
			return fmt.Errorf("write to broker %s failed: %w", addr, err)
		}

		if err := c.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return fmt.Errorf("set read deadline failed: %w", err)
		}
		frameLenBuf := make([]byte, 4)
		if _, err := io.ReadFull(c, frameLenBuf); err != nil {
			return fmt.Errorf("read response frame length from broker %s failed: %w", addr, err)
		}
		frameLen := int(binary.BigEndian.Uint32(frameLenBuf))
		if frameLen < 8 {
			return fmt.Errorf("broker %s response frame too short: %d", addr, frameLen)
		}
		if frameLen > maxFrameSize {
			return fmt.Errorf("broker %s response frame too large: %d", addr, frameLen)
		}
		frameBuf := make([]byte, frameLen)
		if _, err := io.ReadFull(c, frameBuf); err != nil {
			return fmt.Errorf("read response frame from broker %s failed: %w", addr, err)
		}

		code := int16(binary.BigEndian.Uint16(frameBuf[4:6]))
		if code != 0 {
			return fmt.Errorf("broker %s rejected END_TRANSACTION, responseCode=%d", addr, code)
		}
		return nil
	}

	if err := sendAndRead(conn); err != nil {
		conn.Close()
		// If the connection was pooled (not freshly dialed), it may have
		// been closed by the broker during idle time. Retry once with a
		// fresh connection before giving up.
		if !isNew {
			freshConn, dialErr := net.DialTimeout("tcp", addr, timeout)
			if dialErr == nil {
				if retryErr := sendAndRead(freshConn); retryErr != nil {
					freshConn.Close()
					return retryErr
				}
				pool.put(freshConn)
				return nil
			}
		}
		return err
	}

	pool.put(conn)
	return nil
}

func (s *defaultTCPSender) Close() error {
	s.pools.Range(func(key, value interface{}) bool {
		pool := value.(*connPool)
		if _, loaded := s.pools.LoadAndDelete(key); loaded {
			pool.closeAll()
		}
		return true
	})
	return nil
}

func (s *defaultTCPSender) cleanIdlePoolsIfNeeded() {
	now := time.Now().Unix()
	lastClean := atomic.LoadInt64(&s.lastCleanedAt)
	if now-lastClean < int64(connPoolCleanInterval) {
		return
	}
	if !atomic.CompareAndSwapInt64(&s.lastCleanedAt, lastClean, now) {
		return
	}

	s.pools.Range(func(key, value interface{}) bool {
		pool := value.(*connPool)
		if now-atomic.LoadInt64(&pool.lastUsedAt) > int64(connPoolIdleTimeout) {
			if _, loaded := s.pools.LoadAndDelete(key); loaded {
				pool.closeAll()
			}
		}
		return true
	})
}

func sendEndTransaction(
	nameServerAddrs []string,
	topic string,
	brokerName string,
	header *endTransactionRequestHeader,
	timeout time.Duration,
	resolver brokerAddrResolver,
	sender tcpSender,
) error {
	brokerAddr, err := resolver.ResolveBrokerAddr(nameServerAddrs, topic, brokerName, timeout)
	if err != nil {
		return fmt.Errorf("resolve broker addr failed: %w", err)
	}

	cmd := newEndTransactionCommand(header)

	data, err := cmd.encode()
	if err != nil {
		return fmt.Errorf("encode command failed: %w", err)
	}

	if err := sender.Send(brokerAddr, data, timeout); err != nil {
		return fmt.Errorf("send to broker %s failed: %w", brokerAddr, err)
	}

	return nil
}

func queryBrokerAddrFromNameServer(nameServerAddr string, topic string, brokerName string, timeout time.Duration) (string, error) {
	cmd := &remotingCommand{
		Code:     reqGetRouteInfoByTopic,
		Language: rmqLanguageGo,
		Version:  rmqProtocolVersion,
		Opaque:   atomic.AddInt32(&opaqueCounter, 1),
		ExtFields: map[string]string{
			"topic": topic,
		},
	}

	data, err := cmd.encode()
	if err != nil {
		return "", fmt.Errorf("encode route request failed: %w", err)
	}

	conn, err := net.DialTimeout("tcp", nameServerAddr, timeout)
	if err != nil {
		return "", fmt.Errorf("dial name server %s failed: %w", nameServerAddr, err)
	}
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	if _, err := conn.Write(data); err != nil {
		return "", fmt.Errorf("send route request failed: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	frameLenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, frameLenBuf); err != nil {
		return "", fmt.Errorf("read frame length failed: %w", err)
	}
	frameLen := int(binary.BigEndian.Uint32(frameLenBuf))
	if frameLen < 8 {
		return "", fmt.Errorf("name server %s response frame too short: %d", nameServerAddr, frameLen)
	}
	if frameLen > maxFrameSize {
		return "", fmt.Errorf("name server %s response frame too large: %d", nameServerAddr, frameLen)
	}

	frameBuf := make([]byte, frameLen)
	if _, err := io.ReadFull(conn, frameBuf); err != nil {
		return "", fmt.Errorf("read frame data failed: %w", err)
	}

	if len(frameBuf) < 4 {
		return "", fmt.Errorf("response frame too short")
	}
	oriHeaderLen := binary.BigEndian.Uint32(frameBuf[0:4])
	headerLen := int(oriHeaderLen & 0xFFFFFF)
	if len(frameBuf) < 4+headerLen {
		return "", fmt.Errorf("response header truncated")
	}

	respCode := int16(binary.BigEndian.Uint16(frameBuf[4:6]))
	if respCode != 0 {
		return "", fmt.Errorf("name server %s returned error code %d for route query", nameServerAddr, respCode)
	}

	_, err = decodeResponseHeader(frameBuf[4 : 4+headerLen])
	if err != nil {
		return "", fmt.Errorf("decode response header failed: %w", err)
	}

	bodyStart := 4 + headerLen
	if bodyStart >= len(frameBuf) {
		return "", fmt.Errorf("response has no body")
	}
	body := frameBuf[bodyStart:]

	return parseBrokerAddrFromRouteBody(body, brokerName)
}

func decodeResponseHeader(data []byte) (map[string]string, error) {
	buf := bytes.NewReader(data)

	if buf.Len() < rmqResponseHeaderFixedPartLength {
		return nil, fmt.Errorf("header too short for fixed fields")
	}
	discard := make([]byte, rmqResponseHeaderFixedPartLength)
	if _, err := io.ReadFull(buf, discard); err != nil {
		return nil, err
	}

	var remarkLen int32
	if err := binary.Read(buf, binary.BigEndian, &remarkLen); err != nil {
		return nil, err
	}
	if remarkLen < 0 || int(remarkLen) > buf.Len() {
		return nil, fmt.Errorf("invalid remark length: %d", remarkLen)
	}
	if remarkLen > 0 {
		discardRemark := make([]byte, remarkLen)
		if _, err := io.ReadFull(buf, discardRemark); err != nil {
			return nil, err
		}
	}

	var extLen int32
	if err := binary.Read(buf, binary.BigEndian, &extLen); err != nil {
		return nil, err
	}
	if extLen < 0 || int(extLen) > buf.Len() {
		return nil, fmt.Errorf("invalid ext fields length: %d", extLen)
	}
	extFields := make(map[string]string)
	if extLen > 0 {
		extData := make([]byte, extLen)
		if _, err := io.ReadFull(buf, extData); err != nil {
			return nil, err
		}
		extBuf := bytes.NewReader(extData)
		for extBuf.Len() > 0 {
			var kLen int16
			if err := binary.Read(extBuf, binary.BigEndian, &kLen); err != nil {
				return nil, fmt.Errorf("read ext key length failed: %w", err)
			}
			if kLen < 0 || int(kLen) > extBuf.Len() {
				return nil, fmt.Errorf("invalid ext key length: %d", kLen)
			}
			key := make([]byte, kLen)
			if _, err := io.ReadFull(extBuf, key); err != nil {
				return nil, fmt.Errorf("read ext key failed: %w", err)
			}
			var vLen int32
			if err := binary.Read(extBuf, binary.BigEndian, &vLen); err != nil {
				return nil, fmt.Errorf("read ext value length failed: %w", err)
			}
			if vLen < 0 || int(vLen) > extBuf.Len() {
				return nil, fmt.Errorf("invalid ext value length: %d", vLen)
			}
			value := make([]byte, vLen)
			if _, err := io.ReadFull(extBuf, value); err != nil {
				return nil, fmt.Errorf("read ext value failed: %w", err)
			}
			extFields[string(key)] = string(value)
		}
	}

	return extFields, nil
}

type topicRouteData struct {
	QueueDataList  []queueData  `json:"queueDatas"`
	BrokerDataList []brokerData `json:"brokerDatas"`
}

type queueData struct {
	BrokerName string `json:"brokerName"`
}

type brokerData struct {
	BrokerName  string            `json:"brokerName"`
	BrokerAddrs map[string]string `json:"brokerAddrs"`
}

func parseBrokerAddrFromRouteBody(body []byte, brokerName string) (string, error) {
	var route topicRouteData
	if err := json.Unmarshal(body, &route); err != nil {
		return "", fmt.Errorf("unmarshal topic route data failed: %w", err)
	}

	for _, bd := range route.BrokerDataList {
		if bd.BrokerName == brokerName {
			if addr, ok := bd.BrokerAddrs["0"]; ok {
				return addr, nil
			}
			return "", fmt.Errorf("broker %s master (addr key '0') not found in route data", brokerName)
		}
	}

	return "", fmt.Errorf("broker %s not found in route data", brokerName)
}

func getQueueOffsetFromActionContext(actionCtx map[string]interface{}) int64 {
	v, ok := actionCtx[ActionContextKeyQueueOffset]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case json.Number:
		n, _ := val.Int64()
		return n
	default:
		return 0
	}
}
