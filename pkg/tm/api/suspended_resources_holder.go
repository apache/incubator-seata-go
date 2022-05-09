package api

type SuspendedResourcesHolder struct {
	Xid string
}

func NewSuspendedResourcesHolder(xid string) SuspendedResourcesHolder {
	return SuspendedResourcesHolder{
		Xid: xid,
	}
}
