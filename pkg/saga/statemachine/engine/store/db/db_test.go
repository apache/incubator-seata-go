package db

import (
	"database/sql"
	"os"
	"sync"
)

var (
	oncePrepareDB sync.Once
	db            *sql.DB
)

func prepareDB() {
	oncePrepareDB.Do(func() {
		var err error
		db, err = sql.Open("sqlite3", ":memory:")
		query_, err := os.ReadFile("testdata/sql/saga/sqlite_init.sql")
		initScript := string(query_)
		if err != nil {
			panic(err)
		}
		if _, err := db.Exec(initScript); err != nil {
			panic(err)
		}
	})

}
