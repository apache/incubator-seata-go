package common

import (
	"fmt"
	"strconv"
	"strings"
)

type XId struct {
	Port int
	IpAddress string
}

var XID = &XId{}

func (xId *XId) GenerateXID(tranId int64) string {
	return fmt.Sprintf("%s:%d:%d",xId.IpAddress,xId.Port,tranId)
}

func (xId *XId) GetTransactionId(xid string) int64 {
	if xid == "" { return -1 }

	idx := strings.LastIndex(xid,":")
	tranId,_ := strconv.ParseInt(xid[idx:],10,64)
	return tranId
}
