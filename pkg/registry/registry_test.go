package registry

import (
	"github.com/seata/seata-go/pkg/client"
	"net"
	"testing"
	"time"
)

func TestGetRegistry(t *testing.T) {
	cfg := client.LoadPath("/Users/ali/Desktop/GO/vader/seata-go/testdata/conf/seatago.yml")
	register, _ := GetRegistry(&cfg.RegistryConfig)
	address := net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 9002,
	}
	register.RegisterServiceInstance(address)
	//for i := 0; i < 10; i++ {
	time.Sleep(3 * time.Second)
	//}
}
