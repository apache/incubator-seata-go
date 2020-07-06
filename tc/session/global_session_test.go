package session

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestGlobalSession_Encode_Decode(t *testing.T) {
	gs := globalSessionProvider()
	result, _ := gs.Encode()

	newGs := &GlobalSession{}
	newGs.Decode(result)

	assert.Equal(t, newGs.TransactionId, gs.TransactionId)
	assert.Equal(t, newGs.Timeout, gs.Timeout)
	assert.Equal(t, newGs.ApplicationId, gs.ApplicationId)
	assert.Equal(t, newGs.TransactionServiceGroup, gs.TransactionServiceGroup)
	assert.Equal(t, newGs.TransactionName, gs.TransactionName)
}

func globalSessionProvider() *GlobalSession {
	gs := NewGlobalSession(
		WithGsApplicationId("demo-app"),
		WithGsTransactionServiceGroup("my_test_tx_group"),
		WithGsTransactionName("test"),
		WithGsTimeout(6000),
		WithGsActive(true),
	)

	return gs
}
