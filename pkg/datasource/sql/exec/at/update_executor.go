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
	"strings"

	"github.com/arana-db/parser/model"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/util/bytes"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

var (
	maxInSize = 1000
)

// updateExecutor execute update SQL
type updateExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

// NewUpdateExecutor get update executor
func NewUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	// Because update join cannot be clearly identified when SQL cannot be parsed
	if parserCtx.UpdateStmt.TableRefs.TableRefs.Right != nil {
		return NewUpdateJoinExecutor(parserCtx, execContent, hooks)
	}
	return &updateExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

// ExecContext exec SQL, and generate before image and after image
func (u *updateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if err := u.beforeHooks(ctx, u.execContext); err != nil {
		return nil, err
	}
	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()
	if err := u.resolveTableMetaKey(ctx, u.execContext, u.parserCtx); err != nil {
		return nil, err
	}

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

	if err := u.prepareUndoPair(u.execContext, beforeImage, afterImage); err != nil {
		return nil, err
	}

	return res, nil
}

// beforeImage build before image
func (u *updateExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	dbType := effectiveDBType(u.execContext.DBType)
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(ctx, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	metaData, err := u.getTableMeta(ctx, u.execContext, u.parserCtx)
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
		if err != nil && !strings.Contains(err.Error(), "skip fast-path") {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}

		// If skip fast-path error, fallback to prepared statement
		if err != nil {
			log.Debugf("direct query not supported, falling back to prepared statement")
			stmt, prepErr := u.execContext.Conn.Prepare(selectSQL)
			if prepErr != nil {
				log.Errorf("prepare statement failed: %+v", prepErr)
				return nil, prepErr
			}

			if stmtQueryCtx, ok := stmt.(driver.StmtQueryContext); ok {
				rowsi, err = stmtQueryCtx.QueryContext(ctx, selectArgs)
			} else {
				dargs := make([]driver.Value, len(selectArgs))
				for i, arg := range selectArgs {
					dargs[i] = arg.Value
				}
				rowsi, err = stmt.Query(dargs)
			}

			if err != nil {
				stmt.Close()
				return nil, err
			}

			// Wrap rows with statement to close both together
			rowsi = util.NewRowsWithStmt(rowsi, stmt)
		}

		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	image, err := u.buildRecordImages(rowsi, metaData, types.SQLTypeUpdate, dbType)
	if err != nil {
		return nil, err
	}

	image.SQLType = u.parserCtx.SQLType
	image.TableMeta = metaData

	return image, nil
}

// afterImage build after image
func (u *updateExecutor) afterImage(ctx context.Context, beforeImage types.RecordImage) (*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}
	if len(beforeImage.Rows) == 0 {
		return types.NewEmptyRecordImage(beforeImage.TableMeta, types.SQLTypeUpdate), nil
	}

	dbType := effectiveDBType(u.execContext.DBType)
	metaData, err := u.getTableMeta(ctx, u.execContext, u.parserCtx)
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
		if err != nil && !strings.Contains(err.Error(), "skip fast-path") {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}

		// If skip fast-path error, fallback to prepared statement
		if err != nil {
			log.Debugf("direct query not supported, falling back to prepared statement")
			stmt, prepErr := u.execContext.Conn.Prepare(selectSQL)
			if prepErr != nil {
				log.Errorf("prepare statement failed: %+v", prepErr)
				return nil, prepErr
			}

			if stmtQueryCtx, ok := stmt.(driver.StmtQueryContext); ok {
				rowsi, err = stmtQueryCtx.QueryContext(ctx, selectArgs)
			} else {
				dargs := make([]driver.Value, len(selectArgs))
				for i, arg := range selectArgs {
					dargs[i] = arg.Value
				}
				rowsi, err = stmt.Query(dargs)
			}

			if err != nil {
				stmt.Close()
				return nil, err
			}

			// Wrap rows with statement to close both together
			rowsi = util.NewRowsWithStmt(rowsi, stmt)
		}

		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	afterImage, err := u.buildRecordImages(rowsi, metaData, types.SQLTypeUpdate, dbType)
	if err != nil {
		return nil, err
	}
	afterImage.SQLType = u.parserCtx.SQLType
	afterImage.TableMeta = metaData

	return afterImage, nil
}

func (u *updateExecutor) isAstStmtValid() bool {
	return u.parserCtx != nil && u.parserCtx.UpdateStmt != nil
}

// buildAfterImageSQL build the SQL to query after image data
func (u *updateExecutor) buildAfterImageSQL(beforeImage types.RecordImage, meta *types.TableMeta) (string, []driver.NamedValue) {
	if len(beforeImage.Rows) == 0 {
		return "", nil
	}
	dbType := effectiveDBType(u.execContext.DBType)
	sb := strings.Builder{}
	// todo: OnlyCareUpdateColumns should load from config first
	var selectFields string
	var separator = ","
	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, column := range beforeImage.Rows[0].Columns {
			columnName := column.ColumnName
			if dbType == types.DBTypePostgreSQL {
				columnName = `"` + strings.ReplaceAll(columnName, `"`, `""`) + `"`
			}
			selectFields += columnName + separator
		}
		selectFields = strings.TrimSuffix(selectFields, separator)
	} else {
		selectFields = "*"
	}
	sb.WriteString("SELECT " + selectFields + " FROM " + qualifiedTableName(u.execContext.TableMetaKey, util.AddEscape(meta.TableName, dbType), dbType) + " WHERE ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), dbType, maxInSize)
	sb.WriteString(" " + whereSQL + " ")
	return u.normalizeGeneratedSQL(sb.String(), dbType), u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName(), dbType)
}

// buildAfterImageSQL build the SQL to query before image data
func (u *updateExecutor) buildBeforeImageSQL(ctx context.Context, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	if !u.isAstStmtValid() {
		log.Errorf("invalid update stmt")
		return "", nil, fmt.Errorf("invalid update stmt")
	}

	dbType := effectiveDBType(u.execContext.DBType)
	updateStmt := u.parserCtx.UpdateStmt
	if dbType == types.DBTypePostgreSQL {
		var err error
		updateStmt, err = parser.ParsePostgreSQLUpdateForImage(updateStmt.Text())
		if err != nil {
			return "", nil, err
		}
	}
	fields := make([]*ast.SelectField, 0, len(updateStmt.List))
	metaData, err := u.getTableMeta(ctx, u.execContext, u.parserCtx)
	if err != nil {
		return "", nil, err
	}
	if dbType == types.DBTypeMySQL {
		if err := u.validatePrimaryKeyAssignments(metaData); err != nil {
			return "", nil, err
		}
	}

	if undo.UndoConfig.OnlyCareUpdateColumns {
		columnIdentity := func(name string) string {
			if dbType == types.DBTypePostgreSQL {
				return name
			}
			return strings.ToLower(name)
		}
		selectedColumns := make(map[string]struct{}, len(updateStmt.List))
		for _, column := range updateStmt.List {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: column.Column,
				},
			})
			selectedColumns[columnIdentity(column.Column.Name.O)] = struct{}{}
		}

		// select indexes columns
		for _, columnName := range metaData.GetPrimaryKeyOnlyName() {
			if _, ok := selectedColumns[columnIdentity(columnName)]; ok {
				continue
			}
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
	} else if dbType == types.DBTypePostgreSQL {
		fields = append(fields, &ast.SelectField{WildCard: &ast.WildCardField{}})
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

	from := updateStmt.TableRefs
	if dbType == types.DBTypePostgreSQL {
		from, err = postgresAuxiliaryQueryTable(from, u.execContext.TableMetaKey)
		if err != nil {
			return "", nil, err
		}
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           from,
		Where:          updateStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		OrderBy:        updateStmt.Order,
		Limit:          updateStmt.Limit,
		TableHints:     updateStmt.TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := bytes.NewByteBuffer([]byte{})
	restoreFlags := format.RestoreKeyWordUppercase
	if dbType == types.DBTypePostgreSQL {
		restoreFlags |= format.RestoreNameDoubleQuotes | format.RestoreStringSingleQuotes
	}
	if err := selStmt.Restore(format.NewRestoreCtx(restoreFlags, b)); err != nil {
		return "", nil, err
	}
	sql := string(b.Bytes())
	if dbType == types.DBTypePostgreSQL {
		// Literals are already restored as PostgreSQL SQL. MySQL charset
		// rewriting would also change matching text inside dollar strings.
		sql = strings.Replace(sql, "SELECT SQL_NO_CACHE ", "SELECT ", 1)
		sql = util.RewritePlaceholders(sql, dbType)
	} else {
		sql = u.normalizeGeneratedSQL(sql, dbType)
	}
	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	if dbType == types.DBTypePostgreSQL {
		return util.CompactPostgreSQLPlaceholders(sql, args)
	}

	return sql, u.buildSelectArgs(&selStmt, args), nil
}

// postgresAuxiliaryQueryTable changes only the auxiliary query's table identity. Other
// identifiers retain their original restore behavior, including case folding.
func postgresAuxiliaryQueryTable(from *ast.TableRefsClause, key *types.TableMetaKey) (*ast.TableRefsClause, error) {
	if from == nil || from.TableRefs == nil || key == nil {
		return nil, fmt.Errorf("postgres auxiliary query table identity is missing")
	}
	// Follow GetTableRef's path to the first table, copying each enclosing node.
	var qualify func(ast.ResultSetNode) (ast.ResultSetNode, error)
	qualify = func(node ast.ResultSetNode) (ast.ResultSetNode, error) {
		switch source := node.(type) {
		case *ast.Join:
			left, err := qualify(source.Left)
			if err != nil {
				return nil, err
			}
			qualified := *source
			qualified.Left = left
			return &qualified, nil
		case *ast.TableSource:
			table, err := qualify(source.Source)
			if err != nil {
				return nil, err
			}
			qualified := *source
			qualified.Source = table
			return &qualified, nil
		case *ast.TableName:
			qualified := *source
			qualified.Schema = model.NewCIStr(key.Schema)
			qualified.Name = model.NewCIStr(key.TableName)
			return &postgresAuxiliaryQueryTableName{TableName: &qualified}, nil
		default:
			return nil, fmt.Errorf("postgres auxiliary query source is not a table name")
		}
	}
	qualifiedSource, err := qualify(from.TableRefs.Left)
	if err != nil {
		return nil, err
	}
	qualifiedJoin := *from.TableRefs
	qualifiedJoin.Left = qualifiedSource
	qualifiedFrom := *from
	qualifiedFrom.TableRefs = &qualifiedJoin
	return &qualifiedFrom, nil
}

type postgresAuxiliaryQueryTableName struct {
	*ast.TableName
}

func (t *postgresAuxiliaryQueryTableName) Restore(ctx *format.RestoreCtx) error {
	tableCtx := *ctx
	tableCtx.Flags |= format.RestoreNameDoubleQuotes
	return t.TableName.Restore(&tableCtx)
}

func (u *updateExecutor) validatePrimaryKeyAssignments(metaData *types.TableMeta) error {
	for _, assignment := range u.parserCtx.UpdateStmt.List {
		for _, primaryKey := range metaData.GetPrimaryKeyOnlyName() {
			if !strings.EqualFold(assignment.Column.Name.O, primaryKey) {
				continue
			}

			columnExpr, selfAssignment := assignment.Expr.(*ast.ColumnNameExpr)
			if selfAssignment && strings.EqualFold(columnExpr.Name.Name.O, primaryKey) {
				continue
			}
			return fmt.Errorf("updating primary key column %q is not supported", assignment.Column.Name.O)
		}
	}
	return nil
}
