package tdb

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func ConnectMySQL() *sql.DB {
	db, err := sql.Open("mysql", "cloud_setting_dev:PfRV3TeR8YJ9zHindfu4@tcp(rm-wz9u81o20727lu8j4.mysql.rds.aliyuncs.com:3306)/cloud_setting_dev")
	if err != nil {
		log.Printf("[E] open mysql failed. err:%v", err)
		return nil
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	return db
}

func MySQLTest() {
	db := ConnectMySQL()
	if db == nil {
		return
	}
	TxInsert(db)
	Query(db)
	Delete(db)
}
