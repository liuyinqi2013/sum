package tdb

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func connectSQLite3() *sql.DB {
	db, err := sql.Open("sqlite3", "./db")
	if err != nil {
		log.Printf("[E] open mysql failed. err:%v", err)
		return nil
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	return db
}

func SQLite3Test() {
	db := connectSQLite3()
	if db == nil {
		return
	}

	if !isTableExistSQLite3(db, "area") {
		err := CreateTable(db)
		if err != nil {
			log.Printf("[E] create are failed. err:%v", err)
			return
		}
	}

	TxInsert(db)
	Query(db)
	Delete(db)
}
