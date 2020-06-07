package exec

import (
	"database/sql"
)

import (
	p "github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/parser/test_driver"
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser/mysql"
	tx2 "github.com/dk-lockdown/seata-golang/client/at/tx"
	"github.com/dk-lockdown/seata-golang/client/at/undo/manager"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
)

type Tx struct {
	tx *tx2.ProxyTx
	reportRetryCount int
	reportSuccessEnable bool
}

func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var parser = p.New()
	act,_ := parser.ParseOneStmt(query,"","")
	stmt,ok := act.(*ast.SelectStmt)
	if ok && stmt.LockTp == ast.SelectLockForUpdate {
		executor := &SelectForUpdateExecutor{
			tx:            tx.tx,
			sqlRecognizer: mysql.NewMysqlSelectForUpdateRecognizer(query,stmt),
			values:        args,
		}
		return executor.Execute()
	} else {
		return tx.tx.Tx.Query(query,args)
	}
}

func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	var parser = p.New()
	act,_ := parser.ParseOneStmt(query,"","")
	deleteStmt,isDelete := act.(*ast.DeleteStmt)
	if isDelete {
		executor := &DeleteExecutor{
			tx:            tx.tx,
			sqlRecognizer: mysql.NewMysqlDeleteRecognizer(query,deleteStmt),
			values:        args,
		}
		return executor.Execute()
	}

	insertStmt,isInsert := act.(*ast.InsertStmt)
	if isInsert {
		executor := &InsertExecutor{
			tx:            tx.tx,
			sqlRecognizer: mysql.NewMysqlInsertRecognizer(query,insertStmt),
			values:        args,
		}
		return executor.Execute()
	}

	updateStmt,isUpdate := act.(*ast.UpdateStmt)
	if isUpdate {
		executor := &UpdateExecutor{
			tx:            tx.tx,
			sqlRecognizer: mysql.NewMysqlUpdateRecognizer(query,updateStmt),
			values:        args,
		}
		return executor.Execute()
	}

	return tx.tx.Tx.Exec(query,args)
}

func (tx *Tx) Commit() error {
	branchId,err := tx.register()
	if err != nil {
		return errors.WithStack(err)
	}
	tx.tx.Context.BranchId = branchId

	if tx.tx.Context.HasUndoLog() {
		err = manager.GetUndoLogManager().FlushUndoLogs(tx.tx)
		if err != nil {
			err1 := tx.report(false)
			if err1 != nil {
				return errors.WithStack(err1)
			}
			return errors.WithStack(err)
		}
		err = tx.tx.Commit()
		if err != nil {
			err1 := tx.report(false)
			if err1 != nil {
				return errors.WithStack(err1)
			}
			return errors.WithStack(err)
		}
	} else {
		return tx.tx.Commit()
	}
	if tx.reportSuccessEnable {
		tx.report(true)
	}
	tx.tx.Context.Reset()
	return nil
}

func (tx *Tx) Rollback() error {
	err := tx.tx.Rollback()
	if tx.tx.Context.InGlobalTransaction() && tx.tx.Context.IsBranchRegistered() {
		tx.report(false)
	}
	tx.tx.Context.Reset()
	return err
}

func (tx *Tx) register() (int64,error) {
	return dataSourceManager.BranchRegister(meta.BranchTypeAT,tx.tx.ResourceId,"",tx.tx.Context.Xid,
		nil,tx.tx.Context.BuildLockKeys())
}

func (tx *Tx) report(commitDone bool) error {
	retry := tx.reportRetryCount
	for retry > 0 {
		var err error
		if commitDone {
			err = dataSourceManager.BranchReport(meta.BranchTypeAT, tx.tx.Context.Xid, tx.tx.Context.BranchId,
				meta.BranchStatusPhaseoneDone,nil)
		} else {
			err = dataSourceManager.BranchReport(meta.BranchTypeAT, tx.tx.Context.Xid, tx.tx.Context.BranchId,
				meta.BranchStatusPhaseoneFailed,nil)
		}
		if err != nil {
			logging.Logger.Errorf("Failed to report [%d/%s] commit done [%t] Retry Countdown: %d",
				tx.tx.Context.BranchId,tx.tx.Context.Xid,commitDone,retry)
			retry = retry -1
			if retry == 0 {
				return errors.WithMessagef(err,"Failed to report branch status %t",commitDone)
			}
		}
	}
	return nil
}