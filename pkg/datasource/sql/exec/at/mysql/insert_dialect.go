package mysql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/arana-db/parser/ast"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/internal"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/util/log"
)

// SelectSqlBuilder build select sql string(for different db type)
type SelectSqlBuilder interface {
	BuildSelectSqlString(
		selectFields string,
		tableName string,
		where string,
		orderBy string,
		limit string,
		lockType ast.SelectLockType,
	) string
}

type InsertExecutor struct {
	*internal.InsertExecutor
}

func pkValuesMapMerge(dest *map[string][]interface{}, src map[string][]interface{}) {
	for k, v := range src {
		tmpK := k
		tmpV := v
		(*dest)[tmpK] = append((*dest)[tmpK], tmpV)
	}
}

// NewInsertExecutor get insert Executor
func NewInsertExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	mysqlInsertExecutor := &InsertExecutor{}
	baseInsertExecutor := internal.NewInsertExecutor(parserCtx, execContext, hooks, mysqlInsertExecutor)
	mysqlInsertExecutor.InsertExecutor = baseInsertExecutor
	return mysqlInsertExecutor
}

func (i *InsertExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	return i.InsertExecutor.ExecContext(ctx, f)
}

func (i *InsertExecutor) GetPkValues(ctx context.Context, execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error) {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	pkValuesMap := make(map[string][]interface{})
	var err error
	// when there is only one pk in the table
	if len(pkColumnNameList) == 1 {
		if i.ContainsPK(meta, parseCtx) {
			// the insert sql contain pk value
			pkValuesMap, err = i.GetPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else if internal.ContainsColumns(parseCtx) {
			// the insert table pk auto generated
			pkValuesMap, err = i.GetPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else {
			pkValuesMap, err = i.GetPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// when there is multiple pk in the table
		// 1,all pk columns are filled value.
		// 2,the auto increment pk column value is null, and other pk value are not null.
		pkValuesMap, err = i.GetPkValuesByColumn(ctx, execCtx)
		if err != nil {
			return nil, err
		}
		for _, columnName := range pkColumnNameList {
			if _, ok := pkValuesMap[columnName]; !ok {
				curPkValuesMap, err := i.GetPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValuesMap)
			}
		}
	}
	return pkValuesMap, nil
}

func (i *InsertExecutor) GetPkValuesByColumn(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	// getPkValuesByColumn get pk value by column.
	if !i.IsAstStmtValid() {
		return nil, nil
	}

	meta, err := i.GetMetaData(ctx)
	if err != nil {
		return nil, err
	}
	pkValuesMap, err := i.ParsePkValuesFromStatement(i.ParserCtx.InsertStmt, *meta, execCtx.NamedValues)
	if err != nil {
		return nil, err
	}

	// generate pkValue by auto increment
	for _, v := range pkValuesMap {
		tmpV := v
		if len(tmpV) == 1 {
			// pk auto generated while single insert primary key is expression
			if _, ok := tmpV[0].(*ast.FuncCallExpr); ok {
				curPkValueMap, err := i.GetPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValueMap)
			}
		} else if len(tmpV) > 0 && tmpV[0] == nil {
			// pk auto generated while column exists and value is null
			curPkValueMap, err := i.GetPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
			pkValuesMapMerge(&pkValuesMap, curPkValueMap)
		}
	}
	return pkValuesMap, nil
}

func (i *InsertExecutor) GetPkValuesByAuto(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !i.IsAstStmtValid() {
		return nil, nil
	}

	metaData, err := i.GetMetaData(ctx)
	if err != nil {
		return nil, err
	}

	pkValuesMap := make(map[string][]interface{})
	pkMetaMap := metaData.GetPrimaryKeyMap()
	if len(pkMetaMap) == 0 {
		return nil, fmt.Errorf("pk map is empty")
	}
	var autoColumnName string
	for _, columnMeta := range pkMetaMap {
		tmpColumnMeta := columnMeta
		if tmpColumnMeta.Autoincrement {
			autoColumnName = tmpColumnMeta.ColumnName
			break
		}
	}
	if len(autoColumnName) == 0 {
		return nil, fmt.Errorf("auto increment column not exist")
	}

	updateCount, err := i.BusinesSQLResult.GetResult().RowsAffected()
	if err != nil {
		return nil, err
	}

	lastInsertId, err := i.BusinesSQLResult.GetResult().LastInsertId()
	if err != nil {
		return nil, err
	}

	// If there is batch insert
	// do auto increment base LAST_INSERT_ID and variable `auto_increment_increment`
	if lastInsertId > 0 && updateCount > 1 && internal.CanAutoIncrement(pkMetaMap) {
		return i.autoGeneratePks(execCtx, autoColumnName, lastInsertId, updateCount)
	}

	if lastInsertId > 0 {
		var pkValues []interface{}
		pkValues = append(pkValues, lastInsertId)
		pkValuesMap[autoColumnName] = pkValues
		return pkValuesMap, nil
	}

	return nil, nil
}

func (i *InsertExecutor) autoGeneratePks(execCtx *types.ExecContext, autoColumnName string, lastInsetId, updateCount int64) (map[string][]interface{}, error) {
	var step int64
	if i.IncrementStep > 0 {
		step = int64(i.IncrementStep)
	} else {
		// get step by query sql
		stmt, err := execCtx.Conn.Prepare("SHOW VARIABLES LIKE 'auto_increment_increment'")
		if err != nil {
			log.Errorf("build prepare stmt: %+v", err)
			return nil, err
		}

		rows, err := stmt.Query(nil)
		if err != nil {
			log.Errorf("stmt query: %+v", err)
			return nil, err
		}

		if len(rows.Columns()) > 0 {
			var curStep []driver.Value
			if err := rows.Next(curStep); err != nil {
				return nil, err
			}

			if curStepInt, ok := curStep[0].(int64); ok {
				step = curStepInt
			}
		} else {
			return nil, fmt.Errorf("query is empty")
		}
	}

	if step == 0 {
		return nil, fmt.Errorf("get increment step error")
	}

	var pkValues []interface{}
	for j := int64(0); j < updateCount; j++ {
		pkValues = append(pkValues, lastInsetId+j*step)
	}
	pkValuesMap := make(map[string][]interface{})
	pkValuesMap[autoColumnName] = pkValues
	return pkValuesMap, nil
}
