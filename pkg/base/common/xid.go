package common

import (
	"fmt"
	"strconv"
	"strings"
)

type XID struct {
	IpAddress string
	Port      int
}

var xID = &XID{}

func GetXID() *XID {
	return xID
}

func (xid *XID) Init(ipAddress string, port int) {
	xid.IpAddress = ipAddress
	xid.Port = port
}

func (xid *XID) GenerateXID(tranID int64) string {
	return fmt.Sprintf("%s:%d:%d", xid.IpAddress, xid.Port, tranID)
}

func GetTransactionID(xid string) int64 {
	if xid == "" {
		return -1
	}

	idx := strings.LastIndex(xid, ":")
	if len(xid) == idx+1 {
		return -1
	}
	tranID, _ := strconv.ParseInt(xid[idx+1:], 10, 64)
	return tranID
}
