package at

import (
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/util/log"
	"strings"
)

type multiDeleteExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

func (m *multiDeleteExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	m.beforeHooks(ctx, m.execContext)
	defer func() {
		m.afterHooks(ctx, m.execContext)
	}()

	beforeImage, err := m.BeforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, m.execContext.Query, m.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := m.AfterImage(ctx)
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

//NewMultiDeleteExecutor get multiDelete executor
func NewMultiDeleteExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &multiDeleteExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

func (m *multiDeleteExecutor) BeforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	deletes := strings.Split(m.execContext.Query, ";")
	multiQuery, args, err := m.buildBeforeImageSQL(deletes, m.execContext.NamedValues)
	if err != nil {
		return nil, err
	}
	var (
		rowsi driver.Rows

		image   *types.RecordImage
		records []*types.RecordImage
	)

	queryerCtx, ok := m.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = m.execContext.Conn.(driver.Queryer)
	}
	if ok {
		for _, sql := range multiQuery {
			rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, sql, args)
			defer func() {
				if rowsi != nil {
					rowsi.Close()
				}
			}()
			if err != nil {
				log.Errorf("ctx driver query: %+v", err)
				return nil, err
			}
			tableName, _ := m.parserCtx.GetTableName()
			metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, m.execContext.DBName, tableName)
			if err != nil {
				log.Errorf("stmt query: %+v", err)
				return nil, err
			}
			image, err = m.buildRecordImages(rowsi, metaData)
			if err != nil {
				log.Errorf("record images : %+v", err)
				return nil, err
			}
			records = append(records, image)

			lockKey := m.buildLockKey(image, *metaData)
			m.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	return records, err

}

func (m *multiDeleteExecutor) AfterImage(ctx context.Context) ([]*types.RecordImage, error) {
	return nil, nil
}

func (m *multiDeleteExecutor) buildBeforeImageSQL(multiQuery []string, args []driver.NamedValue) ([]string, []driver.NamedValue, error) {
	var (
		err        error
		buf, param bytes.Buffer
		p          *types.ParseContext
		tableName  string
		tables     = make(map[string]multiDelete, len(multiQuery))
	)

	for _, query := range multiQuery {
		p, err = parser.DoParser(query)
		if err != nil {
			return nil, nil, err
		}
		tableName = p.DeleteStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O

		v, ok := tables[tableName]
		if ok && v.clear {
			continue
		}
		buf.WriteString("delete from ")
		buf.WriteString(tableName)
		if p.DeleteStmt.Where == nil {
			tables[tableName] = multiDelete{sql: buf.String(), clear: true}
			buf.Reset()
			continue
		} else {
			buf.WriteString(" where ")
		}

		_ = p.DeleteStmt.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &param))
		v, ok = tables[tableName]
		if ok {
			buf.Reset()
			buf.WriteString(v.sql)
			buf.WriteString(" or ")
		}

		buf.Write(param.Bytes())
		tables[tableName] = multiDelete{sql: buf.String()}

		buf.Reset()
		param.Reset()
	}

	var (
		items   = make([]string, 0, len(tables))
		values  = make([]driver.NamedValue, 0, len(tables))
		selStmt = ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{},
			From:           p.DeleteStmt.TableRefs,
			Where:          p.DeleteStmt.Where,
			Fields:         &ast.FieldList{Fields: []*ast.SelectField{{WildCard: &ast.WildCardField{}}}},
			OrderBy:        p.DeleteStmt.Order,
			TableHints:     p.DeleteStmt.TableHints,
			LockInfo:       &ast.SelectLockInfo{LockType: ast.SelectLockForUpdate},
		}
	)
	for _, table := range tables {
		p, _ = parser.DoParser(table.sql)

		selStmt.From = p.DeleteStmt.TableRefs
		selStmt.Where = p.DeleteStmt.Where
		_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &buf))
		items = append(items, buf.String())
		buf.Reset()
		if table.clear {
			values = append(values, m.buildSelectArgs(&selStmt, nil)...)
		} else {
			values = append(values, m.buildSelectArgs(&selStmt, args)...)
		}
	}
	return items, values, nil
}
