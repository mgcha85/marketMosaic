package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DB 전역 데이터베이스 연결
var DB *sql.DB

// InitDB 데이터베이스 초기화
func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return err
	}

	// 연결 테스트
	if err = DB.Ping(); err != nil {
		return err
	}

	// 스키마 초기화
	if err = initSchema(); err != nil {
		return err
	}

	log.Println("Database initialized successfully")
	return nil
}

// CloseDB 데이터베이스 연결 종료
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
