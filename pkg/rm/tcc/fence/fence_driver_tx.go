package fence

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/seata/seata-go/pkg/tm"
)

type FenceTx struct {
	Ctx           context.Context
	TargetTx      driver.Tx
	TargetFenceTx *sql.Tx
}

func (tx *FenceTx) Commit() error {
	if err := tx.TargetTx.Commit(); err != nil {
		return err
	}

	tx.clearFenceTx()
	return tx.TargetFenceTx.Commit()
}

func (tx *FenceTx) Rollback() error {
	if err := tx.TargetTx.Rollback(); err != nil {
		return err
	}

	tx.clearFenceTx()
	return tx.TargetFenceTx.Rollback()
}

func (tx *FenceTx) clearFenceTx() {
	tm.SetFenceTxBeginedFlag(tx.Ctx, false)
}
