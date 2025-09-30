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
	"fmt"
	"regexp"
	"strings"

	slog "log"

	"github.com/arana-db/parser/ast"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/log"
)

// insertOnUpdateExecutor execute insert on update SQL
type insertOnUpdateExecutor struct {
	baseExecutor
	parserCtx                 *types.ParseContext
	execContext               *types.ExecContext
	beforeImageSqlPrimaryKeys map[string]bool
	beforeSelectSql           string
	beforeSelectArgs          []driver.NamedValue
	isUpdateFlag              bool
	dbType                    types.DBType
}

// NewInsertOnUpdateExecutor get insert on update executor
func NewInsertOnUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &insertOnUpdateExecutor{
		baseExecutor:              baseExecutor{hooks: hooks},
		parserCtx:                 parserCtx,
		execContext:               execContent,
		beforeImageSqlPrimaryKeys: make(map[string]bool),
		dbType:                    execContent.TxCtx.DBType,
	}
}

func (i *insertOnUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	i.beforeHooks(ctx, i.execContext)
	defer func() {
		i.afterHooks(ctx, i.execContext)
	}()

	beforeImage, err := i.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, i.execContext.Query, i.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := i.afterImage(ctx, beforeImage)
	if err != nil {
		return nil, err
	}

	if len(beforeImage.Rows) > 0 {
		beforeImage.SQLType = types.SQLTypeUpdate
		afterImage.SQLType = types.SQLTypeUpdate
	} else {
		beforeImage.SQLType = types.SQLTypeInsert
		afterImage.SQLType = types.SQLTypeInsert
	}

	i.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	i.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

// beforeImage build before image
func (i *insertOnUpdateExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	if !i.isAstStmtValid() {
		log.Errorf("invalid insert statement! parser ctx:%+v", i.parserCtx)
		return nil, fmt.Errorf("invalid insert statement! parser ctx:%+v", i.parserCtx)
	}
	tableName, err := i.parserCtx.GetTableName()
	if err != nil {
		return nil, err
	}
	metaData, err := datasource.GetTableCache(i.dbType).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		log.Errorf("get table meta failed, dbType: %s, db: %s, table: %s, err: %+v", i.dbType, i.execContext.DBName, tableName, err)
		return nil, err
	}
	selectSQL, selectArgs, err := i.buildBeforeImageSQL(*metaData, i.execContext.NamedValues, i.parserCtx.InsertStmt)
	if err != nil {
		return nil, err
	}
	if len(selectArgs) == 0 {
		log.Errorf("the SQL statement has no primary key or unique index value, it will not hit any row data."+
			"recommend to convert to a normal insert statement. dbType: %s, db name:%s, table name:%s, sql:%s",
			i.dbType, i.execContext.DBName, tableName, i.execContext.Query)
		return nil, fmt.Errorf("invalid insert or update sql for dbType: %s", i.dbType)
	}
	i.beforeSelectSql = selectSQL
	i.beforeSelectArgs = selectArgs

	var rowsi driver.Rows
	queryerCtx, queryerCtxExists := i.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	var queryerExists bool

	if !queryerCtxExists {
		queryer, queryerExists = i.execContext.Conn.(driver.Queryer)
	}
	if !queryerExists && !queryerCtxExists {
		log.Errorf("target conn should implement driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn for dbType: %s", i.dbType)
	}
	rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
	defer func() {
		if rowsi != nil {
			rowsi.Close()
		}
	}()
	if err != nil {
		log.Errorf("ctx driver query failed, dbType: %s, sql: %s, err: %+v", i.dbType, selectSQL, err)
		return nil, err
	}
	image, err := i.buildRecordImages(ctx, i.execContext, rowsi, metaData, types.SQLTypeInsertOnDuplicateUpdate)
	if err != nil {
		return nil, err
	}
	return image, nil
}

// buildBeforeImageSQL build the SQL to query before image data
func (i *insertOnUpdateExecutor) buildBeforeImageSQL(metaData types.TableMeta, args []driver.NamedValue, insertStmt interface{}) (string, []driver.NamedValue, error) {
	if insertStmt == nil {
		err := fmt.Errorf("insertStmt is nil, cannot build before image SQL")
		log.Errorf(err.Error())
		return "", nil, err
	}

	if i.execContext == nil {
		err := fmt.Errorf("execContext is nil, cannot build before image SQL")
		log.Errorf(err.Error())
		return "", nil, err
	}

	var mysqlInsertStmt *ast.InsertStmt
	var pgInsertStmt *tree.Insert

	switch stmt := insertStmt.(type) {
	case *ast.InsertStmt:
		// MySQL AST
		mysqlInsertStmt = stmt
	case *tree.Insert:
		// PostgreSQL AST
		pgInsertStmt = stmt
	default:
		return "", nil, fmt.Errorf("unsupported insert statement type: %T", insertStmt)
	}

	if err := i.checkConflictSyntax(mysqlInsertStmt, pgInsertStmt, metaData); err != nil {
		return "", nil, err
	}

	paramMap, insertNum, err := i.buildBeforeImageSQLParameters(mysqlInsertStmt, pgInsertStmt, args, metaData)
	if err != nil {
		return "", nil, err
	}

	pkCols := metaData.GetPrimaryKeyOnlyName()
	for _, pkCol := range pkCols {
		lowerPkCol := strings.ToLower(pkCol)
		if _, ok := paramMap[lowerPkCol]; !ok || len(paramMap[lowerPkCol]) == 0 {
			paramMap[lowerPkCol] = make([]driver.NamedValue, insertNum)
			for rowIdx := 0; rowIdx < insertNum; rowIdx++ {
				paramMap[lowerPkCol][rowIdx] = driver.NamedValue{
					Ordinal: rowIdx + 1,
					Name:    pkCol,
					Value:   nil,
				}
			}
		}
	}

	sql := strings.Builder{}
	escapedTableName := i.escapeIdentifier(metaData.TableName, true)
	sql.WriteString("SELECT * FROM " + escapedTableName + "  ")
	isContainWhere := false
	var selectArgs []driver.NamedValue

	isConflictInsert := false
	if i.execContext.ParseContext != nil {
		isConflictInsert = i.execContext.ParseContext.SQLType == types.SQLTypeInsertOnDuplicateUpdate
	}

	var primaryIndex *types.IndexMeta
	var uniqueIndexes []types.IndexMeta

	isPostgreSQLDoNothing := (i.dbType == types.DBTypePostgreSQL &&
		strings.Contains(strings.ToUpper(i.execContext.Query), "DO NOTHING"))

	for _, idx := range metaData.Indexs {
		skip := false
		if isPostgreSQLDoNothing && idx.IType != types.IndexTypePrimaryKey {
			skip = true
		} else if !isConflictInsert && idx.IType != types.IndexTypePrimaryKey {
			skip = true
		} else if idx.NonUnique {
			skip = true
		} else if idx.IType != types.IndexTypePrimaryKey && isIndexValueNull(idx, paramMap, 0) {
			skip = true
		}
		if skip {
			continue
		}

		if idx.IType == types.IndexTypePrimaryKey && primaryIndex == nil {
			pkIdx := idx
			primaryIndex = &pkIdx
		} else if idx.IType != types.IndexTypePrimaryKey {
			uniqueIndexes = append(uniqueIndexes, idx)
		}
	}

	validPrimary := primaryIndex != nil && primaryIndex.IType == types.IndexTypePrimaryKey
	if !validPrimary {
		for _, idx := range metaData.Indexs {
			if idx.IType == types.IndexTypePrimaryKey && !idx.NonUnique {
				pkIdx := idx
				primaryIndex = &pkIdx
				validPrimary = true
				break
			}
		}
	}

	var orderedIndexes []types.IndexMeta
	if validPrimary {
		orderedIndexes = append(orderedIndexes, *primaryIndex)
	}
	orderedIndexes = append(orderedIndexes, uniqueIndexes...)

	for j := 0; j < insertNum; j++ {
		finalJ := j
		var rowParams []driver.NamedValue
		var rowConditions []string

		for _, index := range orderedIndexes {
			skip := false
			if index.IType != types.IndexTypePrimaryKey && isIndexValueNull(index, paramMap, finalJ) {
				skip = true
			}
			if skip {
				log.Infof("Skip index: %s (row index: %d, not a primary key and value is null)", index.Name, finalJ)
				continue
			}

			columnIsNull := true
			var colConditions []string
			for _, columnMeta := range index.Columns {
				colName := columnMeta.ColumnName
				escapedColName := i.escapeIdentifier(colName, false)
				lowerColName := strings.ToLower(colName)

				colParams, ok := paramMap[lowerColName]
				if !ok {
					log.Errorf("Fatal error: Column [%s] not found in paramMap (row index: %d), skipping directly", lowerColName, finalJ)
					continue
				}
				if finalJ >= len(colParams) {
					log.Errorf("Fatal error: Row index [%d] exceeds parameter length [%d] of column [%s] (insufficient parameter list length)", finalJ, len(colParams), lowerColName)
					continue
				}

				colParam := colParams[finalJ]
				log.Infof("Parameter extraction details: row index=%d, column name=%s (lowercase=%s), parameter ordinal=%d (parameter value=%v)",
					finalJ, colName, lowerColName, colParam.Ordinal, colParam.Value)

				colCond := escapedColName + " = ? "
				colConditions = append(colConditions, colCond)

				log.Infof("Row [%d] rowParams added: ordinal=%d (current rowParams length=%d)",
					finalJ, colParam.Ordinal, len(rowParams)+1)
				rowParams = append(rowParams, colParam)
				columnIsNull = false

				if index.IType == types.IndexTypePrimaryKey {
					i.beforeImageSqlPrimaryKeys[colName] = true
				}
			}

			if !columnIsNull {

				colCondStr := strings.Join(colConditions, " and ")

				indexCondition := "(" + colCondStr + ")"

				if !contains(rowConditions, indexCondition) {
					rowConditions = append(rowConditions, indexCondition)
					log.Infof("Row [%d] added condition: %s (current total row conditions=%d)", finalJ, indexCondition, len(rowConditions))
				}
			}
		}

		if len(rowConditions) > 0 {
			conditionStr := strings.Join(rowConditions, "  OR ")
			if isContainWhere {
				sql.WriteString("  OR " + conditionStr)
			} else {
				sql.WriteString("WHERE " + conditionStr)
				isContainWhere = true
			}
			selectArgs = append(selectArgs, rowParams...)
		}

	}

	if isContainWhere {
		currentSQL := strings.TrimSpace(sql.String())
		sql.Reset()
		sql.WriteString(currentSQL)
		sql.WriteString(" FOR UPDATE")
	}

	finalSQL := strings.TrimSpace(i.adaptSQLSyntax(sql.String()))

	if i.dbType == types.DBTypePostgreSQL {
		counter := 1
		finalSQL = regexp.MustCompile(`\?`).ReplaceAllStringFunc(finalSQL, func(match string) string {
			result := fmt.Sprintf("$%d", counter)
			counter++
			return result
		})

		for idx := range selectArgs {
			selectArgs[idx].Ordinal = idx + 1
		}
	}

	return finalSQL, selectArgs, nil
}

func getIndexNames(indexes []types.IndexMeta) []string {
	names := make([]string, len(indexes))
	for i, idx := range indexes {
		names[i] = idx.Name
	}
	return names
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// buildBeforeImageSQLParameters build the SQL parameters to query before image data
func (i *insertOnUpdateExecutor) buildBeforeImageSQLParameters(mysqlInsertStmt *ast.InsertStmt, pgInsertStmt *tree.Insert, args []driver.NamedValue, metaData types.TableMeta) (map[string][]driver.NamedValue, int, error) {
	var pkIndexArray []int
	var insertRows [][]interface{}
	var err error

	switch i.dbType {
	case types.DBTypeMySQL:
		if mysqlInsertStmt == nil {
			return nil, 0, fmt.Errorf("MySQL insertStmt is nil")
		}
		pkIndexArray = i.getPkIndexArray(mysqlInsertStmt, metaData)
		insertRows, err = getInsertRows(mysqlInsertStmt, pkIndexArray, i.dbType)
	case types.DBTypePostgreSQL:
		if pgInsertStmt == nil {
			return nil, 0, fmt.Errorf("PostgreSQL insertStmt is nil")
		}

		pkIndexArray = i.getPkIndexArrayFromPG(pgInsertStmt, metaData)
		conflictIndexArray := i.getConflictIndexArrayFromPG(pgInsertStmt, metaData)

		combinedIndexArray := i.mergeIndexArrays(pkIndexArray, conflictIndexArray)
		insertRows, err = getInsertRowsFromPG(pgInsertStmt, combinedIndexArray, i.dbType)
	default:
		return nil, 0, fmt.Errorf("unsupported database type: %s", i.dbType)
	}

	if err != nil {
		return nil, 0, err
	}

	parameterMap := make(map[string][]driver.NamedValue)
	var insertColumns []string

	switch i.dbType {
	case types.DBTypeMySQL:
		insertColumns = getInsertColumns(mysqlInsertStmt)
	case types.DBTypePostgreSQL:
		insertColumns = getInsertColumnsFromPG(pgInsertStmt)
	}

	placeHolderIndex := 0

	var globalOrdinal int = 1

	for _, rowColumns := range insertRows {
		if len(rowColumns) != len(insertColumns) {
			return nil, 0, fmt.Errorf("invalid insert row's column size")
		}
		for colIdx, col := range insertColumns {
			columnName := DelEscape(col, i.dbType)
			lowerColName := strings.ToLower(columnName)
			val := rowColumns[colIdx]

			rStr, ok := val.(string)
			if ok && strings.EqualFold(rStr, sqlPlaceholder) {
				if placeHolderIndex >= len(args) {
					return nil, 0, fmt.Errorf("insufficient parameters")
				}
				param := args[placeHolderIndex]
				if i.dbType == types.DBTypePostgreSQL {
					param.Ordinal = globalOrdinal
					globalOrdinal++
				}
				parameterMap[lowerColName] = append(parameterMap[lowerColName], param)
				placeHolderIndex++
			} else {
				namedVal := driver.NamedValue{
					Name:  columnName,
					Value: val,
				}
				switch i.dbType {
				case types.DBTypePostgreSQL:
					namedVal.Ordinal = globalOrdinal
					globalOrdinal++
				case types.DBTypeMySQL:
					namedVal.Ordinal = colIdx + 1
				}
				parameterMap[lowerColName] = append(parameterMap[lowerColName], namedVal)
			}
		}
	}

	pkCols := metaData.GetPrimaryKeyOnlyName()
	for _, pkCol := range pkCols {
		lowerPkCol := strings.ToLower(pkCol)
		if _, ok := parameterMap[lowerPkCol]; !ok || len(parameterMap[lowerPkCol]) == 0 {
			log.Warnf("primary key %s not found in paramMap, force add default", pkCol)

			for rowIdx := 0; rowIdx < len(insertRows); rowIdx++ {
				namedVal := driver.NamedValue{
					Name:  pkCol,
					Value: nil,
				}
				switch i.dbType {
				case types.DBTypePostgreSQL:
					namedVal.Ordinal = globalOrdinal
					globalOrdinal++
				case types.DBTypeMySQL:
					namedVal.Ordinal = rowIdx + 1
				}
				parameterMap[lowerPkCol] = append(parameterMap[lowerPkCol], namedVal)
			}
		}
	}

	log.Infof("final paramMap: %v", parameterMap)

	if idParams, ok := parameterMap["id"]; ok {

		var ordinalAndValues []string
		for _, param := range idParams {
			item := fmt.Sprintf("%d:%v", param.Ordinal, param.Value)
			ordinalAndValues = append(ordinalAndValues, item)
		}
	} else {
		slog.Printf("ID parameter not found in parameterMap")
	}

	return parameterMap, len(insertRows), nil
}

func (i *insertOnUpdateExecutor) escapeIdentifier(name string, isTableName bool) string {
	if name == "" {
		return name
	}
	switch i.dbType {
	case types.DBTypeMySQL:
		if isTableName {
			if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
				return name
			}
			return "`" + name + "`"
		} else {
			if strings.HasPrefix(name, "``") && strings.HasSuffix(name, "``") {
				return name
			}
			return "`" + name + "`"
		}
	case types.DBTypePostgreSQL:
		if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
			return name
		}
		return "\"" + name + "\""
	default:
		return name
	}
}

func (i *insertOnUpdateExecutor) adaptSQLSyntax(sql string) string {
	switch i.dbType {
	case types.DBTypePostgreSQL:
		sql = strings.ReplaceAll(sql, "SQL_NO_CACHE ", "")
		re := regexp.MustCompile(`_UTF8MB4(\w+)`)
		sql = re.ReplaceAllString(sql, "'$1'")
		re = regexp.MustCompile(`= (\w+)(\s|$|\)|\,)`)
		sql = re.ReplaceAllString(sql, "= '$1'$2")
	case types.DBTypeMySQL:
		re := regexp.MustCompile(`(_UTF8MB4)(\w+)`)
		sql = re.ReplaceAllString(sql, "${1}'${2}'")
	}
	return sql
}

func (i *insertOnUpdateExecutor) checkConflictSyntax(mysqlInsertStmt *ast.InsertStmt, pgInsertStmt *tree.Insert, metaData types.TableMeta) error {
	switch i.dbType {
	case types.DBTypeMySQL:
		if mysqlInsertStmt != nil {
			return i.checkMySQLOneDuplicatePK(mysqlInsertStmt, metaData)
		}
		return fmt.Errorf("MySQL insertStmt is nil")
	case types.DBTypePostgreSQL:
		if i.execContext.ParseContext != nil &&
			i.execContext.ParseContext.SQLType == types.SQLTypeInsertOnDuplicateUpdate {
			return i.checkPostgreSQLOnConflict(pgInsertStmt, metaData)
		}
		return nil
	default:
		return fmt.Errorf("unsupported dbType for conflict check: %s", i.dbType)
	}
}

func (i *insertOnUpdateExecutor) checkMySQLOneDuplicatePK(insertStmt *ast.InsertStmt, metaData types.TableMeta) error {
	duplicateColsMap := make(map[string]bool)
	for _, v := range insertStmt.OnDuplicate {
		duplicateColsMap[strings.ToLower(v.Column.Name.L)] = true
	}
	if len(duplicateColsMap) == 0 {
		return nil
	}
	for _, index := range metaData.Indexs {
		if types.IndexTypePrimaryKey != index.IType {
			continue
		}
		for _, col := range index.Columns {
			if duplicateColsMap[strings.ToLower(col.ColumnName)] {
				return fmt.Errorf("mysql does not support updating primary key: index=%s, column=%s", index.Name, col.ColumnName)
			}
		}
	}
	return nil
}

func (i *insertOnUpdateExecutor) checkPostgreSQLOnConflict(pgInsertStmt *tree.Insert, metaData types.TableMeta) error {
	if pgInsertStmt == nil {
		return fmt.Errorf("PostgreSQL insertStmt is nil")
	}

	if !strings.Contains(strings.ToUpper(i.execContext.Query), "ON CONFLICT") {
		return fmt.Errorf("postgresql insert on duplicate requires ON CONFLICT clause")
	}

	query := strings.ToUpper(i.execContext.Query)

	if !strings.Contains(query, "DO UPDATE") && !strings.Contains(query, "DO NOTHING") {
		return fmt.Errorf("postgresql ON CONFLICT requires DO UPDATE or DO NOTHING")
	}

	onConflictPattern := regexp.MustCompile(`ON\s+CONFLICT\s*\(\s*([^)]+)\s*\)`)
	matches := onConflictPattern.FindStringSubmatch(query)
	if len(matches) < 2 {
		constraintPattern := regexp.MustCompile(`ON\s+CONFLICT\s+ON\s+CONSTRAINT\s+(\w+)`)
		constraintMatches := constraintPattern.FindStringSubmatch(query)
		if len(constraintMatches) < 2 {
			return fmt.Errorf("postgresql ON CONFLICT requires conflict target (columns or constraint)")
		}
		return nil
	}

	conflictCols := strings.Split(strings.TrimSpace(matches[1]), ",")
	for _, col := range conflictCols {
		colName := strings.TrimSpace(col)
		colName = strings.Trim(colName, "\"")

		found := false
		for _, columnMeta := range metaData.Columns {
			if strings.EqualFold(columnMeta.ColumnName, colName) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("postgresql ON CONFLICT column '%s' not found in table %s", colName, metaData.TableName)
		}
	}

	if strings.Contains(query, "DO UPDATE") {
		return i.checkPostgreSQLUpdateColumns(query, metaData)
	}

	return nil
}

func (i *insertOnUpdateExecutor) checkPostgreSQLUpdateColumns(query string, metaData types.TableMeta) error {
	setPattern := regexp.MustCompile(`(?i)DO\s+UPDATE\s+SET\s+(.+?)(?:\s+WHERE|$)`)
	matches := setPattern.FindStringSubmatch(query)
	if len(matches) < 2 {
		return nil
	}

	updatePart := matches[1]
	assignments := strings.Split(updatePart, ",")

	pkColumns := metaData.GetPrimaryKeyOnlyName()
	pkMap := make(map[string]bool)
	for _, pk := range pkColumns {
		pkMap[strings.ToLower(pk)] = true
	}

	for _, assignment := range assignments {
		if equalPos := strings.Index(assignment, "="); equalPos > 0 {
			colName := strings.TrimSpace(assignment[:equalPos])
			colName = strings.Trim(colName, "\"")

			rightSide := strings.TrimSpace(assignment[equalPos+1:])
			if strings.HasPrefix(strings.ToUpper(rightSide), "EXCLUDED.") {
				excludedCol := strings.TrimSpace(rightSide[9:])
				excludedCol = strings.Trim(excludedCol, "\"")

				if pkMap[strings.ToLower(excludedCol)] {
					return fmt.Errorf("postgresql does not support updating primary key column: %s (via EXCLUDED.%s)", colName, excludedCol)
				}
			}

			if pkMap[strings.ToLower(colName)] {
				return fmt.Errorf("postgresql does not support updating primary key column: %s", colName)
			}
		}
	}

	return nil
}

// afterImage build after image
func (i *insertOnUpdateExecutor) afterImage(ctx context.Context, beforeImages *types.RecordImage) (*types.RecordImage, error) {
	afterSelectSql, selectArgs := i.buildAfterImageSQL(beforeImages)
	var rowsi driver.Rows
	queryerCtx, queryerCtxExists := i.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	var queryerExists bool

	if !queryerCtxExists {
		queryer, queryerExists = i.execContext.Conn.(driver.Queryer)
	}
	if !queryerCtxExists && !queryerExists {
		log.Errorf("target conn should implement driver.QueryerContext or driver.Queryer (dbType: %s)", i.dbType)
		return nil, fmt.Errorf("invalid conn for dbType: %s", i.dbType)
	}

	rowsi, err := util.CtxDriverQuery(ctx, queryerCtx, queryer, afterSelectSql, selectArgs)
	defer func() {
		if rowsi != nil {
			rowsi.Close()
		}
	}()
	if err != nil {
		log.Errorf("ctx driver query failed (dbType: %s), sql: %s, err: %+v", i.dbType, afterSelectSql, err)
		return nil, err
	}

	tableName, err := i.parserCtx.GetTableName()
	if err != nil {
		return nil, err
	}
	metaData, err := datasource.GetTableCache(i.dbType).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		log.Errorf("get table meta for after-image failed (dbType: %s), table: %s, err: %+v", i.dbType, tableName, err)
		return nil, err
	}

	afterImage, err := i.buildRecordImages(ctx, i.execContext, rowsi, metaData, types.SQLTypeInsertOnDuplicateUpdate)
	if err != nil {
		return nil, err
	}

	lockKey := i.buildLockKey(afterImage, *metaData)
	i.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	return afterImage, nil
}

// buildAfterImageSQL build the SQL to query after image data
func (i *insertOnUpdateExecutor) buildAfterImageSQL(beforeImage *types.RecordImage) (string, []driver.NamedValue) {
	finalSQL := i.adaptSQLSyntax(i.beforeSelectSql)
	log.Infof("build after select sql by insert on duplicate sourceQuery (dbType: %s), sql: %s", i.dbType, finalSQL)
	return finalSQL, i.beforeSelectArgs
}

// isPKColumn check the column name to see if it is a primary key column
func (i *insertOnUpdateExecutor) isPKColumn(columnName string, meta types.TableMeta) bool {
	newColumnName := DelEscape(columnName, i.dbType)
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}

	escapedPKList := make([]string, len(pkColumnNameList))
	for idx, pkName := range pkColumnNameList {
		escapedPKList[idx] = DelEscape(pkName, i.dbType)
	}

	for _, escapedPK := range escapedPKList {
		if strings.EqualFold(newColumnName, escapedPK) {
			return true
		}
	}
	return false
}

// getPkIndexArray get index of primary key from insert statement
func (i *insertOnUpdateExecutor) getPkIndexArray(insertStmt *ast.InsertStmt, meta types.TableMeta) []int {
	var pkIndexArray []int
	if insertStmt == nil {
		return pkIndexArray
	}

	insertColumnsSize := len(insertStmt.Columns)
	if insertColumnsSize == 0 {
		return i.getPkIndexFromAllColumns(meta)
	}

	for paramIdx := 0; paramIdx < insertColumnsSize; paramIdx++ {
		sqlColumnName := insertStmt.Columns[paramIdx].Name.O
		if i.isPKColumn(sqlColumnName, meta) {
			pkIndexArray = append(pkIndexArray, paramIdx)
		}
	}

	if len(pkIndexArray) == 0 {
		pkIndexArray = i.getPkIndexFromAllColumns(meta)
	}

	return pkIndexArray
}

func (i *insertOnUpdateExecutor) getPkIndexFromAllColumns(meta types.TableMeta) []int {
	var pkIndexArray []int
	if len(meta.Columns) == 0 {
		return pkIndexArray
	}

	pkIndex := -1
	for _, columnMeta := range meta.Columns {
		pkIndex++
		if i.isPKColumn(columnMeta.ColumnName, meta) {
			pkIndexArray = append(pkIndexArray, pkIndex)
		}
	}
	return pkIndexArray
}

func (i *insertOnUpdateExecutor) isAstStmtValid() bool {
	return i.parserCtx != nil && i.parserCtx.InsertStmt != nil
}

func isIndexValueNull(indexMeta types.IndexMeta, imageParameterMap map[string][]driver.NamedValue, rowIndex int) bool {
	if indexMeta.IType == types.IndexTypePrimaryKey {
		for _, colMeta := range indexMeta.Columns {
			lowerColName := strings.ToLower(colMeta.ColumnName)
			if _, ok := imageParameterMap[lowerColName]; ok {
				return false
			}
		}
		return true
	}

	for _, colMeta := range indexMeta.Columns {
		lowerColName := strings.ToLower(colMeta.ColumnName)
		imageParameters := imageParameterMap[lowerColName]

		if imageParameters == nil && colMeta.ColumnDef == nil {
			return true
		}
		if imageParameters != nil {
			if rowIndex >= len(imageParameters) {
				return true
			}
			val := imageParameters[rowIndex].Value
			if val == nil || (val == "" && colMeta.ColumnType != "string") {
				return true
			}
		}
	}
	return false
}

// getInsertColumns get insert columns from insert statement
func getInsertColumns(insertStmt *ast.InsertStmt) []string {
	if insertStmt == nil || len(insertStmt.Columns) == 0 {
		return nil
	}
	var list []string
	for _, col := range insertStmt.Columns {

		list = append(list, col.Name.O)
	}
	return list
}

func checkDuplicateKeyUpdate(insert *ast.InsertStmt, metaData types.TableMeta) error {
	if insert == nil || metaData.DBType != types.DBTypeMySQL {
		return nil
	}
	duplicateColsMap := make(map[string]bool)
	for _, v := range insert.OnDuplicate {
		duplicateColsMap[strings.ToLower(v.Column.Name.L)] = true
	}
	if len(duplicateColsMap) == 0 {
		return nil
	}
	for _, index := range metaData.Indexs {
		if types.IndexTypePrimaryKey != index.IType {
			continue
		}
		for _, col := range index.Columns {
			if duplicateColsMap[strings.ToLower(col.ColumnName)] {
				log.Errorf("update pk value is not supported! index name:%s update column name: %s (dbType: MySQL)", index.Name, col.ColumnName)
				return fmt.Errorf("update pk value is not supported! index name:%s update column name: %s", index.Name, col.ColumnName)
			}
		}
	}
	return nil
}

// PostgreSQL AST support functions

// getPkIndexArrayFromPG get index of primary key from PostgreSQL insert statement
func (i *insertOnUpdateExecutor) getPkIndexArrayFromPG(insertStmt *tree.Insert, meta types.TableMeta) []int {
	var pkIndexArray []int
	if insertStmt == nil || insertStmt.Columns == nil {
		return i.getPkIndexFromAllColumns(meta)
	}

	insertColumnsSize := len(insertStmt.Columns)
	if insertColumnsSize == 0 {
		return i.getPkIndexFromAllColumns(meta)
	}

	for paramIdx := 0; paramIdx < insertColumnsSize; paramIdx++ {
		colName := string(insertStmt.Columns[paramIdx])
		if i.isPKColumn(colName, meta) {
			pkIndexArray = append(pkIndexArray, paramIdx)
		}
	}

	if len(pkIndexArray) == 0 {
		pkIndexArray = i.getPkIndexFromAllColumns(meta)
	}

	return pkIndexArray
}

// getInsertRowsFromPG get insert rows from PostgreSQL AST
func getInsertRowsFromPG(insertStmt *tree.Insert, pkIndexArray []int, dbType types.DBType) ([][]interface{}, error) {
	if insertStmt == nil {
		return nil, fmt.Errorf("insertStmt is nil")
	}

	var rows [][]interface{}

	// PostgreSQL AST structure: Rows can be *tree.Select which contains VALUES clause
	if insertStmt.Rows == nil {
		return nil, fmt.Errorf("insert rows is nil")
	}

	// Access the Values from the Select statement (Rows is already *tree.Select)
	select_stmt := insertStmt.Rows

	// Extract VALUES clause from SELECT
	if select_stmt.Select == nil {
		return nil, fmt.Errorf("select statement is nil")
	}

	values_clause, ok := select_stmt.Select.(*tree.ValuesClause)
	if !ok {
		return nil, fmt.Errorf("unsupported PostgreSQL select type: %T", select_stmt.Select)
	}

	// Extract rows from VALUES clause
	for _, row := range values_clause.Rows {
		var rowValues []interface{}
		for _, expr := range row {
			// Extract value from expression
			value := extractValueFromExpr(expr)
			rowValues = append(rowValues, value)
		}
		rows = append(rows, rowValues)
	}

	return rows, nil
}

// getInsertColumnsFromPG get insert columns from PostgreSQL insert statement
func getInsertColumnsFromPG(insertStmt *tree.Insert) []string {
	if insertStmt == nil || insertStmt.Columns == nil {
		return nil
	}
	var list []string
	for _, col := range insertStmt.Columns {
		list = append(list, string(col))
	}
	return list
}

// extractValueFromExpr extract value from PostgreSQL expression
func extractValueFromExpr(expr tree.Expr) interface{} {
	switch e := expr.(type) {
	case *tree.DInt:
		return int64(*e)
	case *tree.DString:
		return string(*e)
	case *tree.DFloat:
		return float64(*e)
	case *tree.DBool:
		return bool(*e)
	case *tree.Placeholder:
		// For placeholders like $1, $2, return the placeholder string
		return sqlPlaceholder
	default:
		// For unknown types, convert to string
		return fmt.Sprintf("%v", e)
	}
}

func (i *insertOnUpdateExecutor) getConflictIndexArrayFromPG(insertStmt *tree.Insert, meta types.TableMeta) []int {
	var conflictIndexArray []int

	if i.execContext.ParseContext == nil ||
		i.execContext.ParseContext.SQLType != types.SQLTypeInsertOnDuplicateUpdate {
		return conflictIndexArray
	}

	if strings.Contains(strings.ToUpper(i.execContext.Query), "DO NOTHING") {
		return conflictIndexArray
	}

	conflictColumns := i.extractConflictColumnsFromSQL(i.execContext.Query)
	if len(conflictColumns) == 0 {
		return conflictIndexArray
	}

	if insertStmt == nil || insertStmt.Columns == nil || len(insertStmt.Columns) == 0 {
		return conflictIndexArray
	}

	for conflictCol := range conflictColumns {
		for paramIdx, col := range insertStmt.Columns {
			colName := DelEscape(string(col), i.dbType)
			if strings.EqualFold(colName, conflictCol) {
				conflictIndexArray = append(conflictIndexArray, paramIdx)
				break
			}
		}
	}

	return conflictIndexArray
}

func (i *insertOnUpdateExecutor) extractConflictColumnsFromSQL(query string) map[string]bool {
	conflictColumns := make(map[string]bool)

	onConflictPattern := regexp.MustCompile(`(?i)ON\s+CONFLICT\s*\(\s*([^)]+)\s*\)`)
	matches := onConflictPattern.FindStringSubmatch(query)

	if len(matches) >= 2 {
		colsPart := strings.TrimSpace(matches[1])
		cols := strings.Split(colsPart, ",")

		for _, col := range cols {
			colName := strings.TrimSpace(col)
			colName = strings.Trim(colName, "\"'`")
			if colName != "" {
				conflictColumns[colName] = true
			}
		}
	}

	return conflictColumns
}

func (i *insertOnUpdateExecutor) mergeIndexArrays(pkIndexArray, conflictIndexArray []int) []int {
	indexSet := make(map[int]bool)
	var result []int

	for _, idx := range pkIndexArray {
		if !indexSet[idx] {
			indexSet[idx] = true
			result = append(result, idx)
		}
	}

	for _, idx := range conflictIndexArray {
		if !indexSet[idx] {
			indexSet[idx] = true
			result = append(result, idx)
		}
	}

	return result
}
