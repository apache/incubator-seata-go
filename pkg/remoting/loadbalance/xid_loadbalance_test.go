package loadbalance

import (
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/seata/seata-go/pkg/remoting/mock"
	"github.com/stretchr/testify/assert"
)

func TestXidLoadBalance(t *testing.T) {

	sessions := &sync.Map{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockTestSession(ctrl)
	m.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8000"
	})
	m.EXPECT().IsClosed().AnyTimes().DoAndReturn(func() bool {
		return false
	})
	sessions.Store(m, 8000)

	m = mock.NewMockTestSession(ctrl)
	m.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8001"
	})
	m.EXPECT().IsClosed().AnyTimes().DoAndReturn(func() bool {
		return true
	})
	sessions.Store(m, 8001)

	m = mock.NewMockTestSession(ctrl)
	m.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8002"
	})
	m.EXPECT().IsClosed().AnyTimes().DoAndReturn(func() bool {
		return false
	})
	sessions.Store(m, 8002)

	// test
	testCases := []struct {
		name        string
		sessions    *sync.Map
		xid         string
		returnAddrs []string
	}{
		{
			name:        "normal",
			sessions:    sessions,
			xid:         "127.0.0.1:8000:111",
			returnAddrs: []string{"127.0.0.1:8000"},
		},
		{
			name:        "xid is close",
			sessions:    sessions,
			xid:         "127.0.0.1:8001:111",
			returnAddrs: []string{"127.0.0.1:8000", "127.0.0.1:8002"},
		},
		{
			name:        "xid is not exist",
			sessions:    sessions,
			xid:         "127.0.0.1:9000:111",
			returnAddrs: []string{"127.0.0.1:8000", "127.0.0.1:8002"},
		},
	}
	for _, test := range testCases {
		session := XidLoadBalance(test.sessions, test.xid)
		assert.Contains(t, test.returnAddrs, session.RemoteAddr())
	}

}
