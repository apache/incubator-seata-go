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

package mock

import (
	"errors"
	"net"
	"testing"
	"time"

	getty "github.com/apache/dubbo-getty"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// mockConn is a mock implementation of net.Conn for testing
type mockConn struct {
	net.Conn
}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// mockEndPoint is a mock implementation of getty.EndPoint
type mockEndPoint struct {
	getty.EndPoint
}

// mockEventListener is a mock implementation of getty.EventListener
type mockEventListener struct {
	getty.EventListener
}

// mockReadWriter is a mock implementation of getty.ReadWriter
type mockReadWriter struct {
	getty.ReadWriter
}

// mockReader is a mock implementation of getty.Reader
type mockReader struct {
	getty.Reader
}

// mockWriter is a mock implementation of getty.Writer
type mockWriter struct {
	getty.Writer
}

// mockSession is a mock implementation of getty.Session
type mockSession struct {
	getty.Session
}

func TestNewMockTestSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	assert.NotNil(t, mock, "NewMockTestSession should return a non-nil mock")
	assert.NotNil(t, mock.EXPECT(), "EXPECT should return a non-nil recorder")
}

func TestMockTestSession_EXPECT(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	recorder := mock.EXPECT()
	assert.NotNil(t, recorder, "EXPECT should return a non-nil recorder")
	assert.Equal(t, mock.recorder, recorder, "EXPECT should return the mock's recorder")
}

func TestMockTestSession_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().Close().Times(1)
	mock.Close()
}

func TestMockTestSession_Close_MultipleCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().Close().Times(3)
	mock.Close()
	mock.Close()
	mock.Close()
}

func TestMockTestSession_CloseConn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	waitTime := 5
	mock.EXPECT().CloseConn(waitTime).Times(1)
	mock.CloseConn(waitTime)
}

func TestMockTestSession_CloseConn_DifferentWaitTimes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().CloseConn(0).Times(1)
	mock.EXPECT().CloseConn(10).Times(1)
	mock.EXPECT().CloseConn(100).Times(1)

	mock.CloseConn(0)
	mock.CloseConn(10)
	mock.CloseConn(100)
}

func TestMockTestSession_Conn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedConn := &mockConn{}
	mock.EXPECT().Conn().Return(expectedConn).Times(1)

	conn := mock.Conn()
	assert.Equal(t, expectedConn, conn, "Conn should return the expected connection")
}

func TestMockTestSession_Conn_Nil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().Conn().Return(nil).Times(1)

	conn := mock.Conn()
	assert.Nil(t, conn, "Conn should return nil when configured to do so")
}

func TestMockTestSession_EndPoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedEndPoint := &mockEndPoint{}
	mock.EXPECT().EndPoint().Return(expectedEndPoint).Times(1)

	endPoint := mock.EndPoint()
	assert.Equal(t, expectedEndPoint, endPoint, "EndPoint should return the expected endpoint")
}

func TestMockTestSession_GetActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedTime := time.Now()
	mock.EXPECT().GetActive().Return(expectedTime).Times(1)

	activeTime := mock.GetActive()
	assert.Equal(t, expectedTime, activeTime, "GetActive should return the expected time")
}

func TestMockTestSession_GetAttribute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	key := "testKey"
	value := "testValue"
	mock.EXPECT().GetAttribute(key).Return(value).Times(1)

	result := mock.GetAttribute(key)
	assert.Equal(t, value, result, "GetAttribute should return the expected value")
}

func TestMockTestSession_GetAttribute_DifferentTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)

	// String key and value
	mock.EXPECT().GetAttribute("stringKey").Return("stringValue").Times(1)
	assert.Equal(t, "stringValue", mock.GetAttribute("stringKey"))

	// Integer key and value
	mock.EXPECT().GetAttribute(42).Return(100).Times(1)
	assert.Equal(t, 100, mock.GetAttribute(42))

	// Nil value
	mock.EXPECT().GetAttribute("nilKey").Return(nil).Times(1)
	assert.Nil(t, mock.GetAttribute("nilKey"))
}

func TestMockTestSession_ID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedID := uint32(12345)
	mock.EXPECT().ID().Return(expectedID).Times(1)

	id := mock.ID()
	assert.Equal(t, expectedID, id, "ID should return the expected session ID")
}

func TestMockTestSession_IncReadPkgNum(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().IncReadPkgNum().Times(1)
	mock.IncReadPkgNum()
}

func TestMockTestSession_IncWritePkgNum(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().IncWritePkgNum().Times(1)
	mock.IncWritePkgNum()
}

func TestMockTestSession_IsClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().IsClosed().Return(false).Times(1)
	mock.EXPECT().IsClosed().Return(true).Times(1)

	assert.False(t, mock.IsClosed(), "IsClosed should return false initially")
	assert.True(t, mock.IsClosed(), "IsClosed should return true after close")
}

func TestMockTestSession_LocalAddr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedAddr := "127.0.0.1:8080"
	mock.EXPECT().LocalAddr().Return(expectedAddr).Times(1)

	addr := mock.LocalAddr()
	assert.Equal(t, expectedAddr, addr, "LocalAddr should return the expected address")
}

func TestMockTestSession_RemoteAddr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedAddr := "192.168.1.1:9090"
	mock.EXPECT().RemoteAddr().Return(expectedAddr).Times(1)

	addr := mock.RemoteAddr()
	assert.Equal(t, expectedAddr, addr, "RemoteAddr should return the expected address")
}

func TestMockTestSession_ReadTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedTimeout := 30 * time.Second
	mock.EXPECT().ReadTimeout().Return(expectedTimeout).Times(1)

	timeout := mock.ReadTimeout()
	assert.Equal(t, expectedTimeout, timeout, "ReadTimeout should return the expected timeout")
}

func TestMockTestSession_WriteTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedTimeout := 60 * time.Second
	mock.EXPECT().WriteTimeout().Return(expectedTimeout).Times(1)

	timeout := mock.WriteTimeout()
	assert.Equal(t, expectedTimeout, timeout, "WriteTimeout should return the expected timeout")
}

func TestMockTestSession_RemoveAttribute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	key := "removeKey"
	mock.EXPECT().RemoveAttribute(key).Times(1)
	mock.RemoveAttribute(key)
}

func TestMockTestSession_Reset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().Reset().Times(1)
	mock.Reset()
}

func TestMockTestSession_Send(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	data := []byte("test data")
	expectedLen := len(data)
	mock.EXPECT().Send(data).Return(expectedLen, nil).Times(1)

	n, err := mock.Send(data)
	assert.Equal(t, expectedLen, n, "Send should return the expected length")
	assert.NoError(t, err, "Send should not return an error")
}

func TestMockTestSession_Send_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	data := []byte("test data")
	expectedErr := errors.New("send error")
	mock.EXPECT().Send(data).Return(0, expectedErr).Times(1)

	n, err := mock.Send(data)
	assert.Equal(t, 0, n, "Send should return 0 on error")
	assert.EqualError(t, err, expectedErr.Error(), "Send should return the expected error")
}

func TestMockTestSession_SetAttribute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	key := "attrKey"
	value := "attrValue"
	mock.EXPECT().SetAttribute(key, value).Times(1)
	mock.SetAttribute(key, value)
}

func TestMockTestSession_SetCompressType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	compressType := getty.CompressType(1)
	mock.EXPECT().SetCompressType(compressType).Times(1)
	mock.SetCompressType(compressType)
}

func TestMockTestSession_SetCronPeriod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	period := 1000
	mock.EXPECT().SetCronPeriod(period).Times(1)
	mock.SetCronPeriod(period)
}

func TestMockTestSession_SetEventListener(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	listener := &mockEventListener{}
	mock.EXPECT().SetEventListener(listener).Times(1)
	mock.SetEventListener(listener)
}

func TestMockTestSession_SetMaxMsgLen(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	maxLen := 4096
	mock.EXPECT().SetMaxMsgLen(maxLen).Times(1)
	mock.SetMaxMsgLen(maxLen)
}

func TestMockTestSession_SetName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	name := "testSession"
	mock.EXPECT().SetName(name).Times(1)
	mock.SetName(name)
}

func TestMockTestSession_SetPkgHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	handler := &mockReadWriter{}
	mock.EXPECT().SetPkgHandler(handler).Times(1)
	mock.SetPkgHandler(handler)
}

func TestMockTestSession_SetReadTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	timeout := 30 * time.Second
	mock.EXPECT().SetReadTimeout(timeout).Times(1)
	mock.SetReadTimeout(timeout)
}

func TestMockTestSession_SetReader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	reader := &mockReader{}
	mock.EXPECT().SetReader(reader).Times(1)
	mock.SetReader(reader)
}

func TestMockTestSession_SetSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	session := &mockSession{}
	mock.EXPECT().SetSession(session).Times(1)
	mock.SetSession(session)
}

func TestMockTestSession_SetWaitTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	waitTime := 5 * time.Second
	mock.EXPECT().SetWaitTime(waitTime).Times(1)
	mock.SetWaitTime(waitTime)
}

func TestMockTestSession_SetWriteTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	timeout := 60 * time.Second
	mock.EXPECT().SetWriteTimeout(timeout).Times(1)
	mock.SetWriteTimeout(timeout)
}

func TestMockTestSession_SetWriter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	writer := &mockWriter{}
	mock.EXPECT().SetWriter(writer).Times(1)
	mock.SetWriter(writer)
}

func TestMockTestSession_Stat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	expectedStat := "session stats"
	mock.EXPECT().Stat().Return(expectedStat).Times(1)

	stat := mock.Stat()
	assert.Equal(t, expectedStat, stat, "Stat should return the expected statistics")
}

func TestMockTestSession_UpdateActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().UpdateActive().Times(1)
	mock.UpdateActive()
}

func TestMockTestSession_WriteBytes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	data := []byte("test write bytes")
	expectedLen := len(data)
	mock.EXPECT().WriteBytes(data).Return(expectedLen, nil).Times(1)

	n, err := mock.WriteBytes(data)
	assert.Equal(t, expectedLen, n, "WriteBytes should return the expected length")
	assert.NoError(t, err, "WriteBytes should not return an error")
}

func TestMockTestSession_WriteBytes_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	data := []byte("test write bytes")
	expectedErr := errors.New("write error")
	mock.EXPECT().WriteBytes(data).Return(0, expectedErr).Times(1)

	n, err := mock.WriteBytes(data)
	assert.Equal(t, 0, n, "WriteBytes should return 0 on error")
	assert.EqualError(t, err, expectedErr.Error(), "WriteBytes should return the expected error")
}

func TestMockTestSession_WriteBytesArray(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	data1 := []byte("test1")
	data2 := []byte("test2")
	expectedLen := len(data1) + len(data2)
	mock.EXPECT().WriteBytesArray(data1, data2).Return(expectedLen, nil).Times(1)

	n, err := mock.WriteBytesArray(data1, data2)
	assert.Equal(t, expectedLen, n, "WriteBytesArray should return the expected length")
	assert.NoError(t, err, "WriteBytesArray should not return an error")
}

func TestMockTestSession_WriteBytesArray_EmptyArray(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	mock.EXPECT().WriteBytesArray().Return(0, nil).Times(1)

	n, err := mock.WriteBytesArray()
	assert.Equal(t, 0, n, "WriteBytesArray with empty array should return 0")
	assert.NoError(t, err, "WriteBytesArray should not return an error")
}

func TestMockTestSession_WritePkg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	pkg := "test package"
	timeout := 10 * time.Second
	expectedLen := 100
	expectedPkgNum := 1
	mock.EXPECT().WritePkg(pkg, timeout).Return(expectedLen, expectedPkgNum, nil).Times(1)

	length, pkgNum, err := mock.WritePkg(pkg, timeout)
	assert.Equal(t, expectedLen, length, "WritePkg should return the expected length")
	assert.Equal(t, expectedPkgNum, pkgNum, "WritePkg should return the expected package number")
	assert.NoError(t, err, "WritePkg should not return an error")
}

func TestMockTestSession_WritePkg_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	pkg := "test package"
	timeout := 10 * time.Second
	expectedErr := errors.New("write package error")
	mock.EXPECT().WritePkg(pkg, timeout).Return(0, 0, expectedErr).Times(1)

	length, pkgNum, err := mock.WritePkg(pkg, timeout)
	assert.Equal(t, 0, length, "WritePkg should return 0 length on error")
	assert.Equal(t, 0, pkgNum, "WritePkg should return 0 package number on error")
	assert.EqualError(t, err, expectedErr.Error(), "WritePkg should return the expected error")
}

// Integration tests for common usage patterns

func TestMockTestSession_FullSessionLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)

	// Setup session
	mock.EXPECT().SetName("testSession").Times(1)
	mock.EXPECT().SetReadTimeout(30 * time.Second).Times(1)
	mock.EXPECT().SetWriteTimeout(60 * time.Second).Times(1)
	mock.EXPECT().ID().Return(uint32(1)).Times(1)

	// Use session
	mock.EXPECT().IsClosed().Return(false).Times(1)
	mock.EXPECT().Send("data").Return(4, nil).Times(1)
	mock.EXPECT().UpdateActive().Times(1)

	// Close session
	mock.EXPECT().IsClosed().Return(true).Times(1)
	mock.EXPECT().Close().Times(1)

	// Execute lifecycle
	mock.SetName("testSession")
	mock.SetReadTimeout(30 * time.Second)
	mock.SetWriteTimeout(60 * time.Second)
	id := mock.ID()
	assert.Equal(t, uint32(1), id)

	isClosed := mock.IsClosed()
	assert.False(t, isClosed)

	n, err := mock.Send("data")
	assert.Equal(t, 4, n)
	assert.NoError(t, err)

	mock.UpdateActive()

	isClosed = mock.IsClosed()
	assert.True(t, isClosed)

	mock.Close()
}

func TestMockTestSession_AttributeManagement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)

	// Set attributes
	mock.EXPECT().SetAttribute("key1", "value1").Times(1)
	mock.EXPECT().SetAttribute("key2", 42).Times(1)

	// Get attributes
	mock.EXPECT().GetAttribute("key1").Return("value1").Times(1)
	mock.EXPECT().GetAttribute("key2").Return(42).Times(1)
	mock.EXPECT().GetAttribute("key3").Return(nil).Times(1)

	// Remove attribute
	mock.EXPECT().RemoveAttribute("key1").Times(1)
	mock.EXPECT().GetAttribute("key1").Return(nil).Times(1)

	// Execute operations
	mock.SetAttribute("key1", "value1")
	mock.SetAttribute("key2", 42)

	val1 := mock.GetAttribute("key1")
	assert.Equal(t, "value1", val1)

	val2 := mock.GetAttribute("key2")
	assert.Equal(t, 42, val2)

	val3 := mock.GetAttribute("key3")
	assert.Nil(t, val3)

	mock.RemoveAttribute("key1")
	val1After := mock.GetAttribute("key1")
	assert.Nil(t, val1After)
}

func TestMockTestSession_MultiplePackageWrites(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)

	// Setup expectations for multiple writes
	mock.EXPECT().IncWritePkgNum().Times(3)
	mock.EXPECT().WritePkg("pkg1", gomock.Any()).Return(10, 1, nil).Times(1)
	mock.EXPECT().WritePkg("pkg2", gomock.Any()).Return(20, 2, nil).Times(1)
	mock.EXPECT().WritePkg("pkg3", gomock.Any()).Return(30, 3, nil).Times(1)

	// Execute writes
	timeout := 5 * time.Second

	mock.IncWritePkgNum()
	len1, num1, err1 := mock.WritePkg("pkg1", timeout)
	assert.Equal(t, 10, len1)
	assert.Equal(t, 1, num1)
	assert.NoError(t, err1)

	mock.IncWritePkgNum()
	len2, num2, err2 := mock.WritePkg("pkg2", timeout)
	assert.Equal(t, 20, len2)
	assert.Equal(t, 2, num2)
	assert.NoError(t, err2)

	mock.IncWritePkgNum()
	len3, num3, err3 := mock.WritePkg("pkg3", timeout)
	assert.Equal(t, 30, len3)
	assert.Equal(t, 3, num3)
	assert.NoError(t, err3)
}

func TestMockTestSession_ConnectionInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)
	conn := &mockConn{}

	mock.EXPECT().Conn().Return(conn).Times(1)
	mock.EXPECT().LocalAddr().Return("127.0.0.1:8080").Times(1)
	mock.EXPECT().RemoteAddr().Return("192.168.1.100:9090").Times(1)

	actualConn := mock.Conn()
	assert.Equal(t, conn, actualConn)

	localAddr := mock.LocalAddr()
	assert.Equal(t, "127.0.0.1:8080", localAddr)

	remoteAddr := mock.RemoteAddr()
	assert.Equal(t, "192.168.1.100:9090", remoteAddr)
}

func TestMockTestSession_TimeoutConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockTestSession(ctrl)

	readTimeout := 30 * time.Second
	writeTimeout := 60 * time.Second
	waitTime := 5 * time.Second

	mock.EXPECT().SetReadTimeout(readTimeout).Times(1)
	mock.EXPECT().SetWriteTimeout(writeTimeout).Times(1)
	mock.EXPECT().SetWaitTime(waitTime).Times(1)

	mock.EXPECT().ReadTimeout().Return(readTimeout).Times(1)
	mock.EXPECT().WriteTimeout().Return(writeTimeout).Times(1)

	// Set timeouts
	mock.SetReadTimeout(readTimeout)
	mock.SetWriteTimeout(writeTimeout)
	mock.SetWaitTime(waitTime)

	// Verify timeouts
	actualReadTimeout := mock.ReadTimeout()
	assert.Equal(t, readTimeout, actualReadTimeout)

	actualWriteTimeout := mock.WriteTimeout()
	assert.Equal(t, writeTimeout, actualWriteTimeout)
}
