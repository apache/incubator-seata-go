package common

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	IPAddress string
	Port int
)

func Init(ipAddress string, port int) {
	IPAddress = ipAddress
	Port = port
}

func GenerateXID(tranID int64) string {
	return fmt.Sprintf("%s:%d:%d", IPAddress, Port, tranID)
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