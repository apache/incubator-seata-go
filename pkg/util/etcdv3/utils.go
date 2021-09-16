package etcdv3

// TODO: Import Standard
import (
	"fmt"
	"strconv"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
)

func IsAddressValid(addr registry.Address) bool {
	return len(addr.IP) != 0 || addr.Port != 0
}

func BuildRegistryKey(clusterName string, addr *registry.Address) string {
	return constant.Etcdv3RegistryPrefix + fmt.Sprintf("%s-%s", clusterName, addrToStr(addr))
}

func BuildRegistryValue(addr *registry.Address) string {
	return addrToStr(addr)
}

func addrToStr(addr *registry.Address) string {
	return addr.IP + ":" + strconv.FormatUint(addr.Port, 10)
}
