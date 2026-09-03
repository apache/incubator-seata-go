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

package endphase_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	gettyremoting "seata.apache.org/seata-go/v2/pkg/remoting/getty"
	grpcremoting "seata.apache.org/seata-go/v2/pkg/remoting/grpc"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/tm"
	gettytm "seata.apache.org/seata-go/v2/pkg/tm/transaction/getty"
	grpctm "seata.apache.org/seata-go/v2/pkg/tm/transaction/grpc"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	cancel()
	return ctx
}

// A context that is already done stops the retry loop before the first attempt,
// so no request reaches the TC. Commit used to report that as success and
// Rollback used to panic on the nil response.
func TestSecondPhaseWithDoneContext(t *testing.T) {
	cases := []struct {
		name   string
		ctx    context.Context
		wantIs error
	}{
		{"canceled", canceledContext(), context.Canceled},
		{"deadline exceeded", expiredContext(), context.DeadlineExceeded},
	}

	for _, m := range transportsUnderTest() {
		for _, c := range cases {
			t.Run(m.name+"/commit/"+c.name, func(t *testing.T) {
				defer requireNoPanic(t)
				gtr := &tm.GlobalTransaction{Xid: "xid-1", TxRole: tm.Launcher}
				err := m.m.Commit(c.ctx, gtr)
				requireIs(t, err, c.wantIs)
			})
			t.Run(m.name+"/rollback/"+c.name, func(t *testing.T) {
				defer requireNoPanic(t)
				gtr := &tm.GlobalTransaction{Xid: "xid-1", TxRole: tm.Launcher}
				err := m.m.Rollback(c.ctx, gtr)
				requireIs(t, err, c.wantIs)
			})
		}
	}
}

// A participant does not drive the second phase, so it must keep returning nil
// even when the context is done.
func TestSecondPhaseIgnoredForParticipant(t *testing.T) {
	for _, m := range transportsUnderTest() {
		t.Run(m.name+"/commit", func(t *testing.T) {
			defer requireNoPanic(t)
			gtr := &tm.GlobalTransaction{Xid: "xid-1", TxRole: tm.Participant}
			if err := m.m.Commit(canceledContext(), gtr); err != nil {
				t.Fatalf("Commit() = %v, want nil for a participant", err)
			}
		})
		t.Run(m.name+"/rollback", func(t *testing.T) {
			defer requireNoPanic(t)
			gtr := &tm.GlobalTransaction{Xid: "xid-1", TxRole: tm.Participant}
			if err := m.m.Rollback(canceledContext(), gtr); err != nil {
				t.Fatalf("Rollback() = %v, want nil for a participant", err)
			}
		})
	}
}

func TestSecondPhaseWithEmptyXid(t *testing.T) {
	for _, m := range transportsUnderTest() {
		t.Run(m.name+"/commit", func(t *testing.T) {
			defer requireNoPanic(t)
			gtr := &tm.GlobalTransaction{TxRole: tm.Launcher}
			if err := m.m.Commit(context.Background(), gtr); err == nil {
				t.Fatal("Commit() = nil, want an error for an empty xid")
			}
		})
		t.Run(m.name+"/rollback", func(t *testing.T) {
			defer requireNoPanic(t)
			gtr := &tm.GlobalTransaction{TxRole: tm.Launcher}
			if err := m.m.Rollback(context.Background(), gtr); err == nil {
				t.Fatal("Rollback() = nil, want an error for an empty xid")
			}
		})
	}
}

func requireNoPanic(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("panicked: %v", r)
	}
}

func requireIs(t *testing.T, err, want error) {
	t.Helper()
	if err == nil {
		t.Fatalf("got nil error, want one matching %v", want)
	}
	if !errors.Is(err, want) {
		t.Fatalf("errors.Is(%v, %v) = false, want true", err, want)
	}
}

// The cases below drive the transports through every response shape the second
// phase has to survive. SendSyncRequest is patched the same way the existing
// transport tests do it, since the remoting clients are package level
// singletons with no injection point.
type shapeCase struct {
	name string
	// A nil result means the transport has no equivalent for the case.
	getty      func() (interface{}, error)
	grpc       func() (interface{}, error)
	wantErr    string
	wantStatus message.GlobalStatus
}

// SendSyncRequest is patched once for the whole test binary. Applying and
// resetting a gomonkey patch repeatedly does not reliably restore the original
// method, which let one test's canned response leak into the next, so the
// canned results are swapped through these variables instead.
var (
	gettyResult func() (interface{}, error)
	grpcResult  func() (interface{}, error)
)

func TestMain(m *testing.M) {
	log.Init()

	gettyStub := gomonkey.ApplyMethod(reflect.TypeOf(gettyremoting.GetGettyRemotingClient()), "SendSyncRequest",
		func(_ *gettyremoting.GettyRemotingClient, _ interface{}) (interface{}, error) {
			if gettyResult == nil {
				return nil, errors.New("getty transport called without a canned result")
			}
			return gettyResult()
		})
	grpcStub := gomonkey.ApplyMethod(reflect.TypeOf(grpcremoting.GetGrpcRemotingClient()), "SendSyncRequest",
		func(_ *grpcremoting.GrpcRemotingClient, _ interface{}) (interface{}, error) {
			if grpcResult == nil {
				return nil, errors.New("grpc transport called without a canned result")
			}
			return grpcResult()
		})

	code := m.Run()

	grpcStub.Reset()
	gettyStub.Reset()
	os.Exit(code)
}

// transportUnderTest is the single list of transports every case below runs
// against, so both are held to the same second phase contract.
type transportUnderTest struct {
	name string
	m    tm.GlobalTransactionManager
	// arm points the patched method at the canned result for one subtest.
	arm func(func() (interface{}, error))
}

func transportsUnderTest() []transportUnderTest {
	return []transportUnderTest{
		{
			name: "getty",
			m:    &gettytm.GettyGlobalTransactionManager{},
			arm:  func(r func() (interface{}, error)) { gettyResult = r },
		},
		{
			name: "grpc",
			m:    &grpctm.GrpcGlobalTransactionManager{},
			arm:  func(r func() (interface{}, error)) { grpcResult = r },
		},
	}
}

func runShapeCases(t *testing.T, cases []shapeCase, phase func(tm.GlobalTransactionManager, context.Context, *tm.GlobalTransaction) error) {
	t.Helper()
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                1,
		RollbackRetryCount:              1,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
	})

	for _, transport := range transportsUnderTest() {
		transport := transport
		for _, c := range cases {
			c := c
			canned := c.getty
			if transport.name == "grpc" {
				canned = c.grpc
			}
			if canned == nil {
				continue
			}
			t.Run(transport.name+"/"+c.name, func(t *testing.T) {
				defer requireNoPanic(t)
				transport.arm(canned)

				gtr := &tm.GlobalTransaction{Xid: "xid-1", TxRole: tm.Launcher}
				err := phase(transport.m, context.Background(), gtr)

				if c.wantErr == "" {
					if err != nil {
						t.Fatalf("second phase = %v, want nil", err)
					}
					if gtr.TxStatus != c.wantStatus {
						t.Fatalf("TxStatus = %v, want %v", gtr.TxStatus, c.wantStatus)
					}
					return
				}
				if err == nil {
					t.Fatalf("second phase = nil, want an error mentioning %q", c.wantErr)
				}
				if !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("second phase = %q, want it to mention %q", err, c.wantErr)
				}
				if gtr.TxStatus != 0 {
					t.Fatalf("TxStatus = %v, want it left untouched on failure", gtr.TxStatus)
				}
			})
		}
	}
}

func TestCommitResponseShapes(t *testing.T) {
	sendFailure := func() (interface{}, error) { return nil, errors.New("mock send failure") }
	noResponse := func() (interface{}, error) { return nil, nil }

	runShapeCases(t, []shapeCase{
		{
			name:    "retries exhausted",
			getty:   sendFailure,
			grpc:    sendFailure,
			wantErr: "mock send failure",
		},
		{
			name:    "no response",
			getty:   noResponse,
			grpc:    noResponse,
			wantErr: "empty second phase response",
		},
		{
			name:    "wrong response type",
			getty:   func() (interface{}, error) { return message.GlobalRollbackResponse{}, nil },
			grpc:    func() (interface{}, error) { return &pb.GlobalRollbackResponseProto{}, nil },
			wantErr: "unexpected global commit response type",
		},
		{
			// Protobuf getters return zero values for a missing message, so
			// without an explicit check the transaction would take on status
			// zero and report success.
			name:    "response missing its payload",
			grpc:    func() (interface{}, error) { return &pb.GlobalCommitResponseProto{}, nil },
			wantErr: "incomplete global commit response",
		},
		{
			name: "committed",
			getty: func() (interface{}, error) {
				return message.GlobalCommitResponse{AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
					GlobalStatus: message.GlobalStatusCommitted,
				}}, nil
			},
			grpc: func() (interface{}, error) {
				return &pb.GlobalCommitResponseProto{AbstractGlobalEndResponse: &pb.AbstractGlobalEndResponseProto{
					GlobalStatus: pb.GlobalStatusProto_Committed,
				}}, nil
			},
			wantStatus: message.GlobalStatusCommitted,
		},
	}, func(m tm.GlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
		return m.Commit(ctx, gtr)
	})
}

func TestRollbackResponseShapes(t *testing.T) {
	sendFailure := func() (interface{}, error) { return nil, errors.New("mock send failure") }
	noResponse := func() (interface{}, error) { return nil, nil }

	runShapeCases(t, []shapeCase{
		{
			name:    "retries exhausted",
			getty:   sendFailure,
			grpc:    sendFailure,
			wantErr: "mock send failure",
		},
		{
			name:    "no response",
			getty:   noResponse,
			grpc:    noResponse,
			wantErr: "empty second phase response",
		},
		{
			name:    "wrong response type",
			getty:   func() (interface{}, error) { return message.GlobalCommitResponse{}, nil },
			grpc:    func() (interface{}, error) { return &pb.GlobalCommitResponseProto{}, nil },
			wantErr: "unexpected global rollback response type",
		},
		{
			name:    "response missing its payload",
			grpc:    func() (interface{}, error) { return &pb.GlobalRollbackResponseProto{}, nil },
			wantErr: "incomplete global rollback response",
		},
		{
			name: "rolled back",
			getty: func() (interface{}, error) {
				return message.GlobalRollbackResponse{AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
					GlobalStatus: message.GlobalStatusRollbacked,
				}}, nil
			},
			grpc: func() (interface{}, error) {
				return &pb.GlobalRollbackResponseProto{AbstractGlobalEndResponse: &pb.AbstractGlobalEndResponseProto{
					GlobalStatus: pb.GlobalStatusProto_Rollbacked,
				}}, nil
			},
			wantStatus: message.GlobalStatusRollbacked,
		},
	}, func(m tm.GlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
		return m.Rollback(ctx, gtr)
	})
}
