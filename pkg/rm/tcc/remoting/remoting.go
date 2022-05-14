package remoting

import (
	"reflect"
)

type RemoteType byte
type ServiceType byte

const (
	RemoteTypeSofaRpc      RemoteType = 2
	RemoteTypeDubbo        RemoteType = 3
	RemoteTypeRestful      RemoteType = 4
	RemoteTypeLocalService RemoteType = 5
	RemoteTypeHsf          RemoteType = 8

	ServiceTypeReference ServiceType = 1
	ServiceTypeProvider  ServiceType = 2
)

type RemotingDesc struct {
	/**
	 * is referenc bean ?
	 */
	isReference bool

	/**
	 * rpc target bean, the service bean has this property
	 */
	targetBean interface{}

	/**
	 * the tcc interface tyep
	 */
	interfaceClass reflect.Type

	/**
	 * interface class name
	 */
	interfaceClassName string

	/**
	 * rpc uniqueId: hsf, dubbo's version, sofa-rpc's uniqueId
	 */
	uniqueId string

	/**
	 * dubbo/hsf 's group
	 */
	group string

	/**
	 * protocol: sofa-rpc, dubbo, injvm etc.
	 */
	protocol RemoteType
}

type RemotingParser interface {
	isRemoting(bean interface{}, beanName string) (bool, error)

	/**
	 * if it is reference bean ?
	 */
	isReference(bean interface{}, beanName string) (bool, error)

	/**
	 * if it is service bean ?
	 */
	isService(bean interface{}, beanName string) (bool, error)

	/**
	 * get the remoting bean info
	 */
	getServiceDesc(bean interface{}, beanName string) (RemotingDesc, error)

	/**
	 * the remoting protocol
	 */
	getProtocol() RemoteType
}
