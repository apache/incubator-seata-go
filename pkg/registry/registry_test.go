package registry

//
//func initConfig() (RegistryService, error) {
//	path := client.LoadPath("../seata-go/testdata/conf/seatago.yml").RegistryConfig
//	return GetRegistry(&path)
//}
//func TestRegister(*testing.T) {
//	register, _ := initConfig()
//	address := net.TCPAddr{
//		IP:   net.IPv4zero,
//		Port: 9001,
//	}
//	register.RegisterServiceInstance(address)
//	//for i := 0; i < 10; i++ {
//	time.Sleep(1000 * time.Second)
//	//}
//}
//
//func TestDeregister(*testing.T) {
//	register, _ := initConfig()
//	address := net.TCPAddr{
//		IP:   net.IPv4zero,
//		Port: 9001,
//	}
//	register.DeRegisterServiceInstance(address)
//}
//
//func TestSubscribe(*testing.T) {
//	register, _ := initConfig()
//	register.Subscribe("DEFAULT", "GroupName:SEATA_GROUP")
//}
