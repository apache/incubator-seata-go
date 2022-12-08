package client

// RegisterProcessor register processor
func RegisterProcessor() {
	initHeartBeat()
	initOnResponse()
	initBranchCommit()
	initBranchRollback()
}
