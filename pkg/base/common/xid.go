package common

import (
	"fmt"
	"strconv"
	"strings"
)

type XId struct {
	IpAddress string
	Port      int
}

var XID = &XId{}

func (xId *XId) Init(ipAddress string, port int) {
	xId.IpAddress = ipAddress
	xId.Port = port
}

func (xId *XId) GenerateXID(tranId int64) string {
	return fmt.Sprintf("%s:%d:%d", xId.IpAddress, xId.Port, tranId)
}

func (xId *XId) GetTransactionId(xid string) int64 {
	if xid == "" {
		return -1
	}

	idx := strings.LastIndex(xid, ":")
	if len(xid) == idx+1 {
		return -1
	}
	tranId, _ := strconv.ParseInt(xid[idx+1:], 10, 64)
	return tranId
}
