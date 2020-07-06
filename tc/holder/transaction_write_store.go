package holder

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type TransactionWriteStore struct {
	SessionRequest session.SessionStorable
	LogOperation   LogOperation
}

func (transactionWriteStore *TransactionWriteStore) Encode() ([]byte, error) {
	bySessionRequest, err := transactionWriteStore.SessionRequest.Encode()
	if err != nil {
		return nil, err
	}
	byOpCode := transactionWriteStore.LogOperation

	var result = make([]byte, 0)
	result = append(result, bySessionRequest...)
	result = append(result, byte(byOpCode))
	return result, nil
}

func (transactionWriteStore *TransactionWriteStore) Decode(src []byte) {
	bySessionRequest := src[:len(src)-1]
	byOpCode := src[len(src)-1:]

	transactionWriteStore.LogOperation = LogOperation(byOpCode[0])
	sessionRequest, _ := transactionWriteStore.getSessionInstanceByOperation()
	sessionRequest.Decode(bySessionRequest)
	transactionWriteStore.SessionRequest = sessionRequest
}

func (transactionWriteStore *TransactionWriteStore) getSessionInstanceByOperation() (session.SessionStorable, error) {
	var sessionStorable session.SessionStorable
	switch transactionWriteStore.LogOperation {
	case LogOperationGlobalAdd, LogOperationGlobalUpdate, LogOperationGlobalRemove:
		sessionStorable = session.NewGlobalSession()
		break
	case LogOperationBranchAdd, LogOperationBranchUpdate, LogOperationBranchRemove:
		sessionStorable = session.NewBranchSession()
		break
	default:
		return nil, errors.New("incorrect logOperation.")
	}
	return sessionStorable, nil
}
