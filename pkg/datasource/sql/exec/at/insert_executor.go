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
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/pkg/errors"
	"strconv"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	seatalog "seata.apache.org/seata-go/pkg/util/log"
)

const (
	sqlPlaceholder = "?"
)

// insertExecutor execute insert SQL
type insertExecutor struct {
	baseExecutor
	parserCtx     *types.ParseContext
	execContext   *types.ExecContext
	incrementStep int
	// businesSQLResult after insert sql
	businesSQLResult types.ExecResult
}

// NewInsertExecutor get insert executor
func NewInsertExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &insertExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

func (i *insertExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	i.beforeHooks(ctx, i.execContext)
	defer func() {
		i.afterHooks(ctx, i.execContext)
	}()

	beforeImage, err := i.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	adaptedQuery, adaptedArgs, err := i.adaptInsertSQLForPostgreSQL(ctx, i.execContext.Query, i.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, adaptedQuery, adaptedArgs)
	if err != nil {
		return nil, err
	}

	if i.businesSQLResult == nil {
		i.businesSQLResult = res
	}

	afterImage, err := i.afterImage(ctx)
	if err != nil {
		return nil, err
	}

	i.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	i.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

func (i *insertExecutor) adaptInsertSQLForPostgreSQL(ctx context.Context, query string, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	dbType := i.execContext.TxCtx.DBType
	if dbType != types.DBTypePostgreSQL {
		return query, args, nil
	}

	if i.parserCtx == nil {
		return "", nil, fmt.Errorf("parser context is nil")
	}
	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := i.getTableMeta(ctx, i.execContext, dbType, tableName)
	if err != nil {
		return "", nil, fmt.Errorf("get table meta for PostgreSQL adapt failed: %+v", err)
	}

	autoPkCol := ""
	pkMetaMap := metaData.GetPrimaryKeyMap()
	for colName, colMeta := range pkMetaMap {
		if colMeta.Autoincrement {
			autoPkCol = colName
			break
		}
	}
	if autoPkCol == "" {
		return query, args, nil
	}

	if strings.Contains(strings.ToUpper(query), "RETURNING") {
		return query, args, nil
	}

	escapedCol := i.escapeName(autoPkCol, dbType)
	adaptedQuery := fmt.Sprintf("%s RETURNING %s", strings.TrimSuffix(query, ";"), escapedCol)
	return adaptedQuery, args, nil
}

// beforeImage build before image
func (i *insertExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	if i.parserCtx == nil {
		return nil, fmt.Errorf("parser context is nil")
	}
	tableName, _ := i.parserCtx.GetTableName()
	dbType := i.execContext.TxCtx.DBType
	metaData, err := i.getTableMeta(ctx, i.execContext, dbType, tableName)
	if err != nil {
		seatalog.Errorf("get table meta for before-image failed: %+v, dbType: %s, table: %s, dbName: %s",
			err, dbType, tableName, i.execContext.DBName)
		return nil, err
	}
	return types.NewEmptyRecordImage(metaData, types.SQLTypeInsert), nil
}

// afterImage build after image
func (i *insertExecutor) afterImage(ctx context.Context) (*types.RecordImage, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	if i.parserCtx == nil {
		return nil, fmt.Errorf("parser context is nil")
	}
	tableName, _ := i.parserCtx.GetTableName()
	dbType := i.execContext.TxCtx.DBType
	metaData, err := i.getTableMeta(ctx, i.execContext, dbType, tableName)
	if err != nil {
		return nil, err
	}

	selectSQL, selectArgs, err := i.buildAfterImageSQL(ctx)
	if err != nil {
		return nil, err
	}

	var rowsi driver.Rows
	queryerCtx, ok := i.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = i.execContext.Conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			seatalog.Errorf("ctx driver query: %+v, sql: %s", err, selectSQL)
			return nil, err
		}
	} else {
		seatalog.Errorf("target conn should be driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid database connection (not support QueryerContext/Queryer)")
	}

	image, err := i.buildRecordImages(
		ctx,
		i.execContext,
		rowsi,
		metaData,
		types.SQLTypeInsert,
	)
	if err != nil {
		seatalog.Errorf("build insert after-image failed: %+v", err)
		return nil, err
	}

	lockKey := i.buildLockKey(image, *metaData)
	i.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	return image, nil
}

// buildAfterImageSQL build select sql from insert sql
func (i *insertExecutor) buildAfterImageSQL(ctx context.Context) (string, []driver.NamedValue, error) {
	if i.parserCtx == nil {
		return "", nil, fmt.Errorf("parser context is nil")
	}
	tableName, _ := i.parserCtx.GetTableName()
	dbType := i.execContext.TxCtx.DBType
	meta, err := i.getTableMeta(ctx, i.execContext, dbType, tableName)
	if err != nil {
		return "", nil, fmt.Errorf("get table meta failed: %+v", err)
	}

	pkValuesMap, err := i.getPkValues(ctx, i.execContext, i.parserCtx, *meta)
	if err != nil {
		return "", nil, fmt.Errorf("get pk values failed: %+v", err)
	}
	pkColumns := meta.GetPrimaryKeyOnlyName()
	if len(pkColumns) == 0 {
		return "", nil, fmt.Errorf("table %s has no primary key", tableName)
	}
	rowSize := len(pkValuesMap[pkColumns[0]])

	whereCondition := i.buildWhereConditionByPKs(pkColumns, rowSize, dbType, maxInSize)
	escapedTable := i.escapeName(tableName, dbType)

	var insertColumns []string
	if i.parserCtx.InsertStmt != nil && i.parserCtx.InsertStmt.Columns != nil {
		for _, col := range i.parserCtx.InsertStmt.Columns {
			insertColumns = append(insertColumns, col.Name.O)
		}
	}
	needColumns := i.getNeedColumns(meta, insertColumns, dbType)
	escapedNeedColumns := make([]string, len(needColumns))
	for idx, col := range needColumns {
		if dbType == types.DBTypeMySQL {
			escapedNeedColumns[idx] = col
		} else {
			escapedNeedColumns[idx] = i.escapeName(col, dbType)
		}
	}
	needColumns = escapedNeedColumns

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("SELECT %s FROM %s WHERE %s",
		strings.Join(needColumns, ", "),
		escapedTable,
		whereCondition,
	))

	var selectSQL string
	if dbType == types.DBTypePostgreSQL {
		sb.WriteString(" FOR UPDATE")
		selectSQL = i.replacePlaceholdersForPostgreSQL(sb.String())
	} else {
		selectSQL = sb.String()
	}

	pkRowImages, err := i.buildPKRowImages(pkValuesMap, pkColumns, meta)
	if err != nil {
		return "", nil, fmt.Errorf("build pk row images failed: %+v", err)
	}
	selectArgs := i.buildPKParams(pkRowImages, pkColumns)

	seatalog.Infof("built after-image select sql: %s", selectSQL)
	return selectSQL, selectArgs, nil
}

func (i *insertExecutor) buildPKRowImages(pkValuesMap map[string][]interface{}, pkColumns []string, meta *types.TableMeta) ([]types.RowImage, error) {
	var rowImages []types.RowImage
	rowSize := len(pkValuesMap[pkColumns[0]])

	for rowIdx := 0; rowIdx < rowSize; rowIdx++ {
		var columnImages []types.ColumnImage
		for _, pkCol := range pkColumns {
			pkColMeta, ok := meta.Columns[pkCol]
			if !ok {
				return nil, fmt.Errorf("pk column %s not found in table meta", pkCol)
			}

			columnImages = append(columnImages, types.ColumnImage{
				KeyType:    types.IndexTypePrimaryKey,
				ColumnName: pkCol,
				ColumnType: types.ParseDBTypeToJavaType(i.execContext.TxCtx.DBType, pkColMeta.DatabaseTypeString),
				Value:      pkValuesMap[pkCol][rowIdx],
			})
		}
		rowImages = append(rowImages, types.RowImage{Columns: columnImages})
	}
	return rowImages, nil
}

func (i *insertExecutor) replacePlaceholdersForPostgreSQL(sql string) string {
	if sql == "" {
		return ""
	}
	placeholderIdx := 1
	var result strings.Builder
	for _, c := range sql {
		if c == '?' {
			result.WriteString(fmt.Sprintf("$%d", placeholderIdx))
			placeholderIdx++
		} else {
			result.WriteString(string(c))
		}
	}
	return result.String()
}

func (i *insertExecutor) getPkValues(ctx context.Context, execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error) {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	pkValuesMap := make(map[string][]interface{})
	var err error
	// when there is only one pk in the table
	if len(pkColumnNameList) == 1 {
		containsPKResult := i.containsPK(meta, parseCtx)
		containsColumnsResult := containsColumns(parseCtx)

		if containsPKResult {
			// the insert sql contain pk value
			pkValuesMap, err = i.getPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else if containsColumnsResult {
			// the insert table pk auto generated
			pkValuesMap, err = i.getPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else {
			pkValuesMap, err = i.getPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// when there is multiple pk in the table
		// 1,all pk columns are filled value.
		// 2,the auto increment pk column value is null, and other pk value are not null.
		pkValuesMap, err = i.getPkValuesByColumn(ctx, execCtx)
		if err != nil {
			return nil, err
		}
		for _, columnName := range pkColumnNameList {
			if _, ok := pkValuesMap[columnName]; !ok {
				curPkValuesMap, err := i.getPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValuesMap)
			}
		}
	}
	return pkValuesMap, nil
}

// containsPK the columns contains table meta pk
func (i *insertExecutor) containsPK(meta types.TableMeta, parseCtx *types.ParseContext) bool {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}
	if parseCtx == nil {
		return false
	}

	var columns []string
	dbType := i.execContext.TxCtx.DBType

	if dbType == types.DBTypePostgreSQL && parseCtx.AuxtenInsertStmt != nil {
		if parseCtx.AuxtenInsertStmt.Columns == nil || len(parseCtx.AuxtenInsertStmt.Columns) == 0 {
			return false
		}
		for _, col := range parseCtx.AuxtenInsertStmt.Columns {
			columns = append(columns, string(col))
		}
	} else if parseCtx.InsertStmt != nil && parseCtx.InsertStmt.Columns != nil {
		if len(parseCtx.InsertStmt.Columns) == 0 {
			return false
		}
		for _, column := range parseCtx.InsertStmt.Columns {
			columns = append(columns, column.Name.O)
		}
	} else {
		return false
	}

	matchCounter := 0
	for _, columnName := range columns {
		for _, pkName := range pkColumnNameList {
			cleanColumnName := i.unEscapeName(columnName, dbType)
			if strings.EqualFold(pkName, cleanColumnName) {
				matchCounter++
			}
		}
	}

	return matchCounter == len(pkColumnNameList)
}

// containPK compare column name and primary key name
func (i *insertExecutor) containPK(columnName string, meta types.TableMeta) bool {
	dbType := i.execContext.TxCtx.DBType
	newColumnName := i.unEscapeName(columnName, dbType)
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
func (i *insertExecutor) getPkIndex(insertStmt *ast.InsertStmt, meta types.TableMeta, dbType types.DBType) map[string]int {
	pkIndexMap := make(map[string]int)
	if insertStmt == nil {
		return pkIndexMap
	}
	insertColumnsSize := len(insertStmt.Columns)
	if insertColumnsSize == 0 {
		return pkIndexMap
	}
	if meta.ColumnNames == nil {
		return pkIndexMap
	}

	if len(meta.Columns) > 0 {
		for paramIdx := 0; paramIdx < insertColumnsSize; paramIdx++ {
			sqlColumnName := insertStmt.Columns[paramIdx].Name.O
			if i.containPK(sqlColumnName, meta) {
				pkIndexMap[i.unEscapeName(sqlColumnName, dbType)] = paramIdx
			}
		}
		return pkIndexMap
	}

	pkIndex := -1
	allColumns := meta.Columns
	for _, columnMeta := range allColumns {
		tmpColumnMeta := columnMeta
		pkIndex++
		if i.containPK(tmpColumnMeta.ColumnName, meta) {
			pkIndexMap[i.unEscapeName(tmpColumnMeta.ColumnName, dbType)] = pkIndex
		}
	}

	return pkIndexMap
}

// parsePkValuesFromStatement parse primary key value from statement.
// return the primary key and values<key:primary key,value:primary key values></key:primary>
func (i *insertExecutor) parsePkValuesFromStatement(insertStmt *ast.InsertStmt, meta types.TableMeta, nameValues []driver.NamedValue, dbType types.DBType) (map[string][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	pkIndexMap := i.getPkIndex(insertStmt, meta, dbType)
	if pkIndexMap == nil || len(pkIndexMap) == 0 {
		return nil, fmt.Errorf("pkIndex is not found")
	}
	var pkIndexArray []int
	for _, val := range pkIndexMap {
		tmpVal := val
		pkIndexArray = append(pkIndexArray, tmpVal)
	}

	if insertStmt == nil || len(insertStmt.Lists) == 0 {
		return nil, fmt.Errorf("InsertStmt is empty")
	}

	pkValuesMap := make(map[string][]interface{})

	if nameValues != nil && len(nameValues) > 0 {
		insertRows, err := getInsertRows(insertStmt, pkIndexArray, dbType)
		if err != nil {
			return nil, err
		}
		if insertRows == nil || len(insertRows) == 0 {
			return nil, err
		}

		totalPlaceholderNum := 0
		for _, row := range insertRows {
			if len(row) == 0 {
				continue
			}
			currentRowPlaceholderNum := 0
			for _, r := range row {
				rStr, ok := r.(string)
				if ok && (rStr == sqlPlaceholder ||
					(dbType == types.DBTypePostgreSQL && strings.HasPrefix(rStr, "$"))) {
					currentRowPlaceholderNum++
				}
			}

			for key, index := range pkIndexMap {
				curKey := key
				curIndex := index

				if curIndex > len(row)-1 {
					continue
				}
				pkValue := row[curIndex]
				pkValueStr, ok := pkValue.(string)
				isPlaceholder := ok && (pkValueStr == sqlPlaceholder ||
					(dbType == types.DBTypePostgreSQL && strings.HasPrefix(pkValueStr, "$")))

				if isPlaceholder {
					paramIdx := totalPlaceholderNum
					if dbType == types.DBTypePostgreSQL {
						paramIdx = totalPlaceholderNum
					}
					if paramIdx >= len(nameValues) {
						return nil, fmt.Errorf("placeholder index out of range (paramIdx: %d, nameValues len: %d)",
							paramIdx, len(nameValues))
					}
					pkValuesMap[curKey] = append(pkValuesMap[curKey], nameValues[paramIdx].Value)
					totalPlaceholderNum++
				} else {
					pkValuesMap[curKey] = append(pkValuesMap[curKey], pkValue)
				}
			}

			totalPlaceholderNum += (currentRowPlaceholderNum - len(pkIndexMap))
		}
	} else {
		for _, list := range insertStmt.Lists {
			for pkName, pkIndex := range pkIndexMap {
				tmpPkName := pkName
				tmpPkIndex := pkIndex
				if tmpPkIndex >= len(list) {
					return nil, fmt.Errorf("pkIndex out of range")
				}
				if node, ok := list[tmpPkIndex].(ast.ValueExpr); ok {
					pkValuesMap[tmpPkName] = append(pkValuesMap[tmpPkName], node.GetValue())
				}
			}
		}
	}

	return pkValuesMap, nil
}

// getPkValuesByColumn get pk value by column.
func (i *insertExecutor) getPkValuesByColumn(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	if i.parserCtx == nil {
		return nil, fmt.Errorf("parser context is nil")
	}
	tableName, _ := i.parserCtx.GetTableName()
	dbType := execCtx.TxCtx.DBType
	metaData, err := i.getTableMeta(ctx, execCtx, dbType, tableName)
	if err != nil {
		seatalog.Errorf("get table meta for getPkValuesByColumn failed: %+v, dbType: %s, table: %s",
			err, dbType, tableName)
		return nil, err
	}
	var pkValuesMap map[string][]interface{}
	if dbType == types.DBTypePostgreSQL && i.parserCtx.AuxtenInsertStmt != nil {
		pkValuesMap, err = i.parsePkValuesFromPostgreSQLStatement(
			i.parserCtx.AuxtenInsertStmt,
			*metaData,
			execCtx.NamedValues,
			dbType,
		)
	} else {
		pkValuesMap, err = i.parsePkValuesFromStatement(
			i.parserCtx.InsertStmt,
			*metaData,
			execCtx.NamedValues,
			dbType,
		)
	}
	if err != nil {
		return nil, err
	}

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

// getPkValuesByAuto get pk value by auto increment.
func (i *insertExecutor) getPkValuesByAuto(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	if i.parserCtx == nil {
		return nil, fmt.Errorf("parser context is nil")
	}
	tableName, _ := i.parserCtx.GetTableName()
	dbType := execCtx.TxCtx.DBType
	metaData, err := i.getTableMeta(ctx, execCtx, dbType, tableName)
	if err != nil {
		seatalog.Errorf("get table meta for getPkValuesByAuto failed: %+v, dbType: %s, table: %s",
			err, dbType, tableName)
		return nil, err
	}

	pkValuesMap := make(map[string][]interface{})
	pkMetaMap := metaData.GetPrimaryKeyMap()
	if len(pkMetaMap) == 0 {
		return nil, fmt.Errorf("table %s has no primary key (dbType: %s)", tableName, dbType)
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
		return nil, fmt.Errorf("table %s has no auto-increment primary key (dbType: %s)", tableName, dbType)
	}

	updateCount, err := i.businesSQLResult.GetResult().RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get affected rows failed: %+v (table: %s)", err, tableName)
	}

	switch dbType {
	case types.DBTypeMySQL:
		lastInsertId, err := i.businesSQLResult.GetResult().LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("MySQL get last insert id failed: %+v", err)
		}
		if lastInsertId > 0 && updateCount > 1 && canAutoIncrement(pkMetaMap) {
			return i.autoGeneratePks(execCtx, autoColumnName, lastInsertId, updateCount)
		}
		if lastInsertId > 0 {
			pkValuesMap[autoColumnName] = []interface{}{lastInsertId}
			return pkValuesMap, nil
		}

	case types.DBTypePostgreSQL:
		rows := i.businesSQLResult.GetRows()
		if rows == nil {
			return nil, fmt.Errorf("PostgreSQL insert result Rows is nil (missing RETURNING clause?)")
		}
		defer rows.Close()

		var autoIDs []interface{}
		cols := rows.Columns()
		if len(cols) == 0 {
			return nil, fmt.Errorf("PostgreSQL RETURNING result has no columns")
		}

		for {
			values := make([]driver.Value, len(cols))
			if err := rows.Next(values); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					break
				}
				return nil, fmt.Errorf("PostgreSQL read auto id failed: %+v", err)
			}
			autoIDs = append(autoIDs, values[0])
		}

		if len(autoIDs) != int(updateCount) {
			return nil, fmt.Errorf("PostgreSQL auto id count (%d) != affected rows (%d)", len(autoIDs), updateCount)
		}
		pkValuesMap[autoColumnName] = autoIDs
		return pkValuesMap, nil
	}

	return nil, fmt.Errorf("unsupported dbType for auto-increment: %s", dbType)
}

func canAutoIncrement(pkMetaMap map[string]types.ColumnMeta) bool {
	if len(pkMetaMap) != 1 {
		return false
	}
	for _, meta := range pkMetaMap {
		return meta.Autoincrement
	}
	return false
}

func (i *insertExecutor) isAstStmtValid() bool {
	if i.parserCtx == nil {
		return false
	}
	return i.parserCtx.InsertStmt != nil || i.parserCtx.AuxtenInsertStmt != nil
}

func (i *insertExecutor) autoGeneratePks(execCtx *types.ExecContext, autoColumnName string, lastInsertId, updateCount int64) (map[string][]interface{}, error) {
	var step int64
	dbType := execCtx.TxCtx.DBType

	if i.incrementStep > 0 {
		step = int64(i.incrementStep)
	} else {
		switch dbType {
		case types.DBTypeMySQL:
			stmt, err := execCtx.Conn.Prepare("SHOW VARIABLES LIKE 'auto_increment_increment'")
			if err != nil {
				return nil, fmt.Errorf("MySQL prepare auto-increment step stmt failed: %+v", err)
			}
			defer stmt.Close()

			rows, err := stmt.Query(nil)
			if err != nil {
				return nil, fmt.Errorf("MySQL query auto-increment step failed: %+v", err)
			}
			defer rows.Close()

			var varName, varValue string
			values := []driver.Value{&varName, &varValue}
			if err := rows.Next(values); err != nil {
				return nil, fmt.Errorf("MySQL scan auto-increment step failed: %+v", err)
			}
			step, err = strconv.ParseInt(varValue, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse auto-increment step failed: %+v, value: %s", err, varValue)
			}

		case types.DBTypePostgreSQL:
			if i.parserCtx == nil {
				return nil, fmt.Errorf("parser context is nil")
			}
			tableName, _ := i.parserCtx.GetTableName()
			seqName := fmt.Sprintf("%s_%s_seq", strings.ToLower(tableName), strings.ToLower(autoColumnName))
			stmt, err := execCtx.Conn.Prepare(
				fmt.Sprintf("SELECT increment_by FROM pg_catalog.pg_sequences WHERE sequencename = '%s'", seqName),
			)
			if err != nil {
				return nil, fmt.Errorf("PostgreSQL prepare sequence step stmt failed: %+v", err)
			}
			defer stmt.Close()

			rows, err := stmt.Query(nil)
			if err != nil {
				return nil, fmt.Errorf("PostgreSQL query sequence step failed: %+v", err)
			}
			defer rows.Close()

			var incrementBy int64
			values := []driver.Value{&incrementBy}
			if err := rows.Next(values); err != nil {
				return nil, fmt.Errorf("PostgreSQL scan sequence step failed: %+v", err)
			}
			step = incrementBy
		}
	}

	if step <= 0 {
		step = 1
	}

	var pkValues []interface{}
	for j := int64(0); j < updateCount; j++ {
		pkValues = append(pkValues, lastInsertId+j*step)
	}
	pkValuesMap := make(map[string][]interface{})
	pkValuesMap[autoColumnName] = pkValues
	return pkValuesMap, nil
}

// parsePkValuesFromPostgreSQLStatement parse primary key value from PostgreSQL statement.
func (i *insertExecutor) parsePkValuesFromPostgreSQLStatement(insertStmt *tree.Insert, meta types.TableMeta, nameValues []driver.NamedValue, dbType types.DBType) (map[string][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}

	pkValuesMap := make(map[string][]interface{})

	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return nil, fmt.Errorf("table has no primary key")
	}

	pkIndexMap := make(map[string]int)
	if insertStmt.Columns != nil {
		for idx, col := range insertStmt.Columns {
			colName := i.unEscapeName(string(col), dbType)
			for _, pkName := range pkColumnNameList {
				if strings.EqualFold(pkName, colName) {
					pkIndexMap[pkName] = idx
					break
				}
			}
		}
	}

	if len(pkIndexMap) == 0 {
		return pkValuesMap, nil
	}

	insertRows, err := i.getInsertRowsForPostgreSQL(insertStmt, pkIndexMap, nameValues, dbType)
	if err != nil {
		return nil, err
	}

	for _, row := range insertRows {
		for pkName, pkIndex := range pkIndexMap {
			if pkIndex < len(row) {
				pkValuesMap[pkName] = append(pkValuesMap[pkName], row[pkIndex])
			}
		}
	}

	return pkValuesMap, nil
}

func pkValuesMapMerge(dest *map[string][]interface{}, src map[string][]interface{}) {
	for k, v := range src {
		tmpK := k
		tmpV := v
		(*dest)[tmpK] = append((*dest)[tmpK], tmpV)
	}
}

// containsColumns judge sql specify column
func containsColumns(parseCtx *types.ParseContext) bool {
	if parseCtx == nil {
		return false
	}

	if parseCtx.AuxtenInsertStmt != nil && parseCtx.AuxtenInsertStmt.Columns != nil {
		return len(parseCtx.AuxtenInsertStmt.Columns) > 0
	}

	if parseCtx.InsertStmt != nil && parseCtx.InsertStmt.Columns != nil {
		return len(parseCtx.InsertStmt.Columns) > 0
	}

	return false
}

func getInsertRows(insertStmt *ast.InsertStmt, pkIndexArray []int, dbType types.DBType) ([][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	if len(insertStmt.Lists) == 0 {
		return nil, nil
	}
	var rows [][]interface{}
	globalParamIndex := 1

	for _, nodes := range insertStmt.Lists {
		var row []interface{}
		for _, node := range nodes {
			switch node.(type) {
			case ast.ParamMarkerExpr:
				row = append(row, "?")
				globalParamIndex++
			case ast.ValueExpr:
				row = append(row, node.(ast.ValueExpr).GetValue())
			case *ast.VariableExpr:
				row = append(row, node.(*ast.VariableExpr).Name)
			case *ast.FuncCallExpr:
				row = append(row, ast.FuncCallExpr{})
			default:
				isPkCol := false
				for _, pkIdx := range pkIndexArray {
					if pkIdx == len(row) {
						isPkCol = true
						break
					}
				}
				if isPkCol {
					return nil, fmt.Errorf("unknown expression type for PK column: %T", node)
				}
				row = append(row, ast.DefaultExpr{})
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func getPlaceholderByDBType(index int, dbType types.DBType) string {
	switch dbType {
	case types.DBTypePostgreSQL:
		return fmt.Sprintf("$%d", index)
	default:
		return sqlPlaceholder
	}
}

func (i *insertExecutor) getPlaceholder(index int, dbType types.DBType) string {
	switch dbType {
	case types.DBTypePostgreSQL:
		return fmt.Sprintf("$%d", index)
	default:
		return sqlPlaceholder
	}
}

func (i *insertExecutor) escapeName(name string, dbType types.DBType) string {
	if name == "" {
		return ""
	}
	switch dbType {
	case types.DBTypeMySQL:
		if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
			return name
		}
		return fmt.Sprintf("`%s`", name)
	case types.DBTypePostgreSQL:
		if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
			return name
		}
		return fmt.Sprintf("\"%s\"", name)
	default:
		return name
	}
}

func (i *insertExecutor) unEscapeName(name string, dbType types.DBType) string {
	switch dbType {
	case types.DBTypeMySQL:
		return strings.Trim(name, "`")
	case types.DBTypePostgreSQL:
		return strings.Trim(name, "\"")
	default:
		return name
	}
}

// getInsertRowsForPostgreSQL extracts rows data from PostgreSQL INSERT statement
func (i *insertExecutor) getInsertRowsForPostgreSQL(insertStmt *tree.Insert, pkIndexMap map[string]int, nameValues []driver.NamedValue, dbType types.DBType) ([][]interface{}, error) {
	if insertStmt == nil {
		return nil, fmt.Errorf("PostgreSQL insert statement is nil")
	}

	var rows [][]interface{}

	if insertStmt.Rows != nil {
		if selectClause := insertStmt.Rows.Select; selectClause != nil {
			if valuesClause, ok := selectClause.(*tree.ValuesClause); ok {
				placeholderIndex := 0

				for _, rowTuple := range valuesClause.Rows {
					var row []interface{}

					for _, expr := range rowTuple {
						switch e := expr.(type) {
						case *tree.Placeholder:
							paramIndex := int(e.Idx) - 1
							if nameValues != nil && paramIndex >= 0 && paramIndex < len(nameValues) {
								row = append(row, nameValues[paramIndex].Value)
							} else {
								row = append(row, fmt.Sprintf("$%d", e.Idx))
							}
							if int(e.Idx) > placeholderIndex {
								placeholderIndex = int(e.Idx)
							}
						case *tree.NumVal:
							if val, err := strconv.ParseInt(e.String(), 10, 64); err == nil {
								row = append(row, val)
							} else if val, err := strconv.ParseFloat(e.String(), 64); err == nil {
								row = append(row, val)
							} else {
								row = append(row, e.String())
							}
						case *tree.StrVal:
							row = append(row, e.RawString())
						default:
							row = append(row, fmt.Sprintf("%v", e))
						}
					}
					rows = append(rows, row)
				}
			} else {
				// INSERT INTO ... SELECT ...
				return nil, fmt.Errorf("INSERT SELECT statements are not supported for primary key extraction")
			}
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows found in PostgreSQL INSERT statement")
	}

	return rows, nil
}
