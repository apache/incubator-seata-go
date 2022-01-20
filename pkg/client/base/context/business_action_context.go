package context

// BusinessActionContext store the tcc branch transaction context
type BusinessActionContext struct {
	*RootContext
	XID           string
	BranchID      int64
	ActionName    string
	ActionContext map[string]interface{}
	AsyncCommit   bool
}
