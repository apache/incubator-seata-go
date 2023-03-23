package registry

import (
	"github.com/seata/seata-go/pkg/client"
	"testing"
	"time"
)

func TestGetRegistry(t *testing.T) {
	cfg := client.LoadPath("../../testdata/conf/seatago.yml")
	register, _ := GetRegistry(&cfg.RegistryConfig)
	register.RegisterServiceInstance(cfg.RegistryConfig)
	for i := 0; i < 10; i++ {
		time.Sleep(10000)
	}
}
