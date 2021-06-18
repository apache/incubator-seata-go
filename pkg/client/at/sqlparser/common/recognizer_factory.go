package common

import (
	"github.com/pingcap/parser/ast"
)
import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sqlparser"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sqlparser/mysql"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sqlparser/postgresql"
)

func GetDeleteRecognizer(query string, smt *ast.DeleteStmt, dbType string) sqlparser.ISQLDeleteRecognizer {
	if constant.POSTGRESQL == dbType {
		return postgresql.NewPostgresqlDeleteRecognizer(query, smt)
	}
	return mysql.NewMysqlDeleteRecognizer(query, smt)
}
func GetInsertRecognizer(query string, smt *ast.InsertStmt, dbType string) sqlparser.ISQLInsertRecognizer {
	if constant.POSTGRESQL == dbType {
		return postgresql.NewPostgresqlInsertRecognizer(query, smt)
	}
	return mysql.NewMysqlInsertRecognizer(query, smt)
}

func GetUpdateRecognizer(query string, smt *ast.UpdateStmt, dbType string) sqlparser.ISQLUpdateRecognizer {
	if constant.POSTGRESQL == dbType {
		return postgresql.NewPostgresqlUpdateRecognizer(query, smt)
	}
	return mysql.NewMysqlUpdateRecognizer(query, smt)
}

func GetSelectForUpdateRecognizer(query string, smt *ast.SelectStmt, dbType string) sqlparser.ISQLSelectRecognizer {
	if constant.POSTGRESQL == dbType {
		return postgresql.NewPostgresqlSelectForUpdateRecognizer(query, smt)
	}
	return mysql.NewMysqlSelectForUpdateRecognizer(query, smt)
}
