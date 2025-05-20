package book

import (
	"context"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/joho/godotenv"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
)

func BenchmarkInsertBook(b *testing.B) {
	if err := godotenv.Load("../../../.env.test"); err != nil {
		log.Println(".env.test の読み込みに失敗:", err)
	}

	ctx := context.Background()
	db, err := database.Setup(os.Getenv("DATABASE_URL"))
	if err != nil {
		b.Fatalf("DB接続失敗: %v", err)
	}
	defer db.Close()

	if err := database.CreateTable(db); err != nil {
		b.Fatalf("テーブル作成エラー: %v", err)
	}

	const batchSize = 100

	b.Run("単一レコードのループ挿入", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			db.Exec("TRUNCATE TABLE books")
			b.StartTimer()

			for j := 0; j < batchSize; j++ {
				book := NewBook(strconv.Itoa(j), "タイトル", "", []string{"著者"}, "出版社", "2020", "説明", "http://example.com", "")
				if err := book.Insert(ctx, db); err != nil {
					b.Fatalf("単一挿入失敗: %v", err)
				}
			}
		}
	})

	b.Run("バルクインサート", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			db.Exec("TRUNCATE TABLE books")
			b.StartTimer()

			books := make([]*Book, batchSize)
			for j := 0; j < batchSize; j++ {
				books[j] = NewBook(strconv.Itoa(j), "タイトル", "", []string{"著者"}, "出版社", "2020", "説明", "http://example.com", "")
			}

			tx, txErr := db.BeginTx(ctx, nil)
			if txErr != nil {
				log.Fatalf("トランザクション開始エラー: %v", txErr)
			}
			defer tx.Rollback()

			if _, err := BulkInsert(ctx, tx, books); err != nil {
				b.Fatalf("バルク挿入失敗: %v", err)
			}
			if err := tx.Commit(); err != nil {
				b.Fatalf("コミットエラー: %v", err)
			}
		}
	})
}
