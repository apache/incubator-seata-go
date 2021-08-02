package constant

const (
	// NacosDefaultGroup
	NacosDefaultGroup  = "SEATA_GROUP"
	// NacosDefaultDataID
	NacosDefaultDataID = "seata"
	// NacosKey
	NacosKey = "nacos"
	// FileKey
	FileKey = "file"

	Etcdv3Key                = "etcdv3"
	Etcdv3RegistryPrefix     = "etcdv3-seata-" // according to seata java version
	Etcdv3LeaseRenewInterval = 5               // according to seata java version
	Etcdv3LeaseTtl           = 10              // according to seata java version
	Etcdv3LeaseTtlCritical   = 6               // according to seata java version
)
