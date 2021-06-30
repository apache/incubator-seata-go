package session

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/transaction-wg/seata-golang/pkg/testutil"
)

func TestGlobalSession_Encode_Decode(t *testing.T) {

	// gs.Encode() needs a valid server config
	testutil.InitServerConfig(t)

	gs := globalSessionProvider()
	result, err := gs.Encode()
	assert.NoError(t, err, "Encode() should success")

	newGs := &GlobalSession{}
	newGs.Decode(result)

	assert.Equal(t, newGs.TransactionID, gs.TransactionID)
	assert.Equal(t, newGs.Timeout, gs.Timeout)
	assert.Equal(t, newGs.ApplicationID, gs.ApplicationID)
	assert.Equal(t, newGs.TransactionServiceGroup, gs.TransactionServiceGroup)
	assert.Equal(t, newGs.TransactionName, gs.TransactionName)
}

func globalSessionProvider() *GlobalSession {
	gs := NewGlobalSession(
		WithGsApplicationID("demo-cmd"),
		WithGsTransactionServiceGroup("my_test_tx_group"),
		WithGsTransactionName("test"),
		WithGsTimeout(6000),
		WithGsActive(true),
	)

	return gs
}
