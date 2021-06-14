package dao

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/exec"
	"github.com/transaction-wg/seata-golang/pkg/client/context"
)

const (
	allocateInventorySql = `update seata_product.inventory set available_qty = available_qty - ?, 
		allocated_qty = allocated_qty + ? where product_sysno = ? and available_qty >= ?`
)
const (
	allocateInventorySqlByPostgres = `update seata_product.inventory set available_qty = available_qty - $1, 
		allocated_qty = allocated_qty + $2 where product_sysno = $3 and available_qty >= $4`
)

type Dao struct {
	*exec.DB
}

type AllocateInventoryReq struct {
	ProductSysNo int64
	Qty          int32
}

func (dao *Dao) AllocateInventory(ctx *context.RootContext, reqs []*AllocateInventoryReq) error {
	tx, err := dao.Begin(ctx)
	if err != nil {
		return err
	}
	for _, req := range reqs {
		_, err := tx.Exec(allocateInventorySqlByPostgres, req.Qty, req.Qty, req.ProductSysNo, req.Qty)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
