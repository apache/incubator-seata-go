package builder

import (
	"context"
	"database/sql/driver"
	"fmt"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/util/bytes"
	"github.com/seata/seata-go/pkg/util/log"
)

type MySQLDeleteUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func (u *MySQLDeleteUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *exec.ExecContext) (*types.RecordImage, error) {
	vals := execCtx.Values
	if vals == nil {
		for n, param := range execCtx.NamedValues {
			vals[n] = param.Value
		}
	}
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(execCtx.Query, vals)
	if err != nil {
		return nil, err
	}

	stmt, err := execCtx.Conn.Prepare(selectSQL)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}

	return u.buildRecordImages(rows, execCtx.MetaData)
}

func (u *MySQLDeleteUndoLogBuilder) AfterImage(types.RecordImages) (*types.RecordImages, error) {
	return nil, nil
}

// buildBeforeImageSQL build delete sql from delete sql
func (u *MySQLDeleteUndoLogBuilder) buildBeforeImageSQL(query string, args []driver.Value) (string, []driver.Value, error) {
	p, err := parser.DoParser(query)
	if err != nil {
		return "", nil, err
	}

	if p.DeleteStmt == nil {
		log.Errorf("invalid delete stmt")
		return "", nil, fmt.Errorf("invalid delete stmt")
	}

	fields := []*ast.SelectField{}
	fields = append(fields, &ast.SelectField{
		WildCard: &ast.WildCardField{},
	})

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           p.DeleteStmt.TableRefs,
		Where:          p.DeleteStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		OrderBy:        p.DeleteStmt.Order,
		Limit:          p.DeleteStmt.Limit,
		TableHints:     p.DeleteStmt.TableHints,
	}

	b := bytes.NewByteBuffer([]byte{})
	selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by delete sourceQuery, sql {}", sql)

	return sql, u.buildSelectArgs(&selStmt, args), nil
}

func (u *MySQLDeleteUndoLogBuilder) GetSQLType() types.SQLType {
	return types.SQLTypeDelete
}
