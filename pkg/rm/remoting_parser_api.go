package rm

const (
	RemotingUnknow  = 0
	RemotingGrpc    = 1
	RemotingDubbogo = 2
	RemotingLocal   = 3
)

type RemotingParse interface {
	IsService(target interface{}) bool
	IsReference(target interface{}) bool
	IsRemoting(target interface{}) bool
	ParseService(target interface{}) (*TwoPhaseAction, error)
	ParseReference(target interface{}) (*TwoPhaseAction, error)
	ParseTwoPhaseActionByInterface(target interface{}) (*TwoPhaseAction, error)
	GetRemotingType(target interface{}) int
}
