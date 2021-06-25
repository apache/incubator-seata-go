package context

type BusinessActionContext struct {
	*RootContext
	XID           string
	BranchID      string
	ActionName    string
	ActionContext map[string]interface{}
}
