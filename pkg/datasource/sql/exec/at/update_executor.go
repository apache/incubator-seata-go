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
	"regexp"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	seatabytes "seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	maxInSize = 1000
)

// updateExecutor execute update SQL
type updateExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
	dbType      types.DBType
}

// NewUpdateExecutor get update executor
func NewUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	// Because update join cannot be clearly identified when SQL cannot be parsed
	if parserCtx != nil && parserCtx.UpdateStmt != nil && parserCtx.UpdateStmt.TableRefs != nil && parserCtx.UpdateStmt.TableRefs.TableRefs.Right != nil {
		return NewUpdateJoinExecutor(parserCtx, execContent, hooks)
	}
	var dbType types.DBType = types.DBTypeMySQL
	if execContent.TxCtx != nil {
		dbType = execContent.TxCtx.DBType
	}
	return &updateExecutor{
		parserCtx:    parserCtx,
		execContext:  execContent,
		baseExecutor: baseExecutor{hooks: hooks},
		dbType:       dbType,
	}
}

// ExecContext exec SQL, and generate before image and after image
func (u *updateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	u.beforeHooks(ctx, u.execContext)
	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()

	beforeImage, err := u.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, u.execContext.Query, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := u.afterImage(ctx, *beforeImage)
	if err != nil {
		return nil, err
	}

	if len(beforeImage.Rows) != len(afterImage.Rows) {
		return nil, fmt.Errorf("Before image size is not equaled to after image size, probably because you updated the primary keys.")
	}

	u.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	u.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)

	return res, nil
}

// beforeImage build before image
func (u *updateExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	selectSQL, selectArgs, err := u.buildBeforeImageSQL(ctx, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	tableName, _ := u.parserCtx.GetTableName()

	// Get table meta cache using CreateTableMetaCache to ensure proper initialization
	dsManager := datasource.GetDataSourceManager(u.execContext.TxCtx.TransactionMode.BranchType())
	if dsManager == nil {
		return nil, fmt.Errorf("data source manager not found for transaction mode: %v", u.execContext.TxCtx.TransactionMode)
	}

	db, ok := u.execContext.DB.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("invalid DB type, expected *sql.DB")
	}

	tableMetaCache, err := dsManager.CreateTableMetaCache(ctx, u.execContext.TxCtx.ResourceID, u.dbType, db)
	if err != nil {
		return nil, fmt.Errorf("get table meta cache failed: %w", err)
	}

	metaData, err := tableMetaCache.GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}

	var rowsi driver.Rows
	queryerCtx, ok := u.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = u.execContext.Conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	image, err := u.buildRecordImages(ctx, u.execContext, rowsi, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey(image, *metaData)
	u.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	image.SQLType = u.parserCtx.SQLType

	return image, nil
}

// afterImage build after image
func (u *updateExecutor) afterImage(ctx context.Context, beforeImage types.RecordImage) (*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}
	if len(beforeImage.Rows) == 0 {
		return &types.RecordImage{}, nil
	}

	tableName, _ := u.parserCtx.GetTableName()

	// Get table meta cache using CreateTableMetaCache to ensure proper initialization
	dsManager := datasource.GetDataSourceManager(u.execContext.TxCtx.TransactionMode.BranchType())
	if dsManager == nil {
		return nil, fmt.Errorf("data source manager not found for transaction mode: %v", u.execContext.TxCtx.TransactionMode)
	}

	db, ok := u.execContext.DB.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("invalid DB type, expected *sql.DB")
	}

	tableMetaCache, err := dsManager.CreateTableMetaCache(ctx, u.execContext.TxCtx.ResourceID, u.dbType, db)
	if err != nil {
		return nil, fmt.Errorf("get table meta cache failed: %w", err)
	}

	metaData, err := tableMetaCache.GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs := u.buildAfterImageSQL(beforeImage, metaData)

	var rowsi driver.Rows
	queryerCtx, ok := u.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = u.execContext.Conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	afterImage, err := u.buildRecordImages(ctx, u.execContext, rowsi, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}
	afterImage.SQLType = u.parserCtx.SQLType

	return afterImage, nil
}

func (u *updateExecutor) isAstStmtValid() bool {
	return u.parserCtx != nil && (u.parserCtx.UpdateStmt != nil || u.parserCtx.AuxtenUpdateStmt != nil)
}

// buildAfterImageSQL build the SQL to query after image data
func (u *updateExecutor) buildAfterImageSQL(beforeImage types.RecordImage, meta *types.TableMeta) (string, []driver.NamedValue) {
	if len(beforeImage.Rows) == 0 {
		return "", nil
	}
	sb := strings.Builder{}
	// todo: OnlyCareUpdateColumns should load from config first
	var selectFields string
	var separator = ","
	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, row := range beforeImage.Rows {
			for _, column := range row.Columns {
				selectFields += column.ColumnName + separator
			}
		}
		selectFields = strings.TrimSuffix(selectFields, separator)
	} else {
		selectFields = "*"
	}
	sb.WriteString("SELECT " + selectFields + " FROM " + u.escapeIdentifier(meta.TableName, true) + " WHERE ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), u.dbType, maxInSize)
	sb.WriteString(" " + whereSQL + " ")
	sql := sb.String()
	sql = u.adaptSQLSyntax(sql)
	return sql, u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName())
}

func (u *updateExecutor) buildBeforeImageSQL(ctx context.Context, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	if !u.isAstStmtValid() {
		log.Errorf("invalid update stmt")
		return "", nil, fmt.Errorf("invalid update stmt")
	}

	if u.parserCtx.UpdateStmt != nil {
		return u.buildBeforeImageSQLForMySQL(ctx, args)
	}

	if u.parserCtx.AuxtenUpdateStmt != nil {
		return u.buildBeforeImageSQLForPostgreSQL(ctx, args)
	}

	return "", nil, fmt.Errorf("unsupported update statement type")
}

func (u *updateExecutor) buildBeforeImageSQLForMySQL(ctx context.Context, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	updateStmt := u.parserCtx.UpdateStmt
	fields := make([]*ast.SelectField, 0)

	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, column := range updateStmt.List {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: column.Column,
				},
			})
		}

		tableName, _ := u.parserCtx.GetTableName()

		// Get DB from execContext
		db, ok := u.execContext.DB.(*sql.DB)
		if !ok || db == nil {
			return "", nil, fmt.Errorf("database connection not available in execution context")
		}

		// Get TableMetaCache from DataSourceManager
		dsManager := datasource.GetDataSourceManager(u.execContext.TxCtx.TransactionMode.BranchType())
		if dsManager == nil {
			return "", nil, fmt.Errorf("data source manager not found for transaction mode: %v", u.execContext.TxCtx.TransactionMode)
		}

		tableMetaCache, err := dsManager.CreateTableMetaCache(ctx, u.execContext.TxCtx.ResourceID, u.dbType, db)
		if err != nil {
			return "", nil, fmt.Errorf("get table meta cache failed: %w", err)
		}

		metaData, err := tableMetaCache.GetTableMeta(ctx, u.execContext.DBName, tableName)
		if err != nil {
			return "", nil, err
		}
		for _, columnName := range metaData.GetPrimaryKeyOnlyName() {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: &ast.ColumnName{
						Name: model.CIStr{
							O: columnName,
							L: columnName,
						},
					},
				},
			})
		}
	} else {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.CIStr{
						O: "*",
						L: "*",
					},
				},
			},
		})
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{
			SQLCache: true,
		},
		From:       updateStmt.TableRefs,
		Where:      updateStmt.Where,
		Fields:     &ast.FieldList{Fields: fields},
		OrderBy:    updateStmt.Order,
		Limit:      updateStmt.Limit,
		TableHints: updateStmt.TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := seatabytes.NewByteBuffer([]byte{})
	_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())

	sql = u.adaptSQLSyntax(sql)

	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	return sql, u.buildSelectArgs(&selStmt, args), nil
}

func (u *updateExecutor) buildBeforeImageSQLForPostgreSQL(ctx context.Context, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	auxtenStmt := u.parserCtx.AuxtenUpdateStmt
	fields := make([]*ast.SelectField, 0)

	tableName, err := u.parserCtx.GetTableName()
	if err != nil {
		return "", nil, err
	}

	// Get DB from execContext
	db, ok := u.execContext.DB.(*sql.DB)
	if !ok || db == nil {
		return "", nil, fmt.Errorf("database connection not available in execution context")
	}

	// Get TableMetaCache from DataSourceManager
	dsManager := datasource.GetDataSourceManager(u.execContext.TxCtx.TransactionMode.BranchType())
	if dsManager == nil {
		return "", nil, fmt.Errorf("data source manager not found for transaction mode: %v", u.execContext.TxCtx.TransactionMode)
	}

	tableMetaCache, err := dsManager.CreateTableMetaCache(ctx, u.execContext.TxCtx.ResourceID, u.dbType, db)
	if err != nil {
		return "", nil, fmt.Errorf("get table meta cache failed: %w", err)
	}

	metaData, err := tableMetaCache.GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return "", nil, err
	}

	if undo.UndoConfig.OnlyCareUpdateColumns {
		fieldsMap := make(map[string]bool)

		for _, expr := range auxtenStmt.Exprs {
			columnName := strings.Trim(string(expr.Names[0].String()), `"`)
			if !fieldsMap[columnName] {
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{
						Name: &ast.ColumnName{
							Name: model.CIStr{O: columnName, L: columnName},
						},
					},
				})
				fieldsMap[columnName] = true
			}
		}

		originalSQL := tree.AsString(auxtenStmt)
		whereColumns := u.extractColumnsFromWhereClause(originalSQL)
		for _, columnName := range whereColumns {
			if !fieldsMap[columnName] {
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{
						Name: &ast.ColumnName{
							Name: model.CIStr{O: columnName, L: columnName},
						},
					},
				})
				fieldsMap[columnName] = true
			}
		}

		for _, columnName := range metaData.GetPrimaryKeyOnlyName() {
			if !fieldsMap[columnName] {
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{
						Name: &ast.ColumnName{
							Name: model.CIStr{
								O: columnName,
								L: columnName,
							},
						},
					},
				})
				fieldsMap[columnName] = true
			}
		}
	} else {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.CIStr{
						O: "*",
						L: "*",
					},
				},
			},
		})
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{
			SQLCache: true,
		},
		Fields: &ast.FieldList{Fields: fields},
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	tableName = strings.Trim(tableName, `"`)
	selStmt.From = &ast.TableRefsClause{
		TableRefs: &ast.Join{
			Left: &ast.TableSource{
				Source: &ast.TableName{
					Name: model.CIStr{O: tableName, L: tableName},
				},
			},
		},
	}

	b := seatabytes.NewByteBuffer([]byte{})
	_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())

	originalSQL := tree.AsString(auxtenStmt)
	whereMatch := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+ORDER\s+BY|\s+LIMIT|$)`).FindStringSubmatch(originalSQL)
	whereArgs := make([]driver.NamedValue, 0)

	if len(whereMatch) > 1 {
		whereClause := whereMatch[1]
		whereArgs = u.extractPostgreSQLWhereArgs(auxtenStmt, args)
		whereClause = u.remapPostgreSQLParameters(whereClause, len(auxtenStmt.Exprs))
		escapedWhereClause := strings.ReplaceAll(whereClause, "$", "$$")
		sql = regexp.MustCompile(`(?i)\s+FOR\s+UPDATE`).ReplaceAllString(sql, " WHERE "+escapedWhereClause+" FOR UPDATE")
	}

	sql = u.adaptSQLSyntax(sql)

	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	return sql, whereArgs, nil
}

func (u *updateExecutor) extractPostgreSQLWhereArgs(stmt *tree.Update, allArgs []driver.NamedValue) []driver.NamedValue {
	whereArgs := make([]driver.NamedValue, 0)

	updateExprsCount := u.countParametersInUpdateExprs(stmt.Exprs)

	if len(allArgs) > updateExprsCount {
		whereArgs = allArgs[updateExprsCount:]
	}

	return whereArgs
}

func (u *updateExecutor) countParametersInUpdateExprs(exprs tree.UpdateExprs) int {
	count := 0
	for _, expr := range exprs {
		if expr.Expr != nil {
			count += u.countParametersInExpr(expr.Expr)
		}
	}
	return count
}

func (u *updateExecutor) countParametersInExpr(expr tree.Expr) int {
	count := 0
	switch e := expr.(type) {
	case *tree.Placeholder:
		count = 1
	case *tree.FuncExpr:
		for _, arg := range e.Exprs {
			count += u.countParametersInExpr(arg)
		}
	case *tree.BinaryExpr:
		count += u.countParametersInExpr(e.Left)
		count += u.countParametersInExpr(e.Right)
	case *tree.UnaryExpr:
		count += u.countParametersInExpr(e.Expr)
	case *tree.ParenExpr:
		count += u.countParametersInExpr(e.Expr)
	}
	return count
}

func (u *updateExecutor) remapPostgreSQLParameters(whereClause string, updateExprsCount int) string {
	parameterMap := make(map[string]string)
	newParamIndex := 1

	re := regexp.MustCompile(`\$(\d+)`)
	result := re.ReplaceAllStringFunc(whereClause, func(match string) string {
		originalParam := match

		if replacement, exists := parameterMap[originalParam]; exists {
			return replacement
		} else {
			replacement := fmt.Sprintf("$%d", newParamIndex)
			parameterMap[originalParam] = replacement
			newParamIndex++
			return replacement
		}
	})

	return result
}

func (u *updateExecutor) extractColumnsFromWhereClause(sql string) []string {
	columns := make([]string, 0)

	whereMatch := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+ORDER\s+BY|\s+LIMIT|$)`).FindStringSubmatch(sql)
	if len(whereMatch) <= 1 {
		return columns
	}

	whereClause := whereMatch[1]

	columnPattern := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\s*[=<>!]`)
	matches := columnPattern.FindAllStringSubmatch(whereClause, -1)

	seenColumns := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			columnName := strings.Trim(match[1], `"`)
			if !seenColumns[columnName] {
				columns = append(columns, columnName)
				seenColumns[columnName] = true
			}
		}
	}

	return columns
}

// escapeIdentifier escape table name and column name for different database types
func (u *updateExecutor) escapeIdentifier(name string, isTableName bool) string {
	if name == "" {
		return name
	}
	switch u.dbType {
	case types.DBTypeMySQL:
		if isTableName {
			if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
				return name
			}
			return "`" + name + "`"
		} else {
			if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
				return name
			}
			return "`" + name + "`"
		}
	case types.DBTypePostgreSQL:
		if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
			return name
		}
		// PostgreSQL stores unquoted identifiers in lowercase
		// Convert table names to lowercase to match actual table names
		if isTableName {
			name = strings.ToLower(name)
		}
		return "\"" + name + "\""
	default:
		return name
	}
}

func (u *updateExecutor) adaptSQLSyntax(sql string) string {
	switch u.dbType {
	case types.DBTypePostgreSQL:
		return u.adaptPostgreSQLSyntax(sql)
	case types.DBTypeMySQL:
		return u.adaptMySQLSyntax(sql)
	}
	return sql
}

func (u *updateExecutor) adaptMySQLSyntax(sql string) string {
	sql = regexp.MustCompile(`_UTF8MB4(\w+)`).ReplaceAllString(sql, "'$1'")
	sql = regexp.MustCompile(`_utf8mb4(\w+)`).ReplaceAllString(sql, "'$1'")

	sql = u.convertTableIdentifiersToMySQL(sql)

	return sql
}

func (u *updateExecutor) adaptPostgreSQLSyntax(sql string) string {
	sql = strings.ReplaceAll(sql, "SQL_NO_CACHE ", "")
	sql = strings.ReplaceAll(sql, "SQL_CACHE ", "")

	sql = regexp.MustCompile(`_UTF8MB4(\w+)`).ReplaceAllString(sql, "'$1'")
	sql = regexp.MustCompile(`_utf8mb4(\w+)`).ReplaceAllString(sql, "'$1'")

	sql = regexp.MustCompile(`= (\w+)(\s*(?:AND|OR|\)|$|,))`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`= (\w+)(\s*(?:AND|OR|\)|$|,))`).FindStringSubmatch(match)
		if len(parts) == 3 {
			word := parts[1]
			suffix := parts[2]
			if u.isPostgreSQLIdentifier(word) {
				return match
			}
			return fmt.Sprintf("= '%s'%s", word, suffix)
		}
		return match
	})

	if !strings.Contains(sql, "$") {
		counter := 1
		sql = regexp.MustCompile(`\?`).ReplaceAllStringFunc(sql, func(match string) string {
			result := fmt.Sprintf("$%d", counter)
			counter++
			return result
		})
	}

	sql = u.convertMySQLIdentifiersToPostgreSQL(sql)
	sql = u.convertMySQLFunctionsToPostgreSQL(sql)

	sql = regexp.MustCompile(`\s*=\s*`).ReplaceAllString(sql, "=")
	sql = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(sql, ",")

	sql = regexp.MustCompile(`\(\(([^)]+)\)\s+AND\s+\(([^)]+)\)\)`).ReplaceAllString(sql, "$1 AND $2")
	sql = regexp.MustCompile(`\(([^)]+)\)\s+AND\s+\(([^)]+)\)`).ReplaceAllString(sql, "$1 AND $2")
	sql = regexp.MustCompile(`\(([^)]+\s+BETWEEN\s+[^)]+)\)`).ReplaceAllString(sql, "$1")
	sql = regexp.MustCompile(`\(([^)]+\s+IN\s+[^)]+)\)`).ReplaceAllString(sql, "$1")

	return sql
}

func (u *updateExecutor) isPostgreSQLIdentifier(word string) bool {
	pgKeywords := map[string]bool{
		"true": true, "false": true, "null": true, "TRUE": true, "FALSE": true, "NULL": true,
		"current_timestamp": true, "current_date": true, "current_time": true,
		"CURRENT_TIMESTAMP": true, "CURRENT_DATE": true, "CURRENT_TIME": true,
	}

	if pgKeywords[word] {
		return true
	}

	if regexp.MustCompile(`^\d+(\.\d+)?$`).MatchString(word) {
		return true
	}

	if strings.Contains(word, ".") {
		return true
	}

	return false
}

func (u *updateExecutor) convertMySQLIdentifiersToPostgreSQL(sql string) string {
	sql = regexp.MustCompile("`([^`]+)`").ReplaceAllString(sql, "\"$1\"")

	shouldEscapeIdentifiers := false

	if strings.Contains(sql, " user_table ") || strings.Contains(sql, "FROM user_table") {
		shouldEscapeIdentifiers = true
	}

	if !shouldEscapeIdentifiers {
		sql = regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`).ReplaceAllStringFunc(sql, func(match string) string {
			parts := regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(match)
			if len(parts) == 2 {
				tableName := parts[1]
				if !strings.Contains(tableName, "\"") {
					return "FROM \"" + tableName + "\""
				}
			}
			return match
		})
		return sql
	}

	sql = regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(match)
		if len(parts) == 2 {
			tableName := parts[1]
			if !strings.Contains(tableName, "\"") {
				return "FROM \"" + tableName + "\""
			}
		}
		return match
	})

	sql = regexp.MustCompile(`SELECT\s+([^F]+?)\s+FROM`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`SELECT\s+([^F]+?)\s+FROM`).FindStringSubmatch(match)
		if len(parts) == 2 {
			columnsPart := parts[1]
			if columnsPart != "*" {
				columns := strings.Split(columnsPart, ",")
				var escapedColumns []string
				for _, col := range columns {
					col = strings.TrimSpace(col)
					if col != "" && !strings.Contains(col, "\"") && col != "*" {
						escapedColumns = append(escapedColumns, "\""+col+"\"")
					} else {
						escapedColumns = append(escapedColumns, col)
					}
				}
				return "SELECT " + strings.Join(escapedColumns, ",") + " FROM"
			}
		}
		return match
	})

	sql = regexp.MustCompile(`WHERE\s+([^O]+?)(?:\s+ORDER|\s+FOR|$)`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`WHERE\s+([^O]+?)(?:\s+ORDER|\s+FOR|$)`).FindStringSubmatch(match)
		if len(parts) >= 2 {
			whereClause := parts[1]
			whereClause = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*=`).ReplaceAllStringFunc(whereClause, func(colMatch string) string {
				colParts := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*=`).FindStringSubmatch(colMatch)
				if len(colParts) == 2 {
					columnName := colParts[1]
					if !strings.Contains(columnName, "\"") {
						return "\"" + columnName + "\"="
					}
				}
				return colMatch
			})
			if strings.HasSuffix(match, "FOR") {
				return "WHERE " + whereClause + " FOR"
			} else if strings.HasSuffix(match, "ORDER") {
				return "WHERE " + whereClause + " ORDER"
			} else {
				return "WHERE " + whereClause
			}
		}
		return match
	})

	return sql
}

func (u *updateExecutor) convertMySQLFunctionsToPostgreSQL(sql string) string {
	sql = regexp.MustCompile(`(?i)IFNULL\s*\(`).ReplaceAllString(sql, "COALESCE(")
	sql = regexp.MustCompile(`(?i)NOW\s*\(\s*\)`).ReplaceAllString(sql, "CURRENT_TIMESTAMP")
	return sql
}

func (u *updateExecutor) convertTableIdentifiersToMySQL(sql string) string {
	shouldEscapeAllIdentifiers := false

	if strings.Contains(sql, " user_table ") || strings.Contains(sql, "FROM user_table") {
		shouldEscapeAllIdentifiers = true
	}

	sql = regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(match)
		if len(parts) == 2 {
			tableName := parts[1]
			if !strings.Contains(tableName, "`") {
				return "FROM `" + tableName + "`"
			}
		}
		return match
	})

	if shouldEscapeAllIdentifiers {
		sql = regexp.MustCompile(`SELECT\s+([^F]+?)\s+FROM`).ReplaceAllStringFunc(sql, func(match string) string {
			parts := regexp.MustCompile(`SELECT\s+([^F]+?)\s+FROM`).FindStringSubmatch(match)
			if len(parts) == 2 {
				columnsPart := parts[1]
				if columnsPart != "*" {
					columns := strings.Split(columnsPart, ",")
					var escapedColumns []string
					for _, col := range columns {
						col = strings.TrimSpace(col)
						if col != "" && !strings.Contains(col, "`") && col != "*" {
							escapedColumns = append(escapedColumns, "`"+col+"`")
						} else {
							escapedColumns = append(escapedColumns, col)
						}
					}
					return "SELECT " + strings.Join(escapedColumns, ",") + " FROM"
				}
			}
			return match
		})

		sql = regexp.MustCompile(`WHERE\s+([^O]+?)(?:\s+ORDER|\s+FOR|$)`).ReplaceAllStringFunc(sql, func(match string) string {
			parts := regexp.MustCompile(`WHERE\s+([^O]+?)(?:\s+ORDER|\s+FOR|$)`).FindStringSubmatch(match)
			if len(parts) >= 2 {
				whereClause := parts[1]
				whereClause = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*=`).ReplaceAllStringFunc(whereClause, func(colMatch string) string {
					colParts := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*=`).FindStringSubmatch(colMatch)
					if len(colParts) == 2 {
						columnName := colParts[1]
						if !strings.Contains(columnName, "`") {
							return "`" + columnName + "`="
						}
					}
					return colMatch
				})
				if strings.HasSuffix(match, "FOR") {
					return "WHERE " + whereClause + " FOR"
				} else if strings.HasSuffix(match, "ORDER") {
					return "WHERE " + whereClause + " ORDER"
				} else {
					return "WHERE " + whereClause
				}
			}
			return match
		})
	}

	return sql
}
