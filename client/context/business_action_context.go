package context

type BusinessActionContext struct {
	*RootContext
	Xid           string
	BranchId      string
	ActionName    string
	ActionContext map[string]interface{}
}
