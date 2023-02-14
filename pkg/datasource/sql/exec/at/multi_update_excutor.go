package at

import (
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/util/bytes"
	"github.com/seata/seata-go/pkg/util/log"
	"strings"
)

// MultiUpdateExecutor execute multiple update SQL
type MultiUpdateExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

// NewMultiUpdateExecutor get new multi update executor
func NewMultiUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) *MultiUpdateExecutor {
	return &MultiUpdateExecutor{parserCtx: parserCtx, execContext: execContext, baseExecutor: baseExecutor{hooks: hooks}}
}

// ExecContext exec SQL, and generate before image and after image
func (u *MultiUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	u.beforeHooks(ctx, u.execContext)

	//single update sql handler
	if len(u.execContext.ParseContext.MultiStmt) == 1 {
		u.execContext.ParseContext.UpdateStmt = u.execContext.ParseContext.MultiStmt[0].UpdateStmt
		return NewUpdateExecutor(u.parserCtx, u.execContext, u.hooks).ExecContext(ctx, f)
	}

	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()

	beforeImages, err := u.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, u.execContext.Query, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImages, err := u.afterImage(beforeImages)
	if err != nil {
		return nil, err
	}

	if len(afterImages) != len(beforeImages) {
		return nil, fmt.Errorf("Before image size is not equaled to after image size, probably because you updated the primary keys.")
	}

	for i, afterImage := range afterImages {
		beforeImage := afterImages[i]
		if len(beforeImage.Rows) != len(afterImage.Rows) {
			return nil, fmt.Errorf("Before image size is not equaled to after image size, probably because you updated the primary keys.")
		}

		u.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
		u.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	}

	return res, nil
}

func (u *MultiUpdateExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	var updateStmts []*ast.UpdateStmt
	for _, v := range u.parserCtx.MultiStmt {
		updateStmts = append(updateStmts, v.UpdateStmt)
	}

	tableName := u.execContext.ParseContext.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := u.execContext.MetaDataMap[tableName]

	// use
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(updateStmts, u.execContext.Values, metaData)
	if err != nil {
		return nil, err
	}

	var rows driver.Rows
	queryerContext, ok := u.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = u.execContext.Conn.(driver.Queryer)
	}
	if ok {
		rows, err = util.CtxDriverQuery(ctx, queryerContext, queryer, selectSQL, util.ValueToNamedValue(selectArgs))
		defer func() {
			if rows != nil {
				rows.Close()
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

	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey(image, metaData)
	u.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	image.SQLType = u.parserCtx.SQLType

	return []*types.RecordImage{image}, nil
}

func (u *MultiUpdateExecutor) afterImage(beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	var beforeImage *types.RecordImage
	if len(beforeImages) > 0 {
		beforeImage = beforeImages[0]
	}

	tableName := u.execContext.ParseContext.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := u.execContext.MetaDataMap[tableName]
	selectSQL, selectArgs := u.buildAfterImageSQL(beforeImage, metaData)

	stmt, err := u.execContext.Conn.Prepare(selectSQL)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}

	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}

	image.SQLType = u.execContext.ParseContext.SQLType
	return []*types.RecordImage{image}, nil
}

// buildAfterImageSQL build the SQL to query after image data
func (u *MultiUpdateExecutor) buildAfterImageSQL(beforeImage *types.RecordImage, meta types.TableMeta) (string, []driver.Value) {
	if !u.isAstStmtValid() {
		return "", nil
	}

	sb := strings.Builder{}
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
		selectFields = strings.Join(meta.ColumnNames, separator)
	}

	sb.WriteString("SELECT " + selectFields + " FROM " + meta.TableName + " WHERE ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), "mysql", maxInSize)
	sb.WriteString(" " + whereSQL + " ")
	return sb.String(), util.NamedValueToValue(u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName()))
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *MultiUpdateExecutor) buildBeforeImageSQL(_ []*ast.UpdateStmt, args []driver.Value, meta types.TableMeta) (string, []driver.Value, error) {
	if !u.isAstStmtValid() {
		log.Errorf("invalid multi update stmt")
		return "", nil, fmt.Errorf("invalid muliti update stmt")
	}

	var newArgs []driver.Value
	var fields []*ast.SelectField
	fieldsExits := make(map[string]struct{})
	var whereCondition strings.Builder
	multiStmts := u.parserCtx.MultiStmt

	for _, multiStmt := range multiStmts {
		updateStmt := multiStmt.UpdateStmt
		if updateStmt.Limit != nil {
			return "", nil, fmt.Errorf("multi update SQL with limit condition is not support yet")
		}
		if updateStmt.Order != nil {
			return "", nil, fmt.Errorf("multi update SQL with orderBy condition is not support yet")
		}

		for _, column := range updateStmt.List {
			if _, exist := fieldsExits[column.Column.String()]; exist {
				continue
			}

			fieldsExits[column.Column.String()] = struct{}{}
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: column.Column,
				},
			})
		}

		if !undo.UndoConfig.OnlyCareUpdateColumns {
			fields = make([]*ast.SelectField, len(meta.Columns))
			for _, column := range meta.ColumnNames {
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{
						Name: &ast.ColumnName{
							Name: model.CIStr{
								O: column,
							},
						},
					},
				})
			}
		}

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
		newArgs = append(newArgs, util.NamedValueToValue(u.buildSelectArgs(&tmpSelectStmt, util.ValueToNamedValue(args)))...)

		in := bytes.NewByteBuffer([]byte{})
		_ = updateStmt.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, in))
		whereConditionStr := string(in.Bytes())

		if whereCondition.Len() > 0 {
			whereCondition.Write([]byte(" OR "))
		}
		whereCondition.Write([]byte(whereConditionStr))
	}

	// only just get the where condition
	fakeSql := "select * from t where " + whereCondition.String()
	fakeStmt, err := parser.New().ParseOneStmt(fakeSql, "", "")
	if err != nil {
		return "", nil, errors.Wrap(err, "multi update parse fake sql error")
	}
	fakeNode, ok := fakeStmt.Accept(&updateVisitor{})
	if !ok {
		return "", nil, errors.Wrap(err, "multi update accept update visitor error")
	}
	fakeSelectStmt, ok := fakeNode.(*ast.SelectStmt)
	if !ok {
		return "", nil, fmt.Errorf("multi update fake node is not select stmt")
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           multiStmts[0].UpdateStmt.TableRefs,
		Where:          fakeSelectStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		TableHints:     multiStmts[0].UpdateStmt.TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := bytes.NewByteBuffer([]byte{})
	_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by update sourceQuery, sql {}", sql)

	return sql, newArgs, nil
}

func (u *MultiUpdateExecutor) isAstStmtValid() bool {
	return u.parserCtx != nil && u.parserCtx.MultiStmt != nil && len(u.parserCtx.MultiStmt) > 0
}

type updateVisitor struct {
	stmt *ast.UpdateStmt
}

func (m *updateVisitor) Enter(n ast.Node) (node ast.Node, skipChildren bool) {
	return n, true
}

func (m *updateVisitor) Leave(n ast.Node) (node ast.Node, ok bool) {
	node = n
	return node, true
}
