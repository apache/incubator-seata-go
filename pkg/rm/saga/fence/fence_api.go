package fence

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/seata/seata-go/pkg/rm/saga/fence/handler"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/enum"
	"github.com/seata/seata-go/pkg/tm"
)

// WithFence Execute the fence database operation first and then call back the business method
func WithFence(ctx context.Context, tx *sql.Tx, callback func() error) (err error) {
	if err = DoFence(ctx, tx); err != nil {
		return err
	}

	if err := callback(); err != nil {
		return fmt.Errorf("the business method error msg of: %p, [%w]", callback, err)
	}

	return
}

// DeFence This method is a suspended API interface that asserts the phase timing of a transaction
// and performs corresponding database operations to ensure transaction consistency
// case 1: if fencePhase is FencePhaseNotExist, will return a fence not found error.
// case 2: if fencePhase is FencePhaseAction, will do commit fence operation.
// case 3: if fencePhase is FencePhaseCompensationAction, will do rollback fence operation.
// case 4: if fencePhase not in above case, will return a fence phase illegal error.
func DoFence(ctx context.Context, tx *sql.Tx) error {
	hd := handler.GetSagaFenceHandler()
	phase := tm.GetFencePhase(ctx)

	switch phase {
	case enum.FencePhaseNotExist:
		return fmt.Errorf("xid %s, tx name %s, fence phase not exist", tm.GetXID(ctx), tm.GetTxName(ctx))
	case enum.FencePhaseAction:
		return hd.ActionFence(ctx, tx)
	case enum.FencePhaseCompensationAction:
		return hd.CompensationFence(ctx, tx)
	}

	return fmt.Errorf("fence phase: %v illegal", phase)
}
