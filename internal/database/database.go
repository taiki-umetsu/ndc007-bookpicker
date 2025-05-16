package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func Setup(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	fmt.Println("DB接続OK")
	return db, nil
}

func CreateTable(db *sql.DB) error {
	schemaPath := "internal/database/schema.sql"

	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("DDLファイル読み込みエラー: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始エラー: %w", err)
	}

	if _, err := tx.Exec(string(schemaSQL)); err != nil {
		tx.Rollback()
		return fmt.Errorf("DDL実行エラー: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションコミットエラー: %w", err)
	}

	fmt.Println("テーブル作成OK")
	return nil
}
