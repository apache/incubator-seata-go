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

	"github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

// multiUpdateExecutor execute multiple update SQL
type multiUpdateExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
	dbType      types.DBType
}

var rows driver.Rows
var comma = ","

// NewMultiUpdateExecutor get new multi update executor
func NewMultiUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) *multiUpdateExecutor {
	return &multiUpdateExecutor{
		parserCtx:    parserCtx,
		execContext:  execContext,
		baseExecutor: baseExecutor{hooks: hooks},
		dbType:       execContext.TxCtx.DBType,
	}
}

// ExecContext exec SQL, and generate before image and after image
func (u *multiUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	u.beforeHooks(ctx, u.execContext)
	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()

	//single update sql handler
	if len(u.parserCtx.MultiStmt) == 1 {
		u.parserCtx.UpdateStmt = u.parserCtx.MultiStmt[0].UpdateStmt
		return NewUpdateExecutor(u.parserCtx, u.execContext, u.hooks).ExecContext(ctx, f)
	}
	beforeImages, err := u.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, u.execContext.Query, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImages, err := u.afterImage(ctx, beforeImages)
	if err != nil {
		return nil, err
	}

	if len(afterImages) != len(beforeImages) {
		return nil, errors.New("Before image size is not equaled to after image size, probably because you updated the primary keys.")
	}

	for i, afterImage := range afterImages {
		beforeImage := afterImages[i]
		if len(beforeImage.Rows) != len(afterImage.Rows) {
			return nil, errors.New("Before image size is not equaled to after image size, probably because you updated the primary keys.")
		}

		u.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
		u.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	}

	return res, nil
}

func (u *multiUpdateExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	tableName, err := u.getFromTableInSQL()
	if err != nil {
		return nil, err
	}
	metaData, err := datasource.GetTableCache(u.dbType).GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}

	// use
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(u.execContext.NamedValues, metaData)
	if err != nil {
		return nil, err
	}

	rows, err := u.rowsPrepare(ctx, selectSQL, selectArgs)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("rows close fail, err:%v", err)
			return
		}
	}()
	if err != nil {
		return nil, err
	}

	image, err := u.buildRecordImages(ctx, u.execContext, rows, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey(image, *metaData)
	u.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	image.SQLType = u.parserCtx.SQLType

	return []*types.RecordImage{image}, nil
}

func (u *multiUpdateExecutor) afterImage(ctx context.Context, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	if len(beforeImages) == 0 {
		return nil, errors.New("empty beforeImages")
	}
	beforeImage := beforeImages[0]

	tableName, err := u.getFromTableInSQL()
	if err != nil {
		return nil, err
	}
	metaData, err := datasource.GetTableCache(u.dbType).GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}

	// use
	selectSQL, selectArgs := u.buildAfterImageSQL(beforeImage, *metaData)

	rows, err = u.rowsPrepare(ctx, selectSQL, selectArgs)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("rows close fail, err:%v", err)
			return
		}
	}()
	if err != nil {
		return nil, err
	}

	image, err := u.buildRecordImages(ctx, u.execContext, rows, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}

	image.SQLType = u.parserCtx.SQLType
	return []*types.RecordImage{image}, nil
}

func (u *multiUpdateExecutor) rowsPrepare(ctx context.Context, selectSQL string, selectArgs []driver.NamedValue) (driver.Rows, error) {
	var queryer driver.Queryer

	queryerContext, ok := u.execContext.Conn.(driver.QueryerContext)
	if !ok {
		queryer, ok = u.execContext.Conn.(driver.Queryer)
	}
	if ok {
		var err error
		rows, err = util.CtxDriverQuery(ctx, queryerContext, queryer, selectSQL, selectArgs)

		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, errors.New("invalid conn")
	}
	return rows, nil
}

// buildAfterImageSQL build the SQL to query after image data
func (u *multiUpdateExecutor) buildAfterImageSQL(beforeImage *types.RecordImage, meta types.TableMeta) (string, []driver.NamedValue) {
	if !u.isAstStmtValid() {
		return "", nil
	}

	selectSql := strings.Builder{}
	selectFields := make([]string, 0, len(meta.ColumnNames))
	var selectFieldsStr string
	var fieldsExits = make(map[string]struct{})
	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, row := range beforeImage.Rows {
			for _, column := range row.Columns {
				if _, exist := fieldsExits[column.ColumnName]; exist {
					continue
				}

				fieldsExits[column.ColumnName] = struct{}{}
				selectFields = append(selectFields, column.ColumnName)
			}
		}
		selectFieldsStr = strings.Join(selectFields, comma)
	} else {
		selectFieldsStr = strings.Join(meta.ColumnNames, comma)
	}
	selectSql.WriteString("SELECT " + selectFieldsStr + " FROM " + u.escapeIdentifier(meta.TableName, true) + " WHERE ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), u.execContext.DBType, maxInSize)
	selectSql.WriteString(" " + whereSQL + " ")
	return selectSql.String(), u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName())
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *multiUpdateExecutor) buildBeforeImageSQL(args []driver.NamedValue, meta *types.TableMeta) (string, []driver.NamedValue, error) {
	if !u.isAstStmtValid() {
		log.Errorf("invalid multi update stmt")
		return "", nil, errors.New("invalid muliti update stmt")
	}

	var (
		whereCondition strings.Builder
		multiStmts     = u.parserCtx.MultiStmt
		newArgs        = make([]driver.NamedValue, 0, len(u.parserCtx.MultiStmt))
		fields         = make([]*ast.SelectField, 0, len(meta.ColumnNames))
		fieldsExits    = make(map[string]struct{}, len(meta.ColumnNames))
	)

	argOffset := 0
	for _, multiStmt := range u.parserCtx.MultiStmt {
		updateStmt := multiStmt.UpdateStmt

		if updateStmt != nil {
			if updateStmt.Limit != nil {
				return "", nil, fmt.Errorf("multi update SQL with limit condition is not support yet")
			}
			if updateStmt.Order != nil {
				return "", nil, fmt.Errorf("multi update SQL with orderBy condition is not support yet")
			}
		} else if multiStmt.AuxtenUpdateStmt == nil {
			log.Errorf("No valid update statement found in multiStmt")
			continue
		}

		if undo.UndoConfig.OnlyCareUpdateColumns && updateStmt != nil {
			for _, column := range updateStmt.List {
				if _, exist := fieldsExits[column.Column.String()]; exist {
					continue
				}

				fieldsExits[column.Column.String()] = struct{}{}
				fields = append(fields, &ast.SelectField{Expr: &ast.ColumnNameExpr{Name: column.Column}})
			}
		} else if undo.UndoConfig.OnlyCareUpdateColumns && multiStmt.AuxtenUpdateStmt != nil {
			auxtenStmt := multiStmt.AuxtenUpdateStmt
			for _, expr := range auxtenStmt.Exprs {
				columnName := strings.Trim(string(expr.Names[0].String()), `"`)
				if _, exist := fieldsExits[columnName]; exist {
					continue
				}

				fieldsExits[columnName] = struct{}{}
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.CIStr{O: columnName}}},
				})
			}

			for _, columnName := range meta.GetPrimaryKeyOnlyName() {
				if _, exist := fieldsExits[columnName]; exist {
					continue
				}

				fieldsExits[columnName] = struct{}{}
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.CIStr{O: columnName, L: columnName}}},
				})
			}
		} else {
			fields = make([]*ast.SelectField, 0, len(meta.ColumnNames))
			for _, column := range meta.ColumnNames {
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.CIStr{O: column}}}})
			}
		}

		if updateStmt != nil {
			tmpSelectStmt := ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				From:           updateStmt.TableRefs,
				Where:          updateStmt.Where,
				Fields:         &ast.FieldList{Fields: fields},
				OrderBy:        updateStmt.Order,
				Limit:          updateStmt.Limit,
				TableHints:     updateStmt.TableHints,
				LockInfo: &ast.SelectLockInfo{
					LockType: ast.SelectLockForUpdate,
				},
			}
			newArgs = append(newArgs, u.buildSelectArgs(&tmpSelectStmt, args)...)

			in := bytes.NewByteBuffer([]byte{})
			_ = updateStmt.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, in))

			if whereCondition.Len() > 0 {
				whereCondition.Write([]byte(" OR "))
			}
			whereCondition.Write(in.Bytes())
		} else if multiStmt.AuxtenUpdateStmt != nil {
			auxtenStmt := multiStmt.AuxtenUpdateStmt
			if auxtenStmt.OrderBy != nil {
				return "", nil, fmt.Errorf("multi update SQL with orderBy condition is not support yet")
			}
			
			stmtStr := tree.AsString(auxtenStmt)
			if strings.Contains(strings.ToUpper(stmtStr), "LIMIT") {
				return "", nil, fmt.Errorf("multi update SQL with limit condition is not support yet")
			}
			if auxtenStmt.Where != nil {
				whereStr := tree.AsString(auxtenStmt.Where)
				whereStr = strings.TrimPrefix(whereStr, "WHERE ")
				if whereCondition.Len() > 0 {
					whereCondition.Write([]byte(" OR "))
				}
				whereCondition.WriteString(whereStr)

				setClauseStr := ""
				for i, expr := range auxtenStmt.Exprs {
					if i > 0 {
						setClauseStr += ", "
					}
					setClauseStr += tree.AsString(expr)
				}
				setClauseParamCount := strings.Count(setClauseStr, "$")
				whereParamCount := strings.Count(whereStr, "$") 
				if whereParamCount > 0 {
					startIdx := argOffset + setClauseParamCount
					endIdx := argOffset + setClauseParamCount + whereParamCount
					if endIdx <= len(args) {
						stmtArgs := args[startIdx:endIdx]
						newArgs = append(newArgs, stmtArgs...)
					}
				}
				totalStmtParams := setClauseParamCount + whereParamCount
				argOffset += totalStmtParams
			}
		}
	}

	if undo.UndoConfig.OnlyCareUpdateColumns && u.dbType == types.DBTypePostgreSQL {
		for _, columnName := range meta.GetPrimaryKeyOnlyName() {
			if _, exist := fieldsExits[columnName]; exist {
				continue
			}
			fieldsExits[columnName] = struct{}{}
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.CIStr{O: columnName, L: columnName}}},
			})
		}
	}

	if whereCondition.Len() == 0 {
		if u.dbType == types.DBTypePostgreSQL {
			tableName, err := u.getFromTableInSQL()
			if err != nil {
				return "", nil, err
			}

			selStmt := ast.SelectStmt{
				SelectStmtOpts: &ast.SelectStmtOpts{},
				From: &ast.TableRefsClause{
					TableRefs: &ast.Join{
						Left: &ast.TableSource{
							Source: &ast.TableName{Name: model.CIStr{O: tableName}},
						},
					},
				},
				Fields: &ast.FieldList{Fields: fields},
				LockInfo: &ast.SelectLockInfo{
					LockType: ast.SelectLockForUpdate,
				},
			}

			b := bytes.NewByteBuffer([]byte{})
			selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
			sql := string(b.Bytes())
			sql = u.adaptSQLSyntax(sql)

			if u.dbType == types.DBTypePostgreSQL {
				for idx := range newArgs {
					newArgs[idx].Ordinal = idx
				}
			}

			return sql, newArgs, nil
		} else {
			log.Errorf("No where conditions found")
			return "", nil, fmt.Errorf("no where conditions found")
		}
	}

	whereStr := whereCondition.String()
	if u.dbType == types.DBTypePostgreSQL {
		whereStr = regexp.MustCompile(`\bILIKE\b`).ReplaceAllString(whereStr, "LIKE")
	}
	fakeSql := "select * from t where " + whereStr
	fakeStmt, err := parser.New().ParseOneStmt(fakeSql, "", "")
	if err != nil {
		log.Errorf("multi update parse fake sql error")
		return "", nil, err
	}
	fakeNode, ok := fakeStmt.Accept(&updateVisitor{})
	if !ok {
		log.Errorf("multi update accept update visitor error")
		return "", nil, err
	}
	fakeSelectStmt, ok := fakeNode.(*ast.SelectStmt)
	if !ok {
		log.Errorf("multi update fake node is not select stmt")
		return "", nil, err
	}

	// Find the first valid UpdateStmt for TableRefs and TableHints
	var tableRefs *ast.TableRefsClause
	var tableHints []*ast.TableOptimizerHint
	
	for _, multiStmt := range multiStmts {
		if multiStmt.UpdateStmt != nil {
			tableRefs = multiStmt.UpdateStmt.TableRefs
			tableHints = multiStmt.UpdateStmt.TableHints
			break
		}
	}
	
	// If no UpdateStmt found, create default TableRefs from table name
	if tableRefs == nil {
		tableName, err := u.getFromTableInSQL()
		if err != nil {
			return "", nil, err
		}
		
		tableRefs = &ast.TableRefsClause{
			TableRefs: &ast.Join{
				Left: &ast.TableSource{
					Source: &ast.TableName{Name: model.CIStr{O: tableName}},
				},
			},
		}
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           tableRefs,
		Where:          fakeSelectStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		TableHints:     tableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := bytes.NewByteBuffer([]byte{})
	selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	log.Infof("build select sql by update sourceQuery, sql {}", string(b.Bytes()))

	sql := string(b.Bytes())
	sql = u.adaptSQLSyntax(sql)
	
	if u.dbType == types.DBTypePostgreSQL && strings.Contains(whereCondition.String(), "ILIKE") {
		sql = regexp.MustCompile(`\b(\w+)\s+LIKE\s+(\$\d+)`).ReplaceAllString(sql, "$1 ILIKE $2")
	}

	if u.dbType == types.DBTypePostgreSQL {
		oldParamNumbers := make([]int, len(newArgs))
		for idx := range newArgs {
			oldParamNumbers[idx] = newArgs[idx].Ordinal + 1
			newArgs[idx].Ordinal = idx
		}
		
		for idx, oldNum := range oldParamNumbers {
			oldParam := fmt.Sprintf("$%d", oldNum)
			newParam := fmt.Sprintf("$%d", idx+1)
			sql = strings.ReplaceAll(sql, oldParam, newParam)
		}
	}

	return sql, newArgs, nil
}

func (u *multiUpdateExecutor) isAstStmtValid() bool {
	return u.parserCtx != nil && u.parserCtx.MultiStmt != nil && len(u.parserCtx.MultiStmt) > 0
}

type updateVisitor struct {
	stmt *ast.UpdateStmt
}

func (m *updateVisitor) Enter(n ast.Node) (ast.Node, bool) {
	return n, true
}

func (m *updateVisitor) Leave(n ast.Node) (ast.Node, bool) {
	node := n
	return node, true
}

func (u *multiUpdateExecutor) getFromTableInSQL() (string, error) {
	for _, parser := range u.parserCtx.MultiStmt {
		if parser != nil {
			return parser.GetTableName()
		}
	}

	if u.parserCtx != nil {
		return u.parserCtx.GetTableName()
	}

	return "", fmt.Errorf("multi update sql has no table name")
}

func (u *multiUpdateExecutor) escapeIdentifier(name string, isTableName bool) string {
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
		return "\"" + name + "\""
	default:
		return name
	}
}

func (u *multiUpdateExecutor) adaptSQLSyntax(sql string) string {
	switch u.dbType {
	case types.DBTypePostgreSQL:
		return u.adaptPostgreSQLSyntax(sql)
	case types.DBTypeMySQL:
		return u.adaptMySQLSyntax(sql)
	}
	return sql
}

func (u *multiUpdateExecutor) adaptPostgreSQLSyntax(sql string) string {
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

	sql = regexp.MustCompile(`\(\(([^)]+)\) AND \(([^)]+)\)\) AND \(([^)]+)\)`).ReplaceAllString(sql, "$1 AND $2 AND $3")
	sql = regexp.MustCompile(`\(\(([^)]+)\) AND \(([^)]+)\)\)`).ReplaceAllString(sql, "$1 AND $2")


	if !strings.Contains(sql, "$") {
		counter := 1
		sql = regexp.MustCompile(`\?`).ReplaceAllStringFunc(sql, func(match string) string {
			result := fmt.Sprintf("$%d", counter)
			counter++
			return result
		})
	}

	if u.dbType == types.DBTypePostgreSQL {
		sql = u.convertMySQLIdentifiersToPostgreSQL(sql)
	}
	sql = u.convertMySQLFunctionsToPostgreSQL(sql)

	return sql
}

func (u *multiUpdateExecutor) adaptMySQLSyntax(sql string) string {
	sql = regexp.MustCompile(`(_UTF8MB4)(\w+)`).ReplaceAllString(sql, "${1}'${2}'")
	return sql
}

func (u *multiUpdateExecutor) isPostgreSQLIdentifier(word string) bool {
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

func (u *multiUpdateExecutor) convertMySQLIdentifiersToPostgreSQL(sql string) string {
	sql = regexp.MustCompile("`([^`]+)`").ReplaceAllString(sql, "\"$1\"")
	
	sql = regexp.MustCompile(`SELECT\s+(.*?)\s+FROM`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`SELECT\s+(.*?)\s+FROM`).FindStringSubmatch(match)
		if len(parts) == 2 {
			columns := parts[1]
			if strings.Contains(columns, "SQL_NO_CACHE") {
				return match
			}
			
			columnParts := strings.Split(columns, ",")
			var quotedColumns []string
			for _, col := range columnParts {
				col = strings.TrimSpace(col)
				if !strings.Contains(col, "\"") && !strings.Contains(col, "*") {
					col = "\"" + col + "\""
				}
				quotedColumns = append(quotedColumns, col)
			}
			return "SELECT " + strings.Join(quotedColumns, ",") + " FROM"
		}
		return match
	})

	sql = regexp.MustCompile(`FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+WHERE`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := regexp.MustCompile(`FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+WHERE`).FindStringSubmatch(match)
		if len(parts) == 2 {
			tableName := parts[1]
			if !strings.Contains(tableName, "\"") {
				tableName = "\"" + tableName + "\""
			}
			return "FROM " + tableName + " WHERE"
		}
		return match
	})
	
	return sql
}

func (u *multiUpdateExecutor) convertMySQLFunctionsToPostgreSQL(sql string) string {
	sql = regexp.MustCompile(`(?i)IFNULL\s*\(`).ReplaceAllString(sql, "COALESCE(")
	sql = regexp.MustCompile(`(?i)NOW\s*\(\s*\)`).ReplaceAllString(sql, "CURRENT_TIMESTAMP")
	return sql
}
