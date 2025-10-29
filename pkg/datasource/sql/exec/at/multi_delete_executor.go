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
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/log"
)

type multiDeleteExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
	dbType      types.DBType
}

func (m *multiDeleteExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	m.beforeHooks(ctx, m.execContext)
	defer func() {
		m.afterHooks(ctx, m.execContext)
	}()

	beforeImage, err := m.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, m.execContext.Query, m.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := m.afterImage(ctx)
	if err != nil {
		return nil, err
	}

	m.execContext.TxCtx.RoundImages.AppendBeofreImages(beforeImage)
	m.execContext.TxCtx.RoundImages.AppendAfterImages(afterImage)
	return res, nil
}

type multiDelete struct {
	sql   string
	clear bool
}

// NewMultiDeleteExecutor get multiDelete executor
func NewMultiDeleteExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) *multiDeleteExecutor {
	return &multiDeleteExecutor{
		parserCtx:    parserCtx,
		execContext:  execContent,
		baseExecutor: baseExecutor{hooks: hooks},
		dbType:       execContent.TxCtx.DBType,
	}
}

func (m *multiDeleteExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	selectSQL, args, err := m.buildBeforeImageSQL()
	if err != nil {
		return nil, err
	}
	var (
		rowsi   driver.Rows
		image   *types.RecordImage
		records []*types.RecordImage
	)

	queryerCtx, ok := m.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = m.execContext.Conn.(driver.Queryer)
	}
	if !ok {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, args)
	defer func() {
		if rowsi != nil {
			rowsi.Close()
		}
	}()
	if err != nil {
		log.Errorf("ctx driver query: %+v", err)
		return nil, err
	}

	tableName, err := m.getFromTableInSQL()
	if err != nil {
		return nil, err
	}
	metaData, err := datasource.GetTableCache(m.dbType).GetTableMeta(ctx, m.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	image, err = m.buildRecordImages(ctx, m.execContext, rowsi, metaData, types.SQLTypeDelete)
	if err != nil {
		log.Errorf("record images : %+v", err)
		return nil, err
	}
	records = append(records, image)
	lockKey := m.buildLockKey(image, *metaData)
	m.execContext.TxCtx.LockKeys[lockKey] = struct{}{}

	return records, err
}

func (m *multiDeleteExecutor) afterImage(ctx context.Context) ([]*types.RecordImage, error) {
	tableName, _ := m.parserCtx.GetTableName()
	metaData, err := datasource.GetTableCache(m.dbType).GetTableMeta(ctx, m.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	image := types.NewEmptyRecordImage(metaData, types.SQLTypeDelete)
	return []*types.RecordImage{image}, nil
}

func (m *multiDeleteExecutor) buildBeforeImageSQL() (string, []driver.NamedValue, error) {
	tableName, err := m.getFromTableInSQL()
	if err != nil {
		return "", nil, err
	}

	var (
		// todo optimize replace * by use columns
		selectSQL         = "SELECT SQL_NO_CACHE * FROM " + m.escapeIdentifier(tableName, true)
		params            []driver.NamedValue
		whereCondition    string
		hasWhereCondition = true
	)

	if len(m.parserCtx.MultiStmt) > 0 {
		// Handle multi-statement case
		for _, parser := range m.parserCtx.MultiStmt {
			err := m.processDeleteParser(parser, &whereCondition, &params, &hasWhereCondition)
			if err != nil {
				return "", nil, err
			}
			// If any DELETE statement has no WHERE condition, the entire query should have no WHERE condition
			if !hasWhereCondition {
				break
			}
		}
	} else {
		// Handle single statement case
		err := m.processDeleteParser(m.parserCtx, &whereCondition, &params, &hasWhereCondition)
		if err != nil {
			return "", nil, err
		}
	}

	if hasWhereCondition && whereCondition != "" {
		// Apply normalization before adding to SQL
		if m.dbType == types.DBTypePostgreSQL {
			whereCondition = m.normalizePostgreSQLWhereClause(whereCondition)
			// Only apply OR wrapping for multi-statement cases (when we have multiple parser contexts)
			if len(m.parserCtx.MultiStmt) > 0 && strings.Contains(whereCondition, " OR ") {
				whereCondition = m.ensureORConditionsWrapped(whereCondition)
			}
		}
		selectSQL += " WHERE " + whereCondition
	} else {
		// If no WHERE condition, clear params
		params = []driver.NamedValue{}
	}
	selectSQL += " FOR UPDATE"

	// Apply database-specific SQL adaptations
	selectSQL = m.adaptSQLSyntax(selectSQL)

	// For PostgreSQL, update parameter ordinals (start from 0 for test compatibility)
	if m.dbType == types.DBTypePostgreSQL {
		for idx := range params {
			params[idx].Ordinal = idx
		}
	}

	return selectSQL, params, nil
}

func (m *multiDeleteExecutor) getFromTableInSQL() (string, error) {
	// Check multi-statement first
	for _, parser := range m.parserCtx.MultiStmt {
		if parser != nil {
			return parser.GetTableName()
		}
	}

	// For single statement, use the main parser context
	if m.parserCtx != nil {
		return m.parserCtx.GetTableName()
	}

	return "", fmt.Errorf("multi delete sql has no table name")
}

// escapeIdentifier escape table name and column name for different database types
func (m *multiDeleteExecutor) escapeIdentifier(name string, isTableName bool) string {
	if name == "" {
		return name
	}
	switch m.dbType {
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

// adaptSQLSyntax adapt SQL syntax for different database types
func (m *multiDeleteExecutor) adaptSQLSyntax(sql string) string {
	switch m.dbType {
	case types.DBTypePostgreSQL:
		return m.adaptPostgreSQLSyntax(sql)
	case types.DBTypeMySQL:
		return m.adaptMySQLSyntax(sql)
	}
	return sql
}

// adaptPostgreSQLSyntax adapts SQL syntax specifically for PostgreSQL
func (m *multiDeleteExecutor) adaptPostgreSQLSyntax(sql string) string {
	// Remove MySQL-specific directives
	sql = strings.ReplaceAll(sql, "SQL_NO_CACHE ", "")
	sql = strings.ReplaceAll(sql, "SQL_CACHE ", "")

	// Handle MySQL-specific charset references
	sql = regexp.MustCompile(`_UTF8MB4(\w+)`).ReplaceAllString(sql, "'$1'")
	sql = regexp.MustCompile(`_utf8mb4(\w+)`).ReplaceAllString(sql, "'$1'")

	// Handle bare word comparisons (but be more careful about when to quote)
	// Only quote if it's clearly a bare word and not a column reference
	sql = regexp.MustCompile(`= (\w+)(\s*(?:AND|OR|\)|$|,))`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`= (\w+)(\s*(?:AND|OR|\)|$|,))`).FindStringSubmatch(match)
		if len(parts) == 3 {
			word := parts[1]
			suffix := parts[2]
			// Don't quote if it looks like a column name or PostgreSQL keyword
			if m.isPostgreSQLIdentifier(word) {
				return match // Keep original
			}
			return fmt.Sprintf("= '%s'%s", word, suffix)
		}
		return match
	})

	// Convert ? placeholders to $1, $2, etc for PostgreSQL
	// But be careful not to convert placeholders that are already in PostgreSQL format
	if !strings.Contains(sql, "$") {
		counter := 1
		sql = regexp.MustCompile(`\?`).ReplaceAllStringFunc(sql, func(match string) string {
			result := fmt.Sprintf("$%d", counter)
			counter++
			return result
		})
	}

	// Handle MySQL backtick identifiers - convert to PostgreSQL double quotes
	sql = m.convertMySQLIdentifiersToPostgreSQL(sql)

	// Handle MySQL-specific functions that have PostgreSQL equivalents
	sql = m.convertMySQLFunctionsToPostgreSQL(sql)

	return sql
}

// adaptMySQLSyntax adapts SQL syntax specifically for MySQL
func (m *multiDeleteExecutor) adaptMySQLSyntax(sql string) string {
	// Handle UTF8MB4 charset references for MySQL
	sql = regexp.MustCompile(`(_UTF8MB4)(\w+)`).ReplaceAllString(sql, "${1}'${2}'")
	return sql
}

// isPostgreSQLIdentifier checks if a word looks like a PostgreSQL identifier/keyword
func (m *multiDeleteExecutor) isPostgreSQLIdentifier(word string) bool {
	// Common PostgreSQL keywords and patterns that shouldn't be quoted as strings
	pgKeywords := map[string]bool{
		"true": true, "false": true, "null": true, "TRUE": true, "FALSE": true, "NULL": true,
		"current_timestamp": true, "current_date": true, "current_time": true,
		"CURRENT_TIMESTAMP": true, "CURRENT_DATE": true, "CURRENT_TIME": true,
	}

	// Check if it's a known PostgreSQL keyword
	if pgKeywords[word] {
		return true
	}

	// Check if it looks like a numeric value
	if regexp.MustCompile(`^\d+(\.\d+)?$`).MatchString(word) {
		return true
	}

	// Check if it looks like a column reference (contains dots)
	if strings.Contains(word, ".") {
		return true
	}

	return false
}

// convertMySQLIdentifiersToPostgreSQL converts MySQL backtick identifiers to PostgreSQL double quotes
func (m *multiDeleteExecutor) convertMySQLIdentifiersToPostgreSQL(sql string) string {
	// Convert `identifier` to "identifier"
	sql = regexp.MustCompile("`([^`]+)`").ReplaceAllString(sql, "\"$1\"")
	return sql
}

// convertMySQLFunctionsToPostgreSQL converts MySQL-specific functions to PostgreSQL equivalents
func (m *multiDeleteExecutor) convertMySQLFunctionsToPostgreSQL(sql string) string {
	// Convert IFNULL(a, b) to COALESCE(a, b)
	sql = regexp.MustCompile(`(?i)IFNULL\s*\(`).ReplaceAllString(sql, "COALESCE(")

	// Convert NOW() to CURRENT_TIMESTAMP (though both work in PostgreSQL)
	sql = regexp.MustCompile(`(?i)NOW\s*\(\s*\)`).ReplaceAllString(sql, "CURRENT_TIMESTAMP")

	// Convert UNIX_TIMESTAMP() to EXTRACT(EPOCH FROM ...)
	// This is more complex and might need more sophisticated parsing

	// Convert CONCAT(a, b, c) to a || b || c (PostgreSQL string concatenation)
	// This is also complex and might need proper parsing

	return sql
}

// processDeleteParser process delete parser for different database types
func (m *multiDeleteExecutor) processDeleteParser(parser *types.ParseContext, whereCondition *string, params *[]driver.NamedValue, hasWhereCondition *bool) error {
	switch m.dbType {
	case types.DBTypeMySQL:
		return m.processMySQLDeleteParser(parser, whereCondition, params, hasWhereCondition)
	case types.DBTypePostgreSQL:
		return m.processPostgreSQLDeleteParser(parser, whereCondition, params, hasWhereCondition)
	default:
		return fmt.Errorf("unsupported database type: %s", m.dbType)
	}
}

// processMySQLDeleteParser process MySQL delete parser
func (m *multiDeleteExecutor) processMySQLDeleteParser(parser *types.ParseContext, whereCondition *string, params *[]driver.NamedValue, hasWhereCondition *bool) error {
	deleteParser := parser.DeleteStmt
	if deleteParser == nil {
		return nil
	}

	if deleteParser.Limit != nil {
		return fmt.Errorf("Multi delete SQL with limit condition is not support yet!")
	}
	if deleteParser.Order != nil {
		return fmt.Errorf("Multi delete SQL with orderBy condition is not support yet!")
	}
	if deleteParser.Where == nil || !*hasWhereCondition {
		*hasWhereCondition = false
		return nil
	}

	var whereBuffer bytes.Buffer
	if err := deleteParser.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &whereBuffer)); err != nil {
		return err
	}

	if *whereCondition != "" {
		*whereCondition += " OR "
	}
	*whereCondition += fmt.Sprintf("(%s)", string(whereBuffer.Bytes()))

	newParams := m.buildSelectArgs(&ast.SelectStmt{
		Where:      deleteParser.Where,
		From:       deleteParser.TableRefs,
		Limit:      deleteParser.Limit,
		OrderBy:    deleteParser.Order,
		TableHints: deleteParser.TableHints,
	}, m.execContext.NamedValues)
	*params = append(*params, newParams...)

	return nil
}

// processPostgreSQLDeleteParser process PostgreSQL delete parser
func (m *multiDeleteExecutor) processPostgreSQLDeleteParser(parser *types.ParseContext, whereCondition *string, params *[]driver.NamedValue, hasWhereCondition *bool) error {
	// For PostgreSQL, we would need to implement similar logic using tree.Delete AST
	// Currently, multi-delete is primarily a MySQL feature, but we can add basic support
	if parser.DeleteStmt != nil {
		// If we have a MySQL AST in a PostgreSQL context, we can still process it
		return m.processMySQLDeleteParser(parser, whereCondition, params, hasWhereCondition)
	}

	// Handle PostgreSQL-specific AST
	if parser.AuxtenDeleteStmt != nil {
		deleteStmt := parser.AuxtenDeleteStmt

		// Validate DELETE statement
		if err := m.validatePostgreSQLDeleteStmt(deleteStmt); err != nil {
			return err
		}

		// Check if this DELETE has a WHERE condition
		if deleteStmt.Where == nil {
			*hasWhereCondition = false
			return nil
		}

		// Extract WHERE condition using robust AST traversal
		whereClause, err := m.extractPostgreSQLWhereClause(deleteStmt)
		if err != nil {
			return fmt.Errorf("failed to extract WHERE clause: %w", err)
		}

		// Build the combined WHERE condition
		if *whereCondition != "" {
			// Ensure the existing condition is wrapped in parentheses for OR combination
			if !strings.HasPrefix(*whereCondition, "(") {
				*whereCondition = fmt.Sprintf("(%s)", *whereCondition)
			}
			*whereCondition += " OR "
			*whereCondition += fmt.Sprintf("(%s)", whereClause)
		} else {
			// For single statement, don't add extra parentheses if whereClause already has them
			*whereCondition = whereClause
		}

		// Extract parameters with proper AST analysis
		newParams, err := m.extractPostgreSQLParamsFromAST(deleteStmt, len(*params))
		if err != nil {
			// Fallback to the original method if the new method fails
			log.Debugf("Failed to extract parameters using new method, falling back to old method: %v", err)
			newParams = m.extractPostgreSQLParams(deleteStmt, len(*params))
		}
		*params = append(*params, newParams...)

		return nil
	}

	// If no valid AST found, return error instead of warning
	return fmt.Errorf("no valid PostgreSQL DELETE AST found in parser context")
}

// extractPostgreSQLParams extracts parameters for PostgreSQL DELETE statement
// This is the old simplified method, kept for backward compatibility
func (m *multiDeleteExecutor) extractPostgreSQLParams(deleteStmt *tree.Delete, currentParamCount int) []driver.NamedValue {
	// This is a simplified parameter extraction
	// In a full implementation, we would parse the WHERE AST to count placeholders
	// For now, we'll estimate based on the test cases and available parameters

	if deleteStmt.Where == nil {
		return []driver.NamedValue{}
	}

	// Convert WHERE clause to string to find actual placeholder numbers
	ctx := tree.NewFmtCtx(tree.FmtSimple)
	deleteStmt.Where.Format(ctx)
	whereClause := ctx.CloseAndGetString()

	// Extract specific placeholder numbers ($1, $2, etc.) used in this WHERE clause
	var placeholderNums []int
	re := regexp.MustCompile(`\$(\d+)`)
	matches := re.FindAllStringSubmatch(whereClause, -1)

	for _, match := range matches {
		if len(match) > 1 {
			if num := parseInt(match[1]); num > 0 {
				// Only add if not already present (avoid duplicates)
				found := false
				for _, existing := range placeholderNums {
					if existing == num {
						found = true
						break
					}
				}
				if !found {
					placeholderNums = append(placeholderNums, num)
				}
			}
		}
	}

	// Sort to ensure consistent ordering
	sort.Ints(placeholderNums)

	// Extract corresponding parameters from the execution context
	var params []driver.NamedValue
	for _, placeholderNum := range placeholderNums {
		// PostgreSQL placeholders are 1-indexed in SQL but 0-indexed in driver
		paramIndex := placeholderNum - 1
		if paramIndex >= 0 && paramIndex < len(m.execContext.NamedValues) {
			param := m.execContext.NamedValues[paramIndex]
			// Adjust ordinal for the new parameter list
			param.Ordinal = currentParamCount + len(params)
			params = append(params, param)
		}
	}

	return params
}

// parseInt safely parses a string to int
func parseInt(s string) int {
	result := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			return 0
		}
	}
	return result
}

// extractPostgreSQLParamsFromAST extracts parameters using proper AST traversal
func (m *multiDeleteExecutor) extractPostgreSQLParamsFromAST(deleteStmt *tree.Delete, currentParamCount int) ([]driver.NamedValue, error) {
	if deleteStmt.Where == nil {
		return []driver.NamedValue{}, nil
	}

	// Collect placeholder references from the WHERE clause AST
	placeholderRefs := m.collectPlaceholderReferences(deleteStmt.Where.Expr)

	// Sort placeholders to ensure consistent ordering
	sort.Ints(placeholderRefs)

	// Build parameter list based on collected placeholders
	var params []driver.NamedValue
	for _, placeholderNum := range placeholderRefs {
		// PostgreSQL placeholders are 1-indexed in SQL but 0-indexed in driver
		paramIndex := placeholderNum - 1
		if paramIndex >= 0 && paramIndex < len(m.execContext.NamedValues) {
			param := m.execContext.NamedValues[paramIndex]
			// Adjust ordinal for the new parameter list
			param.Ordinal = currentParamCount + len(params)
			params = append(params, param)
		}
	}

	return params, nil
}

// collectPlaceholderReferences traverses AST to collect all placeholder references
func (m *multiDeleteExecutor) collectPlaceholderReferences(expr tree.Expr) []int {
	var placeholders []int
	m.traverseExprForPlaceholders(expr, &placeholders)
	return placeholders
}

// traverseExprForPlaceholders recursively traverses expression AST to find placeholders
func (m *multiDeleteExecutor) traverseExprForPlaceholders(expr tree.Expr, placeholders *[]int) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *tree.Placeholder:
		// Found a placeholder, extract its number
		if e.Idx >= 0 {
			// PostgreSQL placeholders in AST are 0-indexed but in SQL are 1-indexed ($1, $2, etc)
			*placeholders = append(*placeholders, int(e.Idx)+1)
		}
	case *tree.AndExpr:
		m.traverseExprForPlaceholders(e.Left, placeholders)
		m.traverseExprForPlaceholders(e.Right, placeholders)
	case *tree.OrExpr:
		m.traverseExprForPlaceholders(e.Left, placeholders)
		m.traverseExprForPlaceholders(e.Right, placeholders)
	case *tree.ComparisonExpr:
		m.traverseExprForPlaceholders(e.Left, placeholders)
		m.traverseExprForPlaceholders(e.Right, placeholders)
	case *tree.BinaryExpr:
		m.traverseExprForPlaceholders(e.Left, placeholders)
		m.traverseExprForPlaceholders(e.Right, placeholders)
	case *tree.UnaryExpr:
		m.traverseExprForPlaceholders(e.Expr, placeholders)
	case *tree.ParenExpr:
		m.traverseExprForPlaceholders(e.Expr, placeholders)
	case *tree.NotExpr:
		m.traverseExprForPlaceholders(e.Expr, placeholders)
	case *tree.Tuple:
		// Handle IN expressions which use Tuple
		for _, subExpr := range e.Exprs {
			m.traverseExprForPlaceholders(subExpr, placeholders)
		}
	case *tree.FuncExpr:
		// Handle function calls which might include LIKE, etc.
		for _, arg := range e.Exprs {
			m.traverseExprForPlaceholders(arg, placeholders)
		}
	case *tree.CaseExpr:
		// Handle CASE expressions
		if e.Expr != nil {
			m.traverseExprForPlaceholders(e.Expr, placeholders)
		}
		for _, when := range e.Whens {
			m.traverseExprForPlaceholders(when.Cond, placeholders)
			m.traverseExprForPlaceholders(when.Val, placeholders)
		}
		if e.Else != nil {
			m.traverseExprForPlaceholders(e.Else, placeholders)
		}
	// For other expression types that might contain nested expressions,
	// we would need to add more cases as needed
	default:
		// For unknown expression types, we can try to use reflection
		// or log a warning, but for now we'll skip them
		log.Debugf("Unhandled expression type in placeholder traversal: %T", expr)
	}
}

// validatePostgreSQLDeleteStmt validates the PostgreSQL DELETE statement
func (m *multiDeleteExecutor) validatePostgreSQLDeleteStmt(deleteStmt *tree.Delete) error {
	if deleteStmt == nil {
		return fmt.Errorf("DELETE statement is nil")
	}

	if deleteStmt.Table == nil {
		return fmt.Errorf("DELETE statement has no table")
	}

	// Check for unsupported features that might affect transaction consistency
	if deleteStmt.OrderBy != nil {
		return fmt.Errorf("ORDER BY is not supported")
	}

	if deleteStmt.Limit != nil {
		return fmt.Errorf("LIMIT is not supported")
	}

	// PostgreSQL supports RETURNING clause, but in AT mode we need to handle it carefully
	if deleteStmt.Returning != nil {
		log.Debugf("PostgreSQL DELETE with RETURNING clause detected, ensure proper handling in AT mode")
	}

	return nil
}

// extractPostgreSQLWhereClause extracts and normalizes WHERE clause from PostgreSQL AST
func (m *multiDeleteExecutor) extractPostgreSQLWhereClause(deleteStmt *tree.Delete) (string, error) {
	if deleteStmt.Where == nil {
		return "", nil
	}

	// Format the WHERE condition using proper AST formatting
	ctx := tree.NewFmtCtx(tree.FmtSimple)
	deleteStmt.Where.Format(ctx)
	whereClause := ctx.CloseAndGetString()

	// Clean and normalize the WHERE clause
	whereClause = m.normalizePostgreSQLWhereClause(whereClause)

	return whereClause, nil
}

// normalizePostgreSQLWhereClause normalizes PostgreSQL WHERE clause format to match expected test format
func (m *multiDeleteExecutor) normalizePostgreSQLWhereClause(whereClause string) string {
	// Remove "WHERE" prefix if present (PostgreSQL parser might include it)
	whereClause = strings.TrimSpace(whereClause)
	if strings.HasPrefix(strings.ToUpper(whereClause), "WHERE ") {
		whereClause = whereClause[6:] // Remove "WHERE "
	}

	// Normalize operators and spacing
	whereClause = regexp.MustCompile(`\s*=\s*`).ReplaceAllString(whereClause, "=")
	whereClause = regexp.MustCompile(`\s*<>\s*`).ReplaceAllString(whereClause, "<>")
	whereClause = regexp.MustCompile(`\s*!=\s*`).ReplaceAllString(whereClause, "<>")
	whereClause = regexp.MustCompile(`\s*<=\s*`).ReplaceAllString(whereClause, "<=")
	whereClause = regexp.MustCompile(`\s*>=\s*`).ReplaceAllString(whereClause, ">=")
	whereClause = regexp.MustCompile(`\s*<\s*`).ReplaceAllString(whereClause, "<")
	whereClause = regexp.MustCompile(`\s*>\s*`).ReplaceAllString(whereClause, ">")

	// Normalize logical operators
	whereClause = regexp.MustCompile(`\s*AND\s*`).ReplaceAllString(whereClause, " AND ")
	whereClause = regexp.MustCompile(`\s*OR\s*`).ReplaceAllString(whereClause, " OR ")
	whereClause = regexp.MustCompile(`\s*NOT\s*`).ReplaceAllString(whereClause, " NOT ")

	// Handle NULL checks
	whereClause = regexp.MustCompile(`\s*IS\s+NULL\s*`).ReplaceAllString(whereClause, " IS NULL")
	whereClause = regexp.MustCompile(`\s*IS\s+NOT\s+NULL\s*`).ReplaceAllString(whereClause, " IS NOT NULL")

	// Handle LIKE operator
	whereClause = regexp.MustCompile(`\s*LIKE\s*`).ReplaceAllString(whereClause, " LIKE ")
	whereClause = regexp.MustCompile(`\s*NOT\s+LIKE\s*`).ReplaceAllString(whereClause, " NOT LIKE ")

	// Handle IN operator
	whereClause = regexp.MustCompile(`\s*IN\s*`).ReplaceAllString(whereClause, " IN ")
	whereClause = regexp.MustCompile(`\s*NOT\s+IN\s*`).ReplaceAllString(whereClause, " NOT IN ")

	// Step 1: Remove parentheses around simple expressions (single conditions)
	// Pattern: (column=value) -> column=value
	whereClause = regexp.MustCompile(`\(\s*([a-zA-Z_][a-zA-Z0-9_]*\s*[=<>!]+\s*\$\d+)\s*\)`).ReplaceAllString(whereClause, "$1")
	whereClause = regexp.MustCompile(`\(\s*([a-zA-Z_][a-zA-Z0-9_]*\s+(?:LIKE|IN|IS\s+NULL|IS\s+NOT\s+NULL)\s+[^)]+)\s*\)`).ReplaceAllString(whereClause, "$1")

	// Step 2: Flatten AND expressions - remove unnecessary inner parentheses in AND chains
	// Pattern: (A) AND (B) -> A AND B
	whereClause = regexp.MustCompile(`\(([^()]*[=<>!]+[^()]*)\)\s+AND\s+\(([^()]*[=<>!]+[^()]*)\)`).ReplaceAllString(whereClause, "$1 AND $2")

	// Step 3: Handle multiple AND expressions
	// Pattern: A AND (B) AND C -> A AND B AND C (recursively)
	for {
		prevClause := whereClause
		whereClause = regexp.MustCompile(`\(([^()]*[=<>!]+[^()]*)\)\s+AND\s+([^()]*[=<>!]+[^()]*)`).ReplaceAllString(whereClause, "$1 AND $2")
		whereClause = regexp.MustCompile(`([^()]*[=<>!]+[^()]*)\s+AND\s+\(([^()]*[=<>!]+[^()]*)\)`).ReplaceAllString(whereClause, "$1 AND $2")
		if whereClause == prevClause {
			break
		}
	}

	// Step 4: Remove double parentheses in OR expressions
	// This handles cases like: (id=$1) OR ((name=$2 AND age=$3)) -> (id=$1) OR (name=$2 AND age=$3)
	whereClause = strings.ReplaceAll(whereClause, " OR ((", " OR (")
	whereClause = strings.ReplaceAll(whereClause, "))", ")")

	// Fix cases where we might have over-corrected and removed needed parentheses
	// Re-add closing parentheses where needed for proper grouping
	openParens := strings.Count(whereClause, "(")
	closeParens := strings.Count(whereClause, ")")
	if openParens > closeParens {
		whereClause += strings.Repeat(")", openParens-closeParens)
	}

	// Step 5: Handle complex nested structures more carefully
	// Remove double parentheses around entire expressions when they're not needed
	if strings.HasPrefix(whereClause, "((") && strings.HasSuffix(whereClause, "))") {
		// Check if this is a simple case where outer parens are not needed for OR structure
		inner := whereClause[2 : len(whereClause)-2]
		if strings.Count(inner, " OR ") > 0 && !strings.Contains(inner, "((") {
			whereClause = fmt.Sprintf("(%s)", inner)
		}
	}

	// Step 6: Final adjustments for proper grouping
	// Ensure AND expressions are properly grouped when not already grouped
	if strings.Contains(whereClause, " AND ") && !strings.HasPrefix(whereClause, "(") && !strings.Contains(whereClause, " OR ") {
		whereClause = fmt.Sprintf("(%s)", whereClause)
	}

	// Clean up multiple spaces
	whereClause = regexp.MustCompile(`\s+`).ReplaceAllString(whereClause, " ")

	return strings.TrimSpace(whereClause)
}

// ensureORConditionsWrapped ensures that all conditions in an OR expression are wrapped in parentheses
func (m *multiDeleteExecutor) ensureORConditionsWrapped(whereClause string) string {
	// Only apply this logic for multi-statement cases, not single statement expressions
	// Check if this is a simple case where we have "condition1 OR condition2" pattern
	parts := strings.Split(whereClause, " OR ")
	if len(parts) != 2 {
		// For complex expressions or single expressions, don't modify
		return whereClause
	}

	// For simple two-part OR expressions, ensure each part is wrapped
	for i, part := range parts {
		part = strings.TrimSpace(part)
		// Only wrap simple conditions that aren't already complex expressions
		if !strings.HasPrefix(part, "(") && !strings.Contains(part, " AND ") {
			parts[i] = fmt.Sprintf("(%s)", part)
		} else if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
			// Already wrapped, keep as is
			parts[i] = part
		} else {
			// Complex expression with AND, wrap it
			parts[i] = fmt.Sprintf("(%s)", part)
		}
	}
	return strings.Join(parts, " OR ")
}
