package discovery

import (
	"fmt"
	"testing"
)

func Test_Lookup(t *testing.T) {
	err := RegisterService("mysql_service", "localhost", 3306)
	if err != nil {
		fmt.Println(err)
	}
	c := &ConsulConfig{
		Cluster:    "default",
		ServerAddr: "localhost:8500",
	}
	cr := newConsulRegistryService(c, nil)
	instances, err := cr.Lookup("mysql_service")
	if err != nil {
		fmt.Println(err)
	}
	for _, ins := range instances {
		fmt.Println(ins)
	}
}

func TestNewWatchPlan(t *testing.T) {

}
