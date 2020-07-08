package exec

import (
	"database/sql"
	"time"
)

import (
	p "github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/parser/test_driver"
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	tx2 "github.com/dk-lockdown/seata-golang/client/at/proxy_tx"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser/mysql"
	"github.com/dk-lockdown/seata-golang/client/at/undo/manager"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
)

type Tx struct {
	proxyTx             *tx2.ProxyTx
	reportRetryCount    int
	reportSuccessEnable bool
	lockRetryInterval   time.Duration
	lockRetryTimes      int
}

func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var parser = p.New()
	act, _ := parser.ParseOneStmt(query, "", "")
	stmt, ok := act.(*ast.SelectStmt)
	if ok && stmt.LockTp == ast.SelectLockForUpdate {
		executor := &SelectForUpdateExecutor{
			proxyTx:       tx.proxyTx,
			sqlRecognizer: mysql.NewMysqlSelectForUpdateRecognizer(query, stmt),
			values:        args,
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
	branchId, err := tx.register()
	if err != nil {
		return errors.WithStack(err)
	}
	tx.proxyTx.Context.BranchId = branchId

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
		return tx.proxyTx.Commit()
	}
	return nil
}

func (tx *Tx) Rollback() error {
	err := tx.proxyTx.Rollback()
	if tx.proxyTx.Context.InGlobalTransaction() && tx.proxyTx.Context.IsBranchRegistered() {
		tx.report(false)
	}
	tx.proxyTx.Context.Reset()
	return err
}

func (tx *Tx) register() (int64, error) {
	var branchId int64
	var err error
	for retryCount := 0; retryCount < tx.lockRetryTimes; retryCount++ {
		branchId, err = dataSourceManager.BranchRegister(meta.BranchTypeAT, tx.proxyTx.ResourceId, "", tx.proxyTx.Context.Xid,
			nil, tx.proxyTx.Context.BuildLockKeys())
		if err == nil {
			break
		}
		logging.Logger.Errorf("branch register err: %v", err)
		var tex meta.TransactionException
		if errors.As(err, &tex) {
			if tex.Code == meta.TransactionExceptionCodeGlobalTransactionNotExist {
				break
			}
		}
		time.Sleep(tx.lockRetryInterval)
	}
	return branchId, err
}

func (tx *Tx) report(commitDone bool) error {
	retry := tx.reportRetryCount
	for retry > 0 {
		var err error
		if commitDone {
			err = dataSourceManager.BranchReport(meta.BranchTypeAT, tx.proxyTx.Context.Xid, tx.proxyTx.Context.BranchId,
				meta.BranchStatusPhaseoneDone, nil)
		} else {
			err = dataSourceManager.BranchReport(meta.BranchTypeAT, tx.proxyTx.Context.Xid, tx.proxyTx.Context.BranchId,
				meta.BranchStatusPhaseoneFailed, nil)
		}
		if err != nil {
			logging.Logger.Errorf("Failed to report [%d/%s] commit done [%t] Retry Countdown: %d",
				tx.proxyTx.Context.BranchId, tx.proxyTx.Context.Xid, commitDone, retry)
			retry = retry - 1
			if retry == 0 {
				return errors.WithMessagef(err, "Failed to report branch status %t", commitDone)
			}
		}
	}
	return nil
}
