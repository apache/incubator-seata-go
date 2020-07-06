package dao

import (
	"github.com/dk-lockdown/seata-golang/client/at/exec"
	"github.com/dk-lockdown/seata-golang/client/context"
)

const (
	allocateInventorySql = `update seata_product.inventory set available_qty = available_qty - ?, 
		allocated_qty = allocated_qty + ? where product_sysno = ? and available_qty >= ?`
)

type Dao struct {
	*exec.DB
}

type AllocateInventoryReq struct {
	ProductSysNo int64
	Qty          int32
}

func (dao *Dao) AllocateInventory(ctx *context.RootContext, reqs []*AllocateInventoryReq) {
	tx, err := dao.Begin(ctx)
	if err != nil {
		panic(err)
	}
	for _, req := range reqs {
		_, err := tx.Exec(allocateInventorySql, req.Qty, req.Qty, req.ProductSysNo, req.Qty)
		if err != nil {
			tx.Rollback()
			panic(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		panic(err)
	}
}
