/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package at

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/arana-db/parser/ast"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

const (
	sqlPlaceholder = "?"
)

var errPostgreSQLInsertLastInsertIDUnsupported = errors.New("LastInsertId is not supported for PostgreSQL AT insert")

type insertResult struct {
	lastInsertID  int64
	rowsAffected  int64
	lastInsertErr error
}

func (r *insertResult) LastInsertId() (int64, error) {
	if r.lastInsertErr != nil {
		return 0, r.lastInsertErr
	}
	return r.lastInsertID, nil
}

func (r *insertResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// insertExecutor execute insert SQL
type insertExecutor struct {
	baseExecutor
	parserCtx      *types.ParseContext
	execContext    *types.ExecContext
	incrementStep  int
	keyPlan        *insertKeyPlan
	resolvedPKRows []types.RowImage
	// businesSQLResult after insert sql
	businesSQLResult types.ExecResult
}

type insertKeyPlan struct {
	rowCount   int
	pkValues   map[string][]interface{}
	autoColumn string
}

// NewInsertExecutor get insert executor
func NewInsertExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &insertExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

func (i *insertExecutor) dbType() types.DBType {
	if i.execContext == nil {
		return types.DBTypeMySQL
	}
	return effectiveDBType(i.execContext.DBType)
}

func (i *insertExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if err := i.beforeHooks(ctx, i.execContext); err != nil {
		return nil, err
	}
	defer func() {
		i.afterHooks(ctx, i.execContext)
	}()

	beforeImage, err := i.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	if i.dbType() == types.DBTypePostgreSQL {
		res, afterImage, err := i.execPostgreSQLInsert(ctx)
		if err != nil {
			return nil, err
		}

		i.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
		i.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
		return res, nil
	}

	i.keyPlan, err = i.buildInsertKeyPlan(ctx, beforeImage.TableMeta)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, i.execContext.Query, i.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	if i.businesSQLResult == nil {
		i.businesSQLResult = res
	}
	noOp, err := i.validateInsertResult()
	if err != nil {
		return nil, err
	}
	if noOp {
		afterImage := types.NewEmptyRecordImage(beforeImage.TableMeta, types.SQLTypeInsert)
		if err := i.prepareUndoPair(i.execContext, beforeImage, afterImage); err != nil {
			return nil, err
		}
		return res, nil
	}

	afterImage, err := i.afterImage(ctx)
	if err != nil {
		return nil, err
	}
	if err := i.validateAfterImage(afterImage); err != nil {
		return nil, err
	}

	if err := i.prepareUndoPair(i.execContext, beforeImage, afterImage); err != nil {
		return nil, err
	}
	return res, nil
}

// beforeImage build before image
func (i *insertExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	tableCache, err := i.getTableCache(i.dbType())
	if err != nil {
		return nil, err
	}

	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := tableCache.GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	return types.NewEmptyRecordImage(metaData, types.SQLTypeInsert), nil
}

// afterImage build after image
func (i *insertExecutor) afterImage(ctx context.Context) (*types.RecordImage, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	dbType := i.dbType()
	tableCache, err := i.getTableCache(dbType)
	if err != nil {
		return nil, err
	}

	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := tableCache.GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs, err := i.buildAfterImageSQL(ctx)
	if err != nil {
		return nil, err
	}

	rowsi, err := i.queryRows(ctx, selectSQL, selectArgs)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rowsi != nil {
			rowsi.Close()
		}
	}()

	image, err := i.buildRecordImages(rowsi, metaData, types.SQLTypeInsert, dbType)
	if err != nil {
		return nil, err
	}
	image.TableMeta = metaData

	return image, nil
}

func (i *insertExecutor) validateAfterImage(afterImage *types.RecordImage) error {
	tableName, _ := i.parserCtx.GetTableName()
	if afterImage == nil || afterImage.TableMeta == nil {
		return fmt.Errorf("insert after image for table %s is unavailable", tableName)
	}
	if len(afterImage.Rows) != len(i.resolvedPKRows) {
		return fmt.Errorf("insert after image for table %s has %d rows, expected %d", tableName, len(afterImage.Rows), len(i.resolvedPKRows))
	}

	actualPKRows := make([]types.RowImage, 0, len(afterImage.Rows))
	for _, row := range afterImage.Rows {
		pkColumns, err := util.GetOrderedPkList(afterImage, row, i.dbType())
		if err != nil {
			return fmt.Errorf("insert after image for table %s: %w", tableName, err)
		}
		actualPKRows = append(actualPKRows, types.RowImage{Columns: pkColumns})
	}

	matchedExpected := make([]bool, len(i.resolvedPKRows))
	for _, actualRow := range actualPKRows {
		match := -1
		for expectedIndex, expectedRow := range i.resolvedPKRows {
			if !primaryKeyRowsEqual(expectedRow, actualRow) {
				continue
			}
			if match >= 0 {
				return fmt.Errorf("insert after image for table %s has duplicate expected primary key %v", tableName, primaryKeyValues(actualRow))
			}
			match = expectedIndex
		}
		if match < 0 {
			return fmt.Errorf("insert after image for table %s contains unexpected primary key %v", tableName, primaryKeyValues(actualRow))
		}
		if matchedExpected[match] {
			return fmt.Errorf("insert after image for table %s contains duplicate primary key %v", tableName, primaryKeyValues(actualRow))
		}
		matchedExpected[match] = true
	}
	for expectedIndex, ok := range matchedExpected {
		if !ok {
			return fmt.Errorf("insert after image for table %s is missing expected primary key %v", tableName, primaryKeyValues(i.resolvedPKRows[expectedIndex]))
		}
	}
	return nil
}

func primaryKeyRowsEqual(expected, actual types.RowImage) bool {
	if len(expected.Columns) != len(actual.Columns) {
		return false
	}
	for index := range expected.Columns {
		if !datasource.DeepEqual(expected.Columns[index].GetActualValue(), actual.Columns[index].GetActualValue()) {
			return false
		}
	}
	return true
}

func primaryKeyValues(row types.RowImage) []interface{} {
	values := make([]interface{}, 0, len(row.Columns))
	for index := range row.Columns {
		values = append(values, row.Columns[index].GetActualValue())
	}
	return values
}

func (i *insertExecutor) execPostgreSQLInsert(ctx context.Context) (types.ExecResult, *types.RecordImage, error) {
	tableCache, err := i.getTableCache(types.DBTypePostgreSQL)
	if err != nil {
		return nil, nil, err
	}

	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := tableCache.GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, nil, err
	}

	returningSQL, err := i.buildPostgreSQLReturningInsertSQL(metaData)
	if err != nil {
		return nil, nil, err
	}

	// PostgreSQL insert uses RETURNING to capture inserted rows, including generated PKs.
	rowsi, err := i.queryRows(ctx, returningSQL, i.execContext.NamedValues)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if rowsi != nil {
			rowsi.Close()
		}
	}()

	afterImage, err := i.buildRecordImages(rowsi, metaData, types.SQLTypeInsert, types.DBTypePostgreSQL)
	if err != nil {
		return nil, nil, err
	}

	lockKey := i.buildLockKey(afterImage, *metaData)
	if lockKey != "" {
		i.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	}

	result := types.NewResult(types.WithResult(&insertResult{
		rowsAffected:  int64(len(afterImage.Rows)),
		lastInsertErr: errPostgreSQLInsertLastInsertIDUnsupported,
	}))
	i.businesSQLResult = result
	return result, afterImage, nil
}

func (i *insertExecutor) buildPostgreSQLReturningInsertSQL(meta *types.TableMeta) (string, error) {
	if meta == nil {
		return "", fmt.Errorf("table meta is nil")
	}
	if !i.isAstStmtValid() {
		return "", fmt.Errorf("invalid insert stmt")
	}

	insertColumns := make([]string, 0, len(i.parserCtx.InsertStmt.Columns))
	for _, column := range i.parserCtx.InsertStmt.Columns {
		insertColumns = append(insertColumns, column.Name.O)
	}

	returningColumns, err := buildImageSelectColumns(meta, insertColumns, types.DBTypePostgreSQL, undo.UndoConfig.OnlyCareUpdateColumns)
	if err != nil {
		return "", err
	}

	return trimTrailingSemicolon(i.execContext.Query) + " RETURNING " + strings.Join(returningColumns, ", "), nil
}

func (i *insertExecutor) queryRows(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	// Try direct query first
	queryerCtx, ok := i.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = i.execContext.Conn.(driver.Queryer)
	}

	if ok {
		rowsi, err := util.CtxDriverQuery(ctx, queryerCtx, queryer, query, args)
		if err == nil {
			return rowsi, nil
		}
		// If not skip-fast-path error, return the error
		if !strings.Contains(err.Error(), "skip fast-path") {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
		// skip-fast-path error, fallback to prepared statement
		log.Debugf("direct query not supported, falling back to prepared statement")
	}

	// Fallback: use PrepareContext + QueryContext
	stmt, err := i.execContext.Conn.Prepare(query)
	if err != nil {
		log.Errorf("prepare statement failed: %+v", err)
		return nil, err
	}

	var rows driver.Rows
	if stmtQueryCtx, ok := stmt.(driver.StmtQueryContext); ok {
		rows, err = stmtQueryCtx.QueryContext(ctx, args)
	} else {
		var dargs []driver.Value
		dargs, err = namedValuesToValues(args)
		if err != nil {
			stmt.Close()
			return nil, err
		}
		rows, err = stmt.Query(dargs)
	}

	if err != nil {
		stmt.Close()
		return nil, err
	}

	return util.NewRowsWithStmt(rows, stmt), nil
}

func namedValuesToValues(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))
	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}
		dargs[n] = param.Value
	}
	return dargs, nil
}

func trimTrailingSemicolon(query string) string {
	trimmed := strings.TrimSpace(query)
	for strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	}
	return trimmed
}

// buildAfterImageSQL build select sql from insert sql
func (i *insertExecutor) buildAfterImageSQL(ctx context.Context) (string, []driver.NamedValue, error) {
	// get all pk value
	dbType := i.dbType()
	tableCache, err := i.getTableCache(dbType)
	if err != nil {
		return "", nil, err
	}

	tableName, _ := i.parserCtx.GetTableName()
	meta, err := tableCache.GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return "", nil, err
	}
	pkValuesMap, err := i.getPkValues(ctx, i.execContext, *meta)
	if err != nil {
		return "", nil, err
	}

	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return "", nil, fmt.Errorf("Pk columnName size is zero")
	}

	dataTypeMap, err := meta.GetPrimaryKeyTypeStrMap()
	if err != nil {
		return "", nil, err
	}
	if len(dataTypeMap) != len(pkColumnNameList) {
		return "", nil, fmt.Errorf("PK columnName size don't equal PK DataType size")
	}
	pkRowImages := make([]types.RowImage, 0)

	rowSize := len(pkValuesMap[pkColumnNameList[0]])
	if rowSize == 0 {
		return "", nil, fmt.Errorf("insert primary key values are empty")
	}
	for rowIndex := 0; rowIndex < rowSize; rowIndex++ {
		columns := make([]types.ColumnImage, 0, len(pkColumnNameList))
		for _, name := range pkColumnNameList {
			values := pkValuesMap[name]
			if len(values) != rowSize {
				return "", nil, fmt.Errorf("insert primary key %s has %d values, want %d", name, len(values), rowSize)
			}
			columns = append(columns, types.ColumnImage{
				KeyType:    types.IndexTypePrimaryKey,
				ColumnName: name,
				ColumnType: jdbcTypeForDatabaseType(dbType, dataTypeMap[name]),
				Value:      values[rowIndex],
			})
		}
		pkRowImages = append(pkRowImages, types.RowImage{Columns: columns})
	}
	// build check sql
	sb := strings.Builder{}
	suffix := strings.Builder{}
	var insertColumns []string

	for _, column := range i.parserCtx.InsertStmt.Columns {
		insertColumns = append(insertColumns, column.Name.O)
	}
	selectColumns, err := buildImageSelectColumns(meta, insertColumns, dbType, undo.UndoConfig.OnlyCareUpdateColumns)
	if err != nil {
		return "", nil, err
	}
	sb.WriteString("SELECT " + strings.Join(selectColumns, ", "))
	suffix.WriteString(" FROM " + tableName)
	whereSQL := i.buildWhereConditionByPKs(pkColumnNameList, rowSize, dbType, maxInSize)
	suffix.WriteString(" WHERE " + whereSQL + " ")
	sb.WriteString(suffix.String())
	i.resolvedPKRows = pkRowImages
	return sb.String(), i.buildPKParams(pkRowImages, pkColumnNameList, dbType), nil
}

func (i *insertExecutor) getPkValues(ctx context.Context, execCtx *types.ExecContext, meta types.TableMeta) (map[string][]interface{}, error) {
	if i.keyPlan == nil {
		plan, err := i.buildInsertKeyPlan(ctx, &meta)
		if err != nil {
			return nil, err
		}
		i.keyPlan = plan
	}
	return i.resolveInsertKeyPlan(execCtx)
}

func (i *insertExecutor) buildInsertKeyPlan(ctx context.Context, meta *types.TableMeta) (*insertKeyPlan, error) {
	if meta == nil || !i.isAstStmtValid() {
		return nil, fmt.Errorf("invalid insert metadata or statement")
	}
	stmt := i.parserCtx.InsertStmt
	if stmt.Select != nil || len(stmt.Lists) == 0 {
		return nil, fmt.Errorf("insert source other than VALUES is unsupported")
	}
	if stmt.IgnoreErr && len(stmt.Lists) > 1 {
		return nil, fmt.Errorf("multi-values INSERT IGNORE is unsupported because successful rows cannot be determined")
	}

	pkNames := meta.GetPrimaryKeyOnlyName()
	if len(pkNames) == 0 {
		return nil, fmt.Errorf("pk columnName size is zero")
	}
	pkValues, err := i.parsePkValuesFromStatement(stmt, *meta, i.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	plan := &insertKeyPlan{rowCount: len(stmt.Lists), pkValues: make(map[string][]interface{})}
	pkMeta := meta.GetPrimaryKeyMap()
	zeroGeneratesAutoIncrement := false
	zeroModeChecked := false
	for _, pkName := range pkNames {
		columnMeta, ok := pkMeta[pkName]
		if !ok {
			return nil, fmt.Errorf("primary key metadata not found for %s", pkName)
		}
		values, present := pkValues[pkName]
		generatedCount := 0
		if present {
			if len(values) != plan.rowCount {
				return nil, fmt.Errorf("insert primary key %s has %d values, want %d", pkName, len(values), plan.rowCount)
			}
			for valueIndex, value := range values {
				if boolValue, ok := value.(bool); ok && columnMeta.Autoincrement {
					if boolValue {
						value = int64(1)
					} else {
						value = int64(0)
					}
					values[valueIndex] = value
				}
				switch value.(type) {
				case nil, *ast.DefaultExpr, ast.DefaultExpr:
					generatedCount++
				case ast.ExprNode:
					return nil, fmt.Errorf("insert primary key expression for %s cannot be determined", pkName)
				default:
					if columnMeta.Autoincrement && isNumericZero(value) {
						if !zeroModeChecked {
							zeroGeneratesAutoIncrement, err = i.zeroGeneratesAutoIncrement(ctx)
							if err != nil {
								return nil, err
							}
							zeroModeChecked = true
						}
						if zeroGeneratesAutoIncrement {
							generatedCount++
						}
					}
				}
			}
		} else {
			generatedCount = plan.rowCount
		}

		if generatedCount == 0 {
			plan.pkValues[pkName] = values
			continue
		}
		if !columnMeta.Autoincrement {
			return nil, fmt.Errorf("primary key %s is not explicitly specified or auto-increment", pkName)
		}
		if generatedCount != plan.rowCount {
			return nil, fmt.Errorf("mixed explicit and generated values for auto-increment primary key %s are unsupported", pkName)
		}
		if plan.autoColumn != "" && plan.autoColumn != pkName {
			return nil, fmt.Errorf("multiple generated primary keys are unsupported")
		}
		plan.autoColumn = pkName
	}
	return plan, nil
}

func (i *insertExecutor) zeroGeneratesAutoIncrement(ctx context.Context) (bool, error) {
	rows, err := i.queryRows(ctx, "SELECT FIND_IN_SET('NO_AUTO_VALUE_ON_ZERO', @@SESSION.sql_mode)", nil)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	values := make([]driver.Value, 1)
	if err := rows.Next(values); err != nil {
		return false, err
	}
	return isNumericZero(values[0]), nil
}

func isNumericZero(value interface{}) bool {
	text := fmt.Sprint(value)
	if bytes, ok := value.([]byte); ok {
		text = string(bytes)
	}
	numeric, err := strconv.ParseFloat(text, 64)
	return err == nil && numeric == 0
}

func (i *insertExecutor) resolveInsertKeyPlan(execCtx *types.ExecContext) (map[string][]interface{}, error) {
	values := make(map[string][]interface{}, len(i.keyPlan.pkValues)+1)
	for name, pkValues := range i.keyPlan.pkValues {
		values[name] = append([]interface{}(nil), pkValues...)
	}

	if i.keyPlan.autoColumn == "" {
		return values, nil
	}
	if i.businesSQLResult == nil || i.businesSQLResult.GetResult() == nil {
		return nil, fmt.Errorf("insert result is unavailable for generated primary key")
	}

	lastInsertID, err := i.businesSQLResult.GetResult().LastInsertId()
	if err != nil {
		return nil, err
	}
	if lastInsertID <= 0 {
		return nil, fmt.Errorf("last insert id is unavailable for generated primary key %s", i.keyPlan.autoColumn)
	}

	if i.keyPlan.rowCount == 1 {
		values[i.keyPlan.autoColumn] = []interface{}{lastInsertID}
		return values, nil
	}

	generated, err := i.autoGeneratePks(execCtx, i.keyPlan.autoColumn, lastInsertID, int64(i.keyPlan.rowCount))
	if err != nil {
		return nil, err
	}
	pkValuesMapMerge(&values, generated)
	return values, nil
}

func (i *insertExecutor) validateInsertResult() (bool, error) {
	if i.keyPlan == nil || i.businesSQLResult == nil || i.businesSQLResult.GetResult() == nil {
		return false, fmt.Errorf("insert key plan or result is unavailable")
	}
	if !i.parserCtx.InsertStmt.IgnoreErr {
		return false, nil
	}
	rowsAffected, err := i.businesSQLResult.GetResult().RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		return true, nil
	}
	if rowsAffected != 1 {
		return false, fmt.Errorf("single-row INSERT IGNORE affected %d rows", rowsAffected)
	}
	return false, nil
}

// containsPK the columns contains table meta pk
func (i *insertExecutor) containsPK(meta types.TableMeta, parseCtx *types.ParseContext) bool {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}
	if parseCtx == nil || parseCtx.InsertStmt == nil || parseCtx.InsertStmt.Columns == nil {
		return false
	}
	if len(parseCtx.InsertStmt.Columns) == 0 {
		return false
	}

	matchCounter := 0
	dbType := i.dbType()
	for _, column := range parseCtx.InsertStmt.Columns {
		cleanName := util.DelEscape(column.Name.O, dbType)
		for _, pkName := range pkColumnNameList {
			if strings.EqualFold(pkName, cleanName) {
				matchCounter++
			}
		}
	}

	return matchCounter == len(pkColumnNameList)
}

// containPK compare column name and primary key name
func (i *insertExecutor) containPK(columnName string, meta types.TableMeta) bool {
	newColumnName := util.DelEscape(columnName, i.dbType())
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}
	for _, name := range pkColumnNameList {
		if strings.EqualFold(name, newColumnName) {
			return true
		}
	}
	return false
}

// getPkIndex get pk index
// return the key is pk column name and the value is index of the pk column
func (i *insertExecutor) getPkIndex(InsertStmt *ast.InsertStmt, meta types.TableMeta) map[string]int {
	pkIndexMap := make(map[string]int)
	if InsertStmt == nil {
		return pkIndexMap
	}
	insertColumnsSize := len(InsertStmt.Columns)
	if meta.ColumnNames == nil {
		return pkIndexMap
	}
	if insertColumnsSize == 0 {
		if len(InsertStmt.Lists) == 0 {
			return pkIndexMap
		}
		// INSERT without a column list follows the physical table column order.
		for idx, columnName := range meta.ColumnNames {
			if pkColumnName, ok := i.matchPKColumnName(columnName, meta); ok {
				pkIndexMap[pkColumnName] = idx
			}
		}
		return pkIndexMap
	}
	for paramIdx := 0; paramIdx < insertColumnsSize; paramIdx++ {
		sqlColumnName := InsertStmt.Columns[paramIdx].Name.O
		if pkColumnName, ok := i.matchPKColumnName(sqlColumnName, meta); ok {
			pkIndexMap[pkColumnName] = paramIdx
		}
	}
	return pkIndexMap
}

func (i *insertExecutor) matchPKColumnName(columnName string, meta types.TableMeta) (string, bool) {
	newColumnName := util.DelEscape(columnName, i.dbType())
	for _, name := range meta.GetPrimaryKeyOnlyName() {
		if strings.EqualFold(name, newColumnName) {
			return name, true
		}
	}
	return "", false
}

// parsePkValuesFromStatement parse primary key value from statement.
// return the primary key and values<key:primary key,value:primary key values></key:primary>
func (i *insertExecutor) parsePkValuesFromStatement(insertStmt *ast.InsertStmt, meta types.TableMeta, nameValues []driver.NamedValue) (map[string][]interface{}, error) {
	if insertStmt == nil {
		return nil, fmt.Errorf("insert statement is nil")
	}
	if len(insertStmt.Lists) == 0 {
		return nil, fmt.Errorf("insert VALUES list is empty")
	}

	expectedColumns := len(insertStmt.Columns)
	if expectedColumns == 0 {
		expectedColumns = len(meta.ColumnNames)
	}
	for rowIndex, list := range insertStmt.Lists {
		if len(insertStmt.Columns) == 0 && len(list) == 0 {
			continue
		}
		if len(list) != expectedColumns {
			return nil, fmt.Errorf("insert row %d has %d values, want %d", rowIndex, len(list), expectedColumns)
		}
		for _, node := range list {
			var indexes []int32
			i.traversalArgs(node, &indexes)
			for _, index := range indexes {
				if index < 0 || int(index) >= len(nameValues) {
					return nil, fmt.Errorf("insert parameter index %d out of range for %d arguments", index, len(nameValues))
				}
			}
		}
	}

	pkValuesMap := make(map[string][]interface{})
	for _, list := range insertStmt.Lists {
		if len(list) == 0 {
			for _, pkName := range meta.GetPrimaryKeyOnlyName() {
				pkValuesMap[pkName] = append(pkValuesMap[pkName], ast.DefaultExpr{})
			}
			continue
		}
		for pkName, pkIndex := range i.getPkIndex(insertStmt, meta) {
			if pkIndex < 0 || pkIndex >= len(list) {
				return nil, fmt.Errorf("primary key %s index %d out of range", pkName, pkIndex)
			}
			node := list[pkIndex]
			if _, ok := node.(ast.ParamMarkerExpr); ok {
				var indexes []int32
				i.traversalArgs(node, &indexes)
				if len(indexes) != 1 {
					return nil, fmt.Errorf("primary key %s parameter position cannot be determined", pkName)
				}
				pkValuesMap[pkName] = append(pkValuesMap[pkName], nameValues[indexes[0]].Value)
				continue
			}
			if valueExpr, ok := node.(ast.ValueExpr); ok {
				pkValuesMap[pkName] = append(pkValuesMap[pkName], valueExpr.GetValue())
				continue
			}
			pkValuesMap[pkName] = append(pkValuesMap[pkName], node)
		}
	}
	return pkValuesMap, nil
}

// getPkValuesByColumn get pk value by column.
func (i *insertExecutor) getPkValuesByColumn(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	tableCache, err := i.getTableCache(i.dbType())
	if err != nil {
		return nil, err
	}

	tableName, _ := i.parserCtx.GetTableName()
	meta, err := tableCache.GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	pkValuesMap, err := i.parsePkValuesFromStatement(i.parserCtx.InsertStmt, *meta, execCtx.NamedValues)
	if err != nil {
		return nil, err
	}

	// generate pkValue by auto increment
	for _, v := range pkValuesMap {
		tmpV := v
		if len(tmpV) == 1 {
			// pk auto generated while single insert primary key is expression
			if _, ok := tmpV[0].(*ast.FuncCallExpr); ok {
				curPkValueMap, err := i.getPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValueMap)
			}
		} else if len(tmpV) > 0 && tmpV[0] == nil {
			// pk auto generated while column exists and value is null
			curPkValueMap, err := i.getPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
			pkValuesMapMerge(&pkValuesMap, curPkValueMap)
		}
	}
	return pkValuesMap, nil
}

func (i *insertExecutor) getPkValuesByAuto(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	if i.dbType() == types.DBTypePostgreSQL {
		return nil, fmt.Errorf("postgresql auto generated pk retrieval should use RETURNING path")
	}

	tableCache, err := i.getTableCache(i.dbType())
	if err != nil {
		return nil, err
	}

	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := tableCache.GetTableMeta(ctx, i.execContext.DBName, tableName)
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

	updateCount, err := i.businesSQLResult.GetResult().RowsAffected()
	if err != nil {
		return nil, err
	}

	lastInsertId, err := i.businesSQLResult.GetResult().LastInsertId()
	if err != nil {
		return nil, err
	}

	// If there is batch insert
	// do auto increment base LAST_INSERT_ID and variable `auto_increment_increment`
	if lastInsertId > 0 && updateCount > 1 && canAutoGeneratePKs(pkMetaMap) {
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

func canAutoGeneratePKs(pkMetaMap map[string]types.ColumnMeta) bool {
	for _, meta := range pkMetaMap {
		if meta.Autoincrement {
			return true
		}
	}
	return false
}

func (i *insertExecutor) isAstStmtValid() bool {
	return i.parserCtx != nil && i.parserCtx.InsertStmt != nil
}

func (i *insertExecutor) autoGeneratePks(execCtx *types.ExecContext, autoColumnName string, lastInsetId, updateCount int64) (map[string][]interface{}, error) {
	var step int64
	if i.incrementStep > 0 {
		step = int64(i.incrementStep)
	} else {
		// get step by query sql
		stmt, err := execCtx.Conn.Prepare("SHOW VARIABLES LIKE 'auto_increment_increment'")
		if err != nil {
			log.Errorf("build prepare stmt: %+v", err)
			return nil, err
		}
		defer stmt.Close()

		rows, err := stmt.Query(nil)
		if err != nil {
			log.Errorf("stmt query: %+v", err)
			return nil, err
		}
		defer rows.Close()

		columns := rows.Columns()
		if len(columns) > 1 {
			curStep := make([]driver.Value, len(columns))
			if err := rows.Next(curStep); err != nil {
				return nil, err
			}

			curStepInt, err := parseAutoIncrementStep(curStep[1])
			if err != nil {
				return nil, err
			}
			step = curStepInt
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

func parseAutoIncrementStep(value driver.Value) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported auto_increment_increment value type %T", value)
	}
}

func pkValuesMapMerge(dest *map[string][]interface{}, src map[string][]interface{}) {
	for k, v := range src {
		tmpK := k
		tmpV := v
		(*dest)[tmpK] = tmpV
	}
}

// containsColumns judge sql specify column
func containsColumns(parseCtx *types.ParseContext) bool {
	if parseCtx == nil || parseCtx.InsertStmt == nil || parseCtx.InsertStmt.Columns == nil {
		return false
	}
	return len(parseCtx.InsertStmt.Columns) > 0
}

func getInsertRows(insertStmt *ast.InsertStmt, pkIndexArray []int) ([][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	if len(insertStmt.Lists) == 0 {
		return nil, nil
	}
	var rows [][]interface{}

	for _, nodes := range insertStmt.Lists {
		var row []interface{}
		for i, node := range nodes {
			if _, ok := node.(ast.ParamMarkerExpr); ok {
				row = append(row, sqlPlaceholder)
			} else if newNode, ok := node.(ast.ValueExpr); ok {
				row = append(row, newNode.GetValue())
			} else if newNode, ok := node.(*ast.VariableExpr); ok {
				row = append(row, newNode.Name)
			} else if _, ok := node.(*ast.FuncCallExpr); ok {
				row = append(row, ast.FuncCallExpr{})
			} else {
				for _, index := range pkIndexArray {
					if index == i {
						return nil, fmt.Errorf("Unknown SQLExpr:%v", node)
					}
				}
				row = append(row, ast.DefaultExpr{})
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
