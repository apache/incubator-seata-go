package main

import "database/sql"
import sql2 "github.com/seata/seata-go/pkg/datasource/sql"

var (
	db *sql.DB
)

func initService() {
	var err error
	db, err = sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		panic("init service error")
	}
}
