package main

import (
	"context"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/seata/seata-go/pkg/tm"
)

// 建议以下代码放到单独的文件里
// ................//

// ZormGlobalTransaction 包装seata-go的*tm.GlobalTransactionManager,实现zorm.IGlobalTransaction接口
type ZormGlobalTransaction struct {
	*tm.GlobalTransactionManager
}

// MyFuncGlobalTransaction zorm适配seata-go全局分布式事务的函数
// 重要!!!!需要配置zorm.DataSourceConfig.FuncGlobalTransaction=MyFuncGlobalTransaction 重要!!!
func MyFuncGlobalTransaction(ctx context.Context) (zorm.IGlobalTransaction, context.Context, context.Context, error) {
	// 创建seata-go事务
	globalTx := tm.GetGlobalTransactionManager()
	// 使用zorm.IGlobalTransaction接口对象包装分布式事务,隔离seata-go依赖
	globalTransaction := &ZormGlobalTransaction{globalTx}

	if tm.IsSeataContext(ctx) {
		return globalTransaction, ctx, ctx, nil
	}
	// open global transaction for the first time
	ctx = tm.InitSeataContext(ctx)
	// 有请求传入,手动获取的XID
	xidObj := ctx.Value("XID")
	if xidObj != nil {
		xid := xidObj.(string)
		tm.SetXID(ctx, xid)
	}
	tm.SetTxName(ctx, "ATSampleLocalGlobalTx")

	// use new context to process current global transaction.
	if tm.IsGlobalTx(ctx) {
		globalRootContext := transferTx(ctx)
		return globalTransaction, ctx, globalRootContext, nil
	}
	return globalTransaction, ctx, ctx, nil
}

// 实现zorm.IGlobalTransaction 托管全局分布式事务接口
// BeginGTX 开启全局分布式事务
func (gtx *ZormGlobalTransaction) BeginGTX(ctx context.Context, globalRootContext context.Context) error {
	//tm.SetTxStatus(globalRootContext, message.GlobalStatusBegin)
	err := gtx.Begin(globalRootContext, time.Second*30)
	return err
}

// CommitGTX 提交全局分布式事务
func (gtx *ZormGlobalTransaction) CommitGTX(ctx context.Context, globalRootContext context.Context) error {
	gtr := tm.GetTx(globalRootContext)
	return gtx.Commit(globalRootContext, gtr)
}

// RollbackGTX 回滚全局分布式事务
func (gtx *ZormGlobalTransaction) RollbackGTX(ctx context.Context, globalRootContext context.Context) error {
	gtr := tm.GetTx(globalRootContext)
	// 如果是Participant角色,修改为Launcher角色,允许分支事务提交全局事务.
	if gtr.TxRole != tm.Launcher {
		gtr.TxRole = tm.Launcher
	}
	return gtx.Rollback(globalRootContext, gtr)
}

// GetGTXID 获取全局分布式事务的XID
func (gtx *ZormGlobalTransaction) GetGTXID(ctx context.Context, globalRootContext context.Context) (string, error) {
	return tm.GetXID(globalRootContext), nil
}

// transferTx transfer the gtx into a new ctx from old ctx.
// use it to implement suspend and resume instead of seata java
func transferTx(ctx context.Context) context.Context {
	newCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(newCtx, tm.GetXID(ctx))
	return newCtx
}

// ................//
