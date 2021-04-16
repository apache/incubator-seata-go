package exec

import (
	"database/sql"
	"time"

	p "github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/parser/test_driver"
	"github.com/pkg/errors"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	tx2 "github.com/transaction-wg/seata-golang/pkg/client/at/proxy_tx"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sqlparser/mysql"
	"github.com/transaction-wg/seata-golang/pkg/client/at/undo/manager"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

type Tx struct {
	proxyTx             *tx2.ProxyTx
	reportRetryCount    int
	reportSuccessEnable bool
	lockRetryInterval   time.Duration
	lockRetryTimes      int
	dataSourceManager   *DataSourceManager
}

func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt(query, "", "")
	stmt, ok := act.(*ast.SelectStmt)
	if ok && stmt.LockTp == ast.SelectLockForUpdate {
		executor := &SelectForUpdateExecutor{
			proxyTx:           tx.proxyTx,
			sqlRecognizer:     mysql.NewMysqlSelectForUpdateRecognizer(query, stmt),
			values:            args,
			dataSourceManager: tx.dataSourceManager,
		}
		return executor.Execute(tx.lockRetryInterval, tx.lockRetryTimes)
	} else {
		return tx.proxyTx.Tx.Query(query, args)
	}
}

func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt(query, "", "")
	deleteStmt, isDelete := act.(*ast.DeleteStmt)
	if isDelete {
		executor := &DeleteExecutor{
			proxyTx:       tx.proxyTx,
			sqlRecognizer: mysql.NewMysqlDeleteRecognizer(query, deleteStmt),
			values:        args,
		}
		return executor.Execute()
	}

	insertStmt, isInsert := act.(*ast.InsertStmt)
	if isInsert {
		executor := &InsertExecutor{
			proxyTx:       tx.proxyTx,
			sqlRecognizer: mysql.NewMysqlInsertRecognizer(query, insertStmt),
			values:        args,
		}
		return executor.Execute()
	}

	updateStmt, isUpdate := act.(*ast.UpdateStmt)
	if isUpdate {
		executor := &UpdateExecutor{
			proxyTx:       tx.proxyTx,
			sqlRecognizer: mysql.NewMysqlUpdateRecognizer(query, updateStmt),
			values:        args,
		}
		return executor.Execute()
	}

	return tx.proxyTx.Tx.Exec(query, args)
}

func (tx *Tx) Commit() error {
	branchID, err := tx.register()
	if err != nil {
		err = tx.proxyTx.Rollback()
		return errors.WithStack(err)
	}
	tx.proxyTx.Context.BranchID = branchID

	if tx.proxyTx.Context.HasUndoLog() {
		err = manager.GetUndoLogManager().FlushUndoLogs(tx.proxyTx)
		if err != nil {
			err1 := tx.report(false)
			if err1 != nil {
				return errors.WithStack(err1)
			}
			return errors.WithStack(err)
		}
		err = tx.proxyTx.Commit()
		if err != nil {
			err1 := tx.report(false)
			if err1 != nil {
				return errors.WithStack(err1)
			}
			return errors.WithStack(err)
		}
	} else {
		log.Error("no undolog")
		return tx.proxyTx.Commit()
	}
	return nil
}

func (tx *Tx) Rollback() error {
	err := tx.proxyTx.Rollback()
	if tx.proxyTx.Context.InGlobalTransaction() {
		branchID, err := tx.register()
		if err != nil {
			return errors.WithStack(err)
		}
		tx.proxyTx.Context.BranchID = branchID
		tx.report(false)
	}
	tx.proxyTx.Context.Reset()
	return err
}

func (tx *Tx) register() (int64, error) {
	var branchID int64
	var err error
	for retryCount := 0; retryCount < tx.lockRetryTimes; retryCount++ {
		branchID, err = tx.dataSourceManager.BranchRegister(meta.BranchTypeAT, tx.proxyTx.ResourceID, "", tx.proxyTx.Context.XID,
			nil, tx.proxyTx.Context.BuildLockKeys())
		if err == nil {
			break
		}
		log.Errorf("branch register err: %v", err)
		var tex *meta.TransactionException
		if errors.As(err, &tex) {
			if tex.Code == meta.TransactionExceptionCodeGlobalTransactionNotExist {
				break
			}
		}
		time.Sleep(tx.lockRetryInterval)
	}
	return branchID, err
}

func (tx *Tx) report(commitDone bool) error {
	retry := tx.reportRetryCount
	for retry > 0 {
		var err error
		if commitDone {
			err = tx.dataSourceManager.BranchReport(meta.BranchTypeAT, tx.proxyTx.Context.XID, tx.proxyTx.Context.BranchID,
				meta.BranchStatusPhaseoneDone, nil)
		} else {
			err = tx.dataSourceManager.BranchReport(meta.BranchTypeAT, tx.proxyTx.Context.XID, tx.proxyTx.Context.BranchID,
				meta.BranchStatusPhaseoneFailed, nil)
		}
		if err != nil {
			log.Errorf("Failed to report [%d/%s] commit done [%t] Retry Countdown: %d",
				tx.proxyTx.Context.BranchID, tx.proxyTx.Context.XID, commitDone, retry)
		}
		retry = retry - 1
		if retry == 0 {
			return errors.WithMessagef(err, "Failed to report branch status %t", commitDone)
		}
	}
	return nil
}
