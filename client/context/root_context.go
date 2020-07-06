package context

import (
	"context"
	"fmt"
	"strings"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
)

const (
	KEY_XID                  = "TX_XID"
	KEY_XID_INTERCEPTOR_TYPE = "tx-xid-interceptor-type"
	KEY_GLOBAL_LOCK_FLAG     = "TX_LOCK"
)

type RootContext struct {
	context.Context

	// like thread local map
	localMap map[string]interface{}
}

func NewRootContext(ctx context.Context) *RootContext {
	rootCtx := &RootContext{
		Context:  ctx,
		localMap: make(map[string]interface{}),
	}

	xId := ctx.Value(KEY_XID)
	if xId != nil {
		xid := xId.(string)
		rootCtx.Bind(xid)
	}
	return rootCtx
}

func (c *RootContext) Set(key string, value interface{}) {
	if c.localMap == nil {
		c.localMap = make(map[string]interface{})
	}
	c.localMap[key] = value
}

func (c *RootContext) Get(key string) (value interface{}, exists bool) {
	value, exists = c.localMap[key]
	return
}

func (c *RootContext) GetXID() string {
	xId := c.localMap[KEY_XID]
	xid, ok := xId.(string)
	if ok && xid != "" {
		return xid
	}

	xIdType := c.localMap[KEY_XID_INTERCEPTOR_TYPE]
	xidType, success := xIdType.(string)

	if success && xidType != "" && strings.Contains(xidType, "_") {
		return strings.Split(xidType, "_")[0]
	}

	return ""
}

func (c *RootContext) GetXIDInterceptorType() string {
	xIdType := c.localMap[KEY_XID_INTERCEPTOR_TYPE]
	xidType, _ := xIdType.(string)
	return xidType
}

func (c *RootContext) Bind(xid string) {
	logging.Logger.Debugf("bind %s", xid)
	c.Set(KEY_XID, xid)
}

func (c *RootContext) BindInterceptorType(xidType string) {
	if xidType != "" {
		xidTypes := strings.Split(xidType, "_")

		if len(xidTypes) == 2 {
			c.BindInterceptorTypeWithBranchType(xidTypes[0], meta.ValueOfBranchType(xidTypes[1]))
		}
	}
}

func (c *RootContext) BindInterceptorTypeWithBranchType(xid string, branchType meta.BranchType) {
	xidType := fmt.Sprintf("%s_%s", xid, branchType.String())
	logging.Logger.Debugf("bind interceptor type xid=%s branchType=%s", xid, branchType.String())
	c.Set(KEY_XID_INTERCEPTOR_TYPE, xidType)
}

func (c *RootContext) BindGlobalLockFlag() {
	logging.Logger.Debug("Local Transaction Global Lock support enabled")
	c.Set(KEY_GLOBAL_LOCK_FLAG, KEY_GLOBAL_LOCK_FLAG)
}

func (c *RootContext) Unbind() string {
	xId := c.localMap[KEY_XID]
	xid, ok := xId.(string)
	if ok && xid != "" {
		logging.Logger.Debugf("unbind %s", xid)
		delete(c.localMap, KEY_XID)
		return xid
	}
	return ""

}

func (c *RootContext) UnbindInterceptorType() string {
	xIdType := c.localMap[KEY_XID_INTERCEPTOR_TYPE]
	xidType, ok := xIdType.(string)
	if ok && xidType != "" {
		logging.Logger.Debugf("unbind inteceptor type %s", xidType)
		delete(c.localMap, KEY_XID_INTERCEPTOR_TYPE)
		return xidType
	}
	return ""
}

func (c *RootContext) UnbindGlobalLockFlag() {
	logging.Logger.Debug("unbind global lock flag")
	delete(c.localMap, KEY_GLOBAL_LOCK_FLAG)
}

func (c *RootContext) InGlobalTransaction() bool {
	return c.localMap[KEY_XID] != nil
}

func (c *RootContext) RequireGlobalLock() bool {
	_, exists := c.localMap[KEY_GLOBAL_LOCK_FLAG]
	return exists
}
