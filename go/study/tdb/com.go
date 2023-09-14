package tdb

import (
	"database/sql"
	"log"
)

func CreateTable(db *sql.DB) error {
	strSQL := `create table area(
		code integer not null primary key,
		name varchar(255),
		level int(4)
	)`
	_, err := db.Exec(strSQL)
	if err != nil {
		log.Printf("[E] create area table failed. err:%v", err)
		return err
	}
	return nil
}

func isTableExistSQLite3(db *sql.DB, name string) bool {
	strSQL := "SELECT count(*) FROM sqlite_master WHERE type='table' AND name = ?;"
	rows, err := db.Query(strSQL, name)
	if err != nil {
		log.Printf("[E] query mysql failed. sql:%v, err:%v", strSQL, err)
		return false
	}
	defer rows.Close()

	if rows.Next() {
		var count int
		err = rows.Scan(&count)
		if err != nil {
			log.Printf("[E] rows scan failed. err:%v", err)
			return false
		}
		if count == 1 {
			return true
		}
	}
	return false
}

func Query(db *sql.DB) {
	rows, err := db.Query("select * from area where level = ?;", 100)
	if err != nil {
		log.Printf("[E] query mysql failed. err:%v", err)
		return
	}

	defer rows.Close()
	colunms, err := rows.Columns()
	if err != nil {
		log.Printf("[E] rows columns failed. err:%v", err)
		return
	}

	log.Printf("[I] column:%v", colunms)

	for rows.Next() {
		var code, name string
		var level uint32
		err = rows.Scan(&code, &name, &level)
		if err != nil {
			log.Printf("[E] rows scan failed. err:%v", err)
			return
		}
		log.Printf("[I] code:%v, name:%v level:%v", code, name, level)
	}
}

func Insert(db *sql.DB) {
	result, err := db.Exec("insert into area(code, name, level) values(?, ?, ?);", "6666", "tiger", 100)
	if err != nil {
		log.Printf("[E] insert mysql failed. err:%v", err)
		return
	}

	n, err := result.RowsAffected()
	if err != nil {
		log.Printf("[E] get RowsAffected failed. err:%v", err)
		return
	}
	log.Printf("[I] RowsAffected:%v", n)
}

func Delete(db *sql.DB) {
	result, err := db.Exec("delete from area where level = ?;", 100)
	if err != nil {
		log.Printf("[E] delete mysql failed. err:%v", err)
		return
	}

	n, err := result.RowsAffected()
	if err != nil {
		log.Printf("[E] get RowsAffected failed. err:%v", err)
		return
	}
	log.Printf("[I] RowsAffected:%v", n)
}

func TxInsert(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[E] start tx failed. err:%v", err)
		return
	}

	defer func() {
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				log.Printf("[E] Rollback failed. err:%v", err)
			}
		}
	}()

	stmt, err := tx.Prepare("insert into area(code, name, level) values(?, ?, ?)")
	if err != nil {
		log.Printf("[E] tx Prepare failed. err:%v", err)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec("66661", "tiger1", 100)
	if err != nil {
		log.Printf("[E] insert mysql failed. err:%v", err)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("[E] get LastInsertId failed. err:%v", err)
		return
	}
	log.Printf("[I] LastInsertId:%v", id)

	tx.Commit()
}
