// Code generated by MockGen. DO NOT EDIT.
// Source: test_driver.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	driver "database/sql/driver"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockTestDriverConnector is a mock of TestDriverConnector interface
type MockTestDriverConnector struct {
	ctrl     *gomock.Controller
	recorder *MockTestDriverConnectorMockRecorder
}

// MockTestDriverConnectorMockRecorder is the mock recorder for MockTestDriverConnector
type MockTestDriverConnectorMockRecorder struct {
	mock *MockTestDriverConnector
}

// NewMockTestDriverConnector creates a new mock instance
func NewMockTestDriverConnector(ctrl *gomock.Controller) *MockTestDriverConnector {
	mock := &MockTestDriverConnector{ctrl: ctrl}
	mock.recorder = &MockTestDriverConnectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTestDriverConnector) EXPECT() *MockTestDriverConnectorMockRecorder {
	return m.recorder
}

// Connect mocks base method
func (m *MockTestDriverConnector) Connect(arg0 context.Context) (driver.Conn, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect", arg0)
	ret0, _ := ret[0].(driver.Conn)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Connect indicates an expected call of Connect
func (mr *MockTestDriverConnectorMockRecorder) Connect(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockTestDriverConnector)(nil).Connect), arg0)
}

// Driver mocks base method
func (m *MockTestDriverConnector) Driver() driver.Driver {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Driver")
	ret0, _ := ret[0].(driver.Driver)
	return ret0
}

// Driver indicates an expected call of Driver
func (mr *MockTestDriverConnectorMockRecorder) Driver() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Driver", reflect.TypeOf((*MockTestDriverConnector)(nil).Driver))
}

// MockTestDriverConn is a mock of TestDriverConn interface
type MockTestDriverConn struct {
	ctrl     *gomock.Controller
	recorder *MockTestDriverConnMockRecorder
}

// MockTestDriverConnMockRecorder is the mock recorder for MockTestDriverConn
type MockTestDriverConnMockRecorder struct {
	mock *MockTestDriverConn
}

// NewMockTestDriverConn creates a new mock instance
func NewMockTestDriverConn(ctrl *gomock.Controller) *MockTestDriverConn {
	mock := &MockTestDriverConn{ctrl: ctrl}
	mock.recorder = &MockTestDriverConnMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTestDriverConn) EXPECT() *MockTestDriverConnMockRecorder {
	return m.recorder
}

// Prepare mocks base method
func (m *MockTestDriverConn) Prepare(query string) (driver.Stmt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Prepare", query)
	ret0, _ := ret[0].(driver.Stmt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Prepare indicates an expected call of Prepare
func (mr *MockTestDriverConnMockRecorder) Prepare(query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Prepare", reflect.TypeOf((*MockTestDriverConn)(nil).Prepare), query)
}

// Close mocks base method
func (m *MockTestDriverConn) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockTestDriverConnMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockTestDriverConn)(nil).Close))
}

// Begin mocks base method
func (m *MockTestDriverConn) Begin() (driver.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Begin")
	ret0, _ := ret[0].(driver.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Begin indicates an expected call of Begin
func (mr *MockTestDriverConnMockRecorder) Begin() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Begin", reflect.TypeOf((*MockTestDriverConn)(nil).Begin))
}

// BeginTx mocks base method
func (m *MockTestDriverConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BeginTx", ctx, opts)
	ret0, _ := ret[0].(driver.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BeginTx indicates an expected call of BeginTx
func (mr *MockTestDriverConnMockRecorder) BeginTx(ctx, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BeginTx", reflect.TypeOf((*MockTestDriverConn)(nil).BeginTx), ctx, opts)
}

// Ping mocks base method
func (m *MockTestDriverConn) Ping(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ping", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Ping indicates an expected call of Ping
func (mr *MockTestDriverConnMockRecorder) Ping(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ping", reflect.TypeOf((*MockTestDriverConn)(nil).Ping), ctx)
}

// PrepareContext mocks base method
func (m *MockTestDriverConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PrepareContext", ctx, query)
	ret0, _ := ret[0].(driver.Stmt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PrepareContext indicates an expected call of PrepareContext
func (mr *MockTestDriverConnMockRecorder) PrepareContext(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrepareContext", reflect.TypeOf((*MockTestDriverConn)(nil).PrepareContext), ctx, query)
}

// Query mocks base method
func (m *MockTestDriverConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Query", query, args)
	ret0, _ := ret[0].(driver.Rows)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Query indicates an expected call of Query
func (mr *MockTestDriverConnMockRecorder) Query(query, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockTestDriverConn)(nil).Query), query, args)
}

// QueryContext mocks base method
func (m *MockTestDriverConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryContext", ctx, query, args)
	ret0, _ := ret[0].(driver.Rows)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryContext indicates an expected call of QueryContext
func (mr *MockTestDriverConnMockRecorder) QueryContext(ctx, query, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryContext", reflect.TypeOf((*MockTestDriverConn)(nil).QueryContext), ctx, query, args)
}

// Exec mocks base method
func (m *MockTestDriverConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exec", query, args)
	ret0, _ := ret[0].(driver.Result)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exec indicates an expected call of Exec
func (mr *MockTestDriverConnMockRecorder) Exec(query, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exec", reflect.TypeOf((*MockTestDriverConn)(nil).Exec), query, args)
}

// ExecContext mocks base method
func (m *MockTestDriverConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecContext", ctx, query, args)
	ret0, _ := ret[0].(driver.Result)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecContext indicates an expected call of ExecContext
func (mr *MockTestDriverConnMockRecorder) ExecContext(ctx, query, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecContext", reflect.TypeOf((*MockTestDriverConn)(nil).ExecContext), ctx, query, args)
}

// ResetSession mocks base method
func (m *MockTestDriverConn) ResetSession(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetSession", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// ResetSession indicates an expected call of ResetSession
func (mr *MockTestDriverConnMockRecorder) ResetSession(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetSession", reflect.TypeOf((*MockTestDriverConn)(nil).ResetSession), ctx)
}

// MockTestDriverStmt is a mock of TestDriverStmt interface
type MockTestDriverStmt struct {
	ctrl     *gomock.Controller
	recorder *MockTestDriverStmtMockRecorder
}

// MockTestDriverStmtMockRecorder is the mock recorder for MockTestDriverStmt
type MockTestDriverStmtMockRecorder struct {
	mock *MockTestDriverStmt
}

// NewMockTestDriverStmt creates a new mock instance
func NewMockTestDriverStmt(ctrl *gomock.Controller) *MockTestDriverStmt {
	mock := &MockTestDriverStmt{ctrl: ctrl}
	mock.recorder = &MockTestDriverStmtMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTestDriverStmt) EXPECT() *MockTestDriverStmtMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockTestDriverStmt) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockTestDriverStmtMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockTestDriverStmt)(nil).Close))
}

// NumInput mocks base method
func (m *MockTestDriverStmt) NumInput() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NumInput")
	ret0, _ := ret[0].(int)
	return ret0
}

// NumInput indicates an expected call of NumInput
func (mr *MockTestDriverStmtMockRecorder) NumInput() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NumInput", reflect.TypeOf((*MockTestDriverStmt)(nil).NumInput))
}

// Exec mocks base method
func (m *MockTestDriverStmt) Exec(args []driver.Value) (driver.Result, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exec", args)
	ret0, _ := ret[0].(driver.Result)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exec indicates an expected call of Exec
func (mr *MockTestDriverStmtMockRecorder) Exec(args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exec", reflect.TypeOf((*MockTestDriverStmt)(nil).Exec), args)
}

// Query mocks base method
func (m *MockTestDriverStmt) Query(args []driver.Value) (driver.Rows, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Query", args)
	ret0, _ := ret[0].(driver.Rows)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Query indicates an expected call of Query
func (mr *MockTestDriverStmtMockRecorder) Query(args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockTestDriverStmt)(nil).Query), args)
}

// QueryContext mocks base method
func (m *MockTestDriverStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryContext", ctx, args)
	ret0, _ := ret[0].(driver.Rows)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryContext indicates an expected call of QueryContext
func (mr *MockTestDriverStmtMockRecorder) QueryContext(ctx, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryContext", reflect.TypeOf((*MockTestDriverStmt)(nil).QueryContext), ctx, args)
}

// ExecContext mocks base method
func (m *MockTestDriverStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecContext", ctx, args)
	ret0, _ := ret[0].(driver.Result)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecContext indicates an expected call of ExecContext
func (mr *MockTestDriverStmtMockRecorder) ExecContext(ctx, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecContext", reflect.TypeOf((*MockTestDriverStmt)(nil).ExecContext), ctx, args)
}

// MockTestDriverTx is a mock of TestDriverTx interface
type MockTestDriverTx struct {
	ctrl     *gomock.Controller
	recorder *MockTestDriverTxMockRecorder
}

// MockTestDriverTxMockRecorder is the mock recorder for MockTestDriverTx
type MockTestDriverTxMockRecorder struct {
	mock *MockTestDriverTx
}

// NewMockTestDriverTx creates a new mock instance
func NewMockTestDriverTx(ctrl *gomock.Controller) *MockTestDriverTx {
	mock := &MockTestDriverTx{ctrl: ctrl}
	mock.recorder = &MockTestDriverTxMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTestDriverTx) EXPECT() *MockTestDriverTxMockRecorder {
	return m.recorder
}

// Commit mocks base method
func (m *MockTestDriverTx) Commit() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Commit")
	ret0, _ := ret[0].(error)
	return ret0
}

// Commit indicates an expected call of Commit
func (mr *MockTestDriverTxMockRecorder) Commit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Commit", reflect.TypeOf((*MockTestDriverTx)(nil).Commit))
}

// Rollback mocks base method
func (m *MockTestDriverTx) Rollback() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rollback")
	ret0, _ := ret[0].(error)
	return ret0
}

// Rollback indicates an expected call of Rollback
func (mr *MockTestDriverTxMockRecorder) Rollback() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rollback", reflect.TypeOf((*MockTestDriverTx)(nil).Rollback))
}
